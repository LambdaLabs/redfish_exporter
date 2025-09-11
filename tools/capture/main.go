package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Capture struct {
	client      *http.Client
	baseURL     string
	outputPath  string
	visited     map[string]bool
	visitedMu   sync.Mutex
	stack       []string
	maxDocs     int
	sleepMs     int
	skipPaths   []string
	outputFile  *os.File
	count       int
	startTime   time.Time
}

func main() {
	var (
		host      = flag.String("host", "", "BMC host or IP address (required)")
		username  = flag.String("user", "", "Username (required)")
		password  = flag.String("pass", "", "Password (required)")
		output    = flag.String("output", "", "Output directory name under testdata/ (required)")
		insecure  = flag.Bool("insecure", true, "Skip TLS certificate verification")
		timeout   = flag.Duration("timeout", 10*time.Second, "Per-request timeout")
		sleepMs   = flag.Int("sleep", 0, "Sleep between requests in milliseconds")
		maxDocs   = flag.Int("max", 0, "Maximum number of documents to fetch (0 = unlimited)")
		skipPaths = flag.String("skipPaths", "Actions", "Comma-separated list of path substrings to skip (default: Actions)")
	)
	flag.Parse()

	// Validate required flags
	if *host == "" || *username == "" || *password == "" || *output == "" {
		flag.Usage()
		log.Fatal("Required flags: -host, -user, -pass, -output")
	}

	// Build base URL
	baseURL := fmt.Sprintf("https://%s/redfish/v1", *host)
	if !strings.Contains(*host, ":") && !strings.HasPrefix(*host, "http") {
		// If no port specified and not a full URL, use default HTTPS
		baseURL = fmt.Sprintf("https://%s/redfish/v1", *host)
	} else if strings.HasPrefix(*host, "http") {
		baseURL = fmt.Sprintf("%s/redfish/v1", strings.TrimSuffix(*host, "/"))
	}

	// Create output directory - relative to project root
	// We're running from tools/capture, so go up two levels
	outputDir := filepath.Join("..", "mock-server", "testdata", *output)
	if err := os.MkdirAll(outputDir, 0750); err != nil { // Use 0750 for directory permissions
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Open output file
	outputPath := filepath.Join(outputDir, "capture.txt")
	outputFile, err := os.Create(outputPath) //nolint:gosec // Output path is controlled by developer
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer func() {
		if err := outputFile.Close(); err != nil {
			log.Printf("Failed to close output file: %v", err)
		}
	}()

	// Create HTTP client with basic auth
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: *insecure, //nolint:gosec // Insecure flag is explicit user choice for testing
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   *timeout,
	}

	// Parse skip paths
	var skipPathsList []string
	if *skipPaths != "" {
		for _, path := range strings.Split(*skipPaths, ",") {
			trimmed := strings.TrimSpace(path)
			if trimmed != "" {
				skipPathsList = append(skipPathsList, trimmed)
			}
		}
	}

	// Create capture instance
	c := &Capture{
		client:     client,
		baseURL:    baseURL,
		outputPath: outputPath,
		visited:    make(map[string]bool),
		stack:      []string{baseURL},
		maxDocs:    *maxDocs,
		sleepMs:    *sleepMs,
		skipPaths:  skipPathsList,
		outputFile: outputFile,
		startTime:  time.Now(),
	}

	// Set up basic auth
	c.client.Transport = &authTransport{
		username:  *username,
		password:  *password,
		transport: transport,
	}

	log.Printf("Connecting to %s...", *host)
	log.Printf("Output will be saved to %s", outputPath)

	// Start crawling
	c.crawl()

	duration := time.Since(c.startTime)
	log.Printf("Captured %d endpoints in %v", c.count, duration)
	log.Printf("Saved to %s", outputPath)
}

// authTransport adds basic auth to requests
type authTransport struct {
	username  string
	password  string
	transport http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(t.username, t.password)
	req.Header.Set("Accept", "application/json")
	return t.transport.RoundTrip(req)
}

func (c *Capture) crawl() {
	for len(c.stack) > 0 {
		// Check max docs limit
		if c.maxDocs > 0 && c.count >= c.maxDocs {
			log.Printf("Reached maximum document limit (%d)", c.maxDocs)
			break
		}

		// Pop URL from stack (DFS)
		url := c.stack[len(c.stack)-1]
		c.stack = c.stack[:len(c.stack)-1]

		// Check if already visited
		c.visitedMu.Lock()
		if c.visited[url] {
			c.visitedMu.Unlock()
			continue
		}
		c.visited[url] = true
		c.visitedMu.Unlock()

		// Skip paths based on skipPaths list
		if c.shouldSkipURL(url) {
			continue
		}

		// Ensure same host
		if !c.sameHost(url) {
			continue
		}

		// Fetch the endpoint
		c.count++
		log.Printf("[%d/∞] Fetching %s", c.count, url)

		doc, err := c.fetchJSON(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "# ERROR fetching %s: %v\n", url, err)
			continue
		}

		// Write to output file
		if _, err := fmt.Fprintf(c.outputFile, "# %s\n", url); err != nil {
			fmt.Fprintf(os.Stderr, "# ERROR writing URL comment: %v\n", err)
		}
		encoder := json.NewEncoder(c.outputFile)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(doc); err != nil {
			fmt.Fprintf(os.Stderr, "# ERROR encoding JSON for %s: %v\n", url, err)
			continue
		}

		// Extract and queue new links
		links := c.extractODataLinks(doc)
		for _, link := range links {
			absURL := c.normalizeLink(link, url)
			
			c.visitedMu.Lock()
			alreadyVisited := c.visited[absURL]
			c.visitedMu.Unlock()
			
			if !alreadyVisited && c.sameHost(absURL) && !c.shouldSkipURL(absURL) {
				c.stack = append(c.stack, absURL)
			}
		}

		// Sleep between requests if configured
		if c.sleepMs > 0 {
			time.Sleep(time.Duration(c.sleepMs) * time.Millisecond)
		}
	}
}

func (c *Capture) fetchJSON(urlStr string) (map[string]interface{}, error) {
	resp, err := c.client.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Log but don't fail - response body close errors are usually not critical
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var doc map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}

	return doc, nil
}

func (c *Capture) extractODataLinks(obj interface{}) []string {
	var links []string
	c.extractLinks(obj, &links)
	return links
}

func (c *Capture) extractLinks(obj interface{}, links *[]string) {
	switch v := obj.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if key == "@odata.id" {
				if str, ok := value.(string); ok {
					*links = append(*links, str)
				}
			} else {
				c.extractLinks(value, links)
			}
		}
	case []interface{}:
		for _, item := range v {
			c.extractLinks(item, links)
		}
	}
}

func (c *Capture) normalizeLink(link, baseURL string) string {
	// If it's already an absolute URL, return as-is
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		return link
	}

	// Parse the base URL to get scheme and host
	base, err := url.Parse(baseURL)
	if err != nil {
		return link
	}

	// Resolve the relative link
	linkURL, err := url.Parse(link)
	if err != nil {
		return link
	}

	return base.ResolveReference(linkURL).String()
}

func (c *Capture) sameHost(urlStr string) bool {
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return false
	}
	
	checkURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	
	return baseURL.Host == checkURL.Host
}

func (c *Capture) shouldSkipURL(urlStr string) bool {
	for _, skipPath := range c.skipPaths {
		if strings.Contains(urlStr, "/"+skipPath+"/") {
			return true
		}
	}
	return false
}