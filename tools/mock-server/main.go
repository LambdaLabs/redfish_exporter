package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

// MockRedfishServer provides a mock Redfish API server for testing
type MockRedfishServer struct {
	endpoints   map[string]json.RawMessage
	mu          sync.RWMutex
	config      *Config
	accessLog   *log.Logger
	requestLog  []RequestLog
	reqLogMutex sync.RWMutex
	authConfig  *AuthConfig
	sessions    map[string]*Session
	sessionMu   sync.RWMutex
}

// Config holds configuration for the mock server
type Config struct {
	DataFile        string            `json:"data_file"`
	Port            string            `json:"port"`
	HTTPSPort       string            `json:"https_port"`
	ResponseDelay   time.Duration     `json:"response_delay"`
	FailureRate     float64           `json:"failure_rate"`
	CustomResponses map[string]string `json:"custom_responses"`
	EnableLogging   bool              `json:"enable_logging"`
	LogFile         string            `json:"log_file"`
	AuthConfigFile  string            `json:"auth_config_file"`
	CertFile        string            `json:"cert_file"`
	KeyFile         string            `json:"key_file"`
	EnableHTTPS     bool              `json:"enable_https"`
	EnableHTTP      bool              `json:"enable_http"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Users []User `yaml:"users"`
}

// User represents a user account
type User struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Session represents an active session
type Session struct {
	ID       string
	Username string
	Token    string
	Created  time.Time
	LastUsed time.Time
}

// RequestLog tracks incoming requests for debugging
type RequestLog struct {
	Timestamp time.Time
	Method    string
	Path      string
	Headers   map[string]string
	Found     bool
}

func NewMockRedfishServer(config *Config) *MockRedfishServer {
	server := &MockRedfishServer{
		endpoints:  make(map[string]json.RawMessage),
		config:     config,
		requestLog: make([]RequestLog, 0),
		sessions:   make(map[string]*Session),
	}

	if config.EnableLogging && config.LogFile != "" {
		logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644) //nolint:gosec // Log files need to be readable
		if err == nil {
			server.accessLog = log.New(logFile, "[ACCESS] ", log.LstdFlags)
		}
	}

	// Load auth config
	if config.AuthConfigFile != "" {
		if err := server.loadAuthConfig(config.AuthConfigFile); err != nil {
			log.Printf("Warning: failed to load auth config, using defaults: %v", err)
			server.useDefaultAuth()
		}
	} else {
		server.useDefaultAuth()
	}

	return server
}

func (s *MockRedfishServer) loadAuthConfig(filename string) error {
	data, err := os.ReadFile(filename) //nolint:gosec // Test data files are controlled by developer
	if err != nil {
		return err
	}

	var authConfig AuthConfig
	if err := yaml.Unmarshal(data, &authConfig); err != nil {
		return err
	}

	s.authConfig = &authConfig
	log.Printf("Loaded %d users from auth config", len(authConfig.Users))
	return nil
}

func (s *MockRedfishServer) useDefaultAuth() {
	s.authConfig = &AuthConfig{
		Users: []User{
			{Username: "admin", Password: "password"},
		},
	}
	log.Println("Using default authentication (admin/password)")
}

// LoadFromFile parses a capture file with format:
// # URL
// JSON content
func (s *MockRedfishServer) LoadFromFile(filename string) error {
	file, err := os.Open(filename) //nolint:gosec // Test data files are controlled by developer
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	// Check file size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	log.Printf("Loading file: %s (%.2f MB)", filename, float64(stat.Size())/(1024*1024))

	reader := bufio.NewReader(file)
	var currentURL string
	var jsonLines []string
	lineNum := 0
	bracketCount := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("error reading file at line %d: %w", lineNum, err)
		}
		if err == io.EOF && line == "" {
			break
		}
		lineNum++

		line = strings.TrimRight(line, "\n\r")

		// Check if this is a URL comment
		if strings.HasPrefix(line, "# http") {
			// Save previous endpoint if we have one
			if currentURL != "" && len(jsonLines) > 0 {
				if err := s.saveEndpoint(currentURL, jsonLines); err != nil {
					log.Printf("Warning: failed to save endpoint %s: %v", currentURL, err)
				}
			}

			// Extract path from URL
			currentURL = s.extractPath(line)
			jsonLines = []string{}
			bracketCount = 0
		} else if currentURL != "" {
			// Accumulate JSON lines
			jsonLines = append(jsonLines, line)

			// Track brackets to know when JSON object is complete
			bracketCount += strings.Count(line, "{") - strings.Count(line, "}")
		}

		if lineNum%10000 == 0 {
			log.Printf("Processed %d lines...", lineNum)
		}
	}

	// Save last endpoint
	if currentURL != "" && len(jsonLines) > 0 {
		if err := s.saveEndpoint(currentURL, jsonLines); err != nil {
			log.Printf("Warning: failed to save endpoint %s: %v", currentURL, err)
		}
	}

	log.Printf("Successfully loaded %d endpoints from %s", len(s.endpoints), filename)
	return nil
}

func (s *MockRedfishServer) extractPath(urlLine string) string {
	// Remove comment prefix and extract path
	url := strings.TrimPrefix(urlLine, "# ")
	url = strings.TrimSpace(url)

	// Find /redfish in the URL
	idx := strings.Index(url, "/redfish")
	if idx >= 0 {
		return url[idx:]
	}

	// If no /redfish found, try to extract path after host
	parts := strings.Split(url, "/")
	if len(parts) >= 4 { // https://host/path...
		return "/" + strings.Join(parts[3:], "/")
	}

	return url
}

func (s *MockRedfishServer) saveEndpoint(url string, jsonLines []string) error {
	jsonStr := strings.Join(jsonLines, "\n")
	jsonStr = strings.TrimSpace(jsonStr)

	if jsonStr == "" {
		return nil
	}

	// Validate JSON
	var temp interface{}
	if err := json.Unmarshal([]byte(jsonStr), &temp); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.endpoints[url] = json.RawMessage(jsonStr)

	return nil
}

func (s *MockRedfishServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log request
	reqLog := RequestLog{
		Timestamp: time.Now(),
		Method:    r.Method,
		Path:      r.URL.Path,
		Headers:   make(map[string]string),
		Found:     false,
	}

	// Capture relevant headers
	for key := range r.Header {
		if strings.HasPrefix(key, "X-") || key == "Authorization" {
			reqLog.Headers[key] = r.Header.Get(key)
		}
	}

	// Check authentication for non-public endpoints
	if !s.isPublicEndpoint(r.URL.Path) && !s.isAuthenticated(r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Redfish API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		s.logRequest(reqLog)
		return
	}

	// Apply response delay if configured
	if s.config.ResponseDelay > 0 {
		time.Sleep(s.config.ResponseDelay)
	}

	// Handle special endpoints
	if s.handleSpecialEndpoints(w, r) {
		reqLog.Found = true
		s.logRequest(reqLog)
		return
	}

	// Look up the endpoint (handle trailing slash)
	path := r.URL.Path
	s.mu.RLock()
	data, exists := s.endpoints[path]
	if !exists && strings.HasSuffix(path, "/") {
		// Try without trailing slash
		path = strings.TrimSuffix(path, "/")
		data, exists = s.endpoints[path]
	}
	s.mu.RUnlock()

	if !exists {
		// Log miss and suggest alternatives
		s.logMiss(r.URL.Path)
		http.Error(w, fmt.Sprintf("Endpoint not found: %s", r.URL.Path), http.StatusNotFound)
		s.logRequest(reqLog)
		return
	}

	reqLog.Found = true

	// Return the JSON data
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Printf("Failed to write response: %v", err)
	}

	s.logRequest(reqLog)
}

func (s *MockRedfishServer) isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/redfish",
		"/redfish/",
		"/redfish/v1",
		"/redfish/v1/",
		"/redfish/v1/SessionService/Sessions",
	}
	for _, p := range publicPaths {
		if path == p {
			return true
		}
	}
	return false
}

func (s *MockRedfishServer) isAuthenticated(r *http.Request) bool {
	// Check X-Auth-Token header
	token := r.Header.Get("X-Auth-Token")
	if token != "" {
		s.sessionMu.RLock()
		session, exists := s.sessions[token]
		s.sessionMu.RUnlock()

		if exists {
			// Update last used time
			s.sessionMu.Lock()
			session.LastUsed = time.Now()
			s.sessionMu.Unlock()
			return true
		}
	}

	// Check Basic Auth
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Basic ") {
		credentials := strings.TrimPrefix(auth, "Basic ")
		decoded, err := base64.StdEncoding.DecodeString(credentials)
		if err != nil {
			return false
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return false
		}

		username, password := parts[0], parts[1]
		return s.validateCredentials(username, password)
	}

	return false
}

func (s *MockRedfishServer) validateCredentials(username, password string) bool {
	for _, user := range s.authConfig.Users {
		if subtle.ConstantTimeCompare([]byte(user.Username), []byte(username)) == 1 &&
			subtle.ConstantTimeCompare([]byte(user.Password), []byte(password)) == 1 {
			return true
		}
	}
	return false
}

func (s *MockRedfishServer) handleSpecialEndpoints(w http.ResponseWriter, r *http.Request) bool {
	// Handle authentication
	if r.URL.Path == "/redfish/v1/SessionService/Sessions" && r.Method == "POST" {
		// Parse credentials from request body
		var loginReq struct {
			UserName string `json:"UserName"`
			Password string `json:"Password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return true
		}

		// Validate credentials
		if !s.validateCredentials(loginReq.UserName, loginReq.Password) {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return true
		}

		// Create session
		sessionID := fmt.Sprintf("%d", time.Now().UnixNano())
		token := "token-" + sessionID
		session := &Session{
			ID:       sessionID,
			Username: loginReq.UserName,
			Token:    token,
			Created:  time.Now(),
			LastUsed: time.Now(),
		}

		s.sessionMu.Lock()
		s.sessions[token] = session
		s.sessionMu.Unlock()

		response := map[string]interface{}{
			"@odata.id":   "/redfish/v1/SessionService/Sessions/" + sessionID,
			"@odata.type": "#Session.v1_0_0.Session",
			"Id":          sessionID,
			"Name":        "User Session",
			"UserName":    loginReq.UserName,
			"Created":     session.Created.Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Auth-Token", token)
		w.Header().Set("Location", "/redfish/v1/SessionService/Sessions/"+sessionID)
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode login response: %v", err)
		}
		return true
	}

	// Handle logout
	if strings.HasPrefix(r.URL.Path, "/redfish/v1/SessionService/Sessions/") && r.Method == "DELETE" {
		// Remove session if it exists
		token := r.Header.Get("X-Auth-Token")
		if token != "" {
			s.sessionMu.Lock()
			delete(s.sessions, token)
			s.sessionMu.Unlock()
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	}

	return false
}

func (s *MockRedfishServer) logRequest(reqLog RequestLog) {
	s.reqLogMutex.Lock()
	s.requestLog = append(s.requestLog, reqLog)
	if len(s.requestLog) > 1000 { // Keep last 1000 requests
		s.requestLog = s.requestLog[1:]
	}
	s.reqLogMutex.Unlock()

	status := "FOUND"
	if !reqLog.Found {
		status = "NOT_FOUND"
	}

	if s.accessLog != nil {
		s.accessLog.Printf("%s %s %s", reqLog.Method, reqLog.Path, status)
	}

	log.Printf("[%s] %s %s", status, reqLog.Method, reqLog.Path)
}

func (s *MockRedfishServer) logMiss(path string) {
	log.Printf("Endpoint not found: %s", path)

	// Suggest similar endpoints
	s.mu.RLock()
	defer s.mu.RUnlock()

	suggestions := []string{}
	pathParts := strings.Split(path, "/")

	for endpoint := range s.endpoints {
		endpointParts := strings.Split(endpoint, "/")

		// Check for similar paths
		if len(pathParts) > 2 && len(endpointParts) > 2 {
			if pathParts[2] == endpointParts[2] { // Same main resource
				suggestions = append(suggestions, endpoint)
				if len(suggestions) >= 5 {
					break
				}
			}
		}
	}

	if len(suggestions) > 0 {
		log.Println("Similar endpoints available:")
		for _, s := range suggestions {
			log.Printf("  - %s", s)
		}
	}
}

// GetRequestLog returns recent request logs for debugging
func (s *MockRedfishServer) GetRequestLog() []RequestLog {
	s.reqLogMutex.RLock()
	defer s.reqLogMutex.RUnlock()

	logs := make([]RequestLog, len(s.requestLog))
	copy(logs, s.requestLog)
	return logs
}

// DebugHandler provides debug information about the mock server
func (s *MockRedfishServer) DebugHandler(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	endpoints := make([]string, 0, len(s.endpoints))
	for endpoint := range s.endpoints {
		endpoints = append(endpoints, endpoint)
	}
	s.mu.RUnlock()

	response := map[string]interface{}{
		"total_endpoints": len(endpoints),
		"endpoints":       endpoints,
		"recent_requests": s.GetRequestLog(),
		"config":          s.config,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode status response: %v", err)
	}
}

func main() {
	var (
		dataFile    = flag.String("file", "", "Path to the capture file containing Redfish responses")
		dataDir     = flag.String("dir", "", "Directory containing multiple capture files")
		system      = flag.String("system", "", "System name to load from testdata/<system>/")
		port        = flag.String("port", "8080", "Port to listen on (HTTP)")
		httpsPort   = flag.String("https-port", "8443", "Port to listen on (HTTPS)")
		delay       = flag.Duration("delay", 0, "Response delay to simulate network latency")
		logFile     = flag.String("log", "", "Path to access log file")
		authConfig  = flag.String("auth", "", "Path to auth config file (default: use admin/password)")
		certFile    = flag.String("cert", "", "Path to TLS certificate file (auto-generated if not provided)")
		keyFile     = flag.String("key", "", "Path to TLS key file (auto-generated if not provided)")
		enableHTTP  = flag.Bool("http", true, "Enable HTTP server")
		enableHTTPS = flag.Bool("https", true, "Enable HTTPS server")
		debug       = flag.Bool("debug", false, "Enable debug endpoints")
	)
	flag.Parse()

	// Determine data source
	var dataSource string
	if *system != "" {
		// Load from testdata/<system>/ directory
		testdataPath := filepath.Join("tools", "mock-server", "testdata", *system)
		files, err := filepath.Glob(filepath.Join(testdataPath, "*.txt"))
		if err != nil || len(files) == 0 {
			log.Fatalf("No data files found in %s", testdataPath)
		}
		dataSource = files[0] // Use first .txt file found
		log.Printf("Loading system '%s' from %s", *system, dataSource)
	} else if *dataFile != "" {
		dataSource = *dataFile
	} else if *dataDir == "" {
		log.Fatal("Please specify either -file, -dir, or -system flag")
	}

	config := &Config{
		Port:           *port,
		HTTPSPort:      *httpsPort,
		ResponseDelay:  *delay,
		EnableLogging:  *logFile != "",
		LogFile:        *logFile,
		AuthConfigFile: *authConfig,
		CertFile:       *certFile,
		KeyFile:        *keyFile,
		EnableHTTP:     *enableHTTP,
		EnableHTTPS:    *enableHTTPS,
	}

	server := NewMockRedfishServer(config)

	// Load data
	if dataSource != "" {
		if err := server.LoadFromFile(dataSource); err != nil {
			log.Fatalf("Failed to load data from file: %v", err)
		}
	}

	if *dataDir != "" {
		files, err := filepath.Glob(filepath.Join(*dataDir, "*.txt"))
		if err != nil {
			log.Fatalf("Failed to find files in directory: %v", err)
		}
		for _, file := range files {
			log.Printf("Loading %s...", file)
			if err := server.LoadFromFile(file); err != nil {
				log.Printf("Warning: failed to load %s: %v", file, err)
			}
		}
	}

	// Set up HTTP server
	mux := http.NewServeMux()
	mux.Handle("/redfish/", server)

	if *debug {
		mux.HandleFunc("/debug", server.DebugHandler)
		log.Println("Debug endpoint available at /debug")
	}

	// Generate or load certificates for HTTPS
	var tlsConfig *tls.Config
	if config.EnableHTTPS {
		var err error
		tlsConfig, err = getTLSConfig(config)
		if err != nil {
			log.Fatalf("Failed to set up TLS: %v", err)
		}
	}

	// Start servers
	var wg sync.WaitGroup

	if config.EnableHTTP {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr := ":" + config.Port
			log.Printf("Starting HTTP mock Redfish server on %s", addr)
			if *debug {
				log.Printf("Debug info available at: http://localhost:%s/debug", config.Port)
			}
			log.Fatal(http.ListenAndServe(addr, mux)) //nolint:gosec // Mock server for testing only
		}()
	}

	if config.EnableHTTPS {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr := ":" + config.HTTPSPort
			log.Printf("Starting HTTPS mock Redfish server on %s", addr)
			if *debug {
				log.Printf("Debug info available at: https://localhost:%s/debug", config.HTTPSPort)
			}
			server := &http.Server{
				Addr:              addr,
				Handler:           mux,
				TLSConfig:         tlsConfig,
				ReadHeaderTimeout: 10 * time.Second, // Prevent Slowloris attacks
			}
			log.Fatal(server.ListenAndServeTLS("", ""))
		}()
	}

	log.Printf("Configure redfish_exporter to scrape:")
	if config.EnableHTTP {
		log.Printf("  HTTP:  localhost:%s", config.Port)
	}
	if config.EnableHTTPS {
		log.Printf("  HTTPS: localhost:%s (with insecure TLS)", config.HTTPSPort)
	}

	wg.Wait()
}

// getTLSConfig returns TLS configuration with certificate
func getTLSConfig(config *Config) (*tls.Config, error) {
	var cert tls.Certificate
	var err error

	if config.CertFile != "" && config.KeyFile != "" {
		// Load existing certificate
		cert, err = tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate: %w", err)
		}
		log.Printf("Using existing certificate from %s", config.CertFile)
	} else {
		// Generate self-signed certificate
		cert, err = generateSelfSignedCert()
		if err != nil {
			return nil, fmt.Errorf("failed to generate certificate: %w", err)
		}
		log.Println("Generated self-signed certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12, // Set minimum TLS version for security
	}, nil
}

// generateSelfSignedCert creates a self-signed certificate for testing
func generateSelfSignedCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Mock Redfish Server"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:              []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Save certificate and key to files for reuse
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	// Optionally save to files
	certPath := "tools/mock-server/server.crt"
	keyPath := "tools/mock-server/server.key"

	if err := os.WriteFile(certPath, certPEM, 0600); err != nil { // Use 0600 for certificate files
		log.Printf("Warning: failed to save certificate to %s: %v", certPath, err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		log.Printf("Warning: failed to save key to %s: %v", keyPath, err)
	}

	return tls.X509KeyPair(certPEM, keyPEM)
}
