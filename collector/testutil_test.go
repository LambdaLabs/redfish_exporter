package collector

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/LambdaLabs/redfish_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stmcginnis/gofish"
	"github.com/stretchr/testify/require"
)

// testRedfishServer provides a test HTTP server that mimics Redfish API responses
type testRedfishServer struct {
	*httptest.Server
	t        *testing.T
	mux      *http.ServeMux
	requests []string // Track requests for debugging
}

// newTestRedfishServer creates a new test Redfish server for testing
func newTestRedfishServer(t *testing.T) *testRedfishServer {
	t.Helper()

	trs := &testRedfishServer{
		t:        t,
		mux:      http.NewServeMux(),
		requests: make([]string, 0),
	}

	trs.Server = httptest.NewServer(trs.mux)
	t.Cleanup(trs.Close)

	// Set up default routes
	trs.setupDefaultRoutes()

	return trs
}

// setupDefaultRoutes registers the default Redfish API routes
func (m *testRedfishServer) setupDefaultRoutes() {
	// Service root
	m.addRouteFromFixture("/redfish/v1/", "service_root.json")

	// Systems collection
	m.addRouteFromFixture("/redfish/v1/Systems", "systems_collection.json")
}

// makeJSONHandler creates a handler that returns a JSON response
func (m *testRedfishServer) makeJSONHandler(response any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.requests = append(m.requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			m.t.Errorf("Failed to encode response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// addRoute registers a new route with the given response
func (m *testRedfishServer) addRoute(path string, response any) {
	m.mux.HandleFunc(path, m.makeJSONHandler(response))
}

// addRouteFromFixture loads a fixture file and registers it as a route
func (m *testRedfishServer) addRouteFromFixture(path string, fixtureFile string) {
	data := loadTestData(m.t, fixtureFile)
	m.addRoute(path, data)
}

// loadFixture loads a JSON fixture and returns it
func (m *testRedfishServer) loadFixture(fixtureFile string) map[string]interface{} {
	return loadTestData(m.t, fixtureFile)
}

// setupSystemWithProcessor sets up a basic system with a processor using fixtures
func (m *testRedfishServer) setupSystemWithProcessor(systemID, processorID string) {
	systemPath := "/redfish/v1/Systems/" + systemID
	processorPath := systemPath + "/Processors/" + processorID

	// Load and modify system fixture
	system := m.loadFixture("system1.json")
	system["@odata.id"] = systemPath
	system["Id"] = systemID
	system["Processors"] = map[string]string{"@odata.id": systemPath + "/Processors"}
	m.addRoute(systemPath, system)

	// Create processor collection
	m.addRoute(systemPath+"/Processors", map[string]interface{}{
		"@odata.type": "#ProcessorCollection.ProcessorCollection",
		"Members": []map[string]string{
			{"@odata.id": processorPath},
		},
	})
}

// collectSystemMetrics runs the system collector and returns metrics as a map
func collectSystemMetrics(t *testing.T, client *gofish.APIClient) map[string]float64 {
	t.Helper()

	// Create a test logger that discards output
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewSystemCollector(t.Name(), client, logger, config.DefaultSystemCollector)
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 100)
	collector.Collect(ch)
	close(ch)

	metrics := make(map[string]float64)
	for metric := range ch {
		dto := &dto.Metric{}
		err := metric.Write(dto)
		require.NoError(t, err)

		desc := metric.Desc().String()
		if gauge := dto.GetGauge(); gauge != nil {
			// Extract processor-related metrics from system collector output
			switch {
			case contains(desc, "processor_pcie_errors_l0_to_recovery_count"):
				metrics["pcie_l0_recovery"] = gauge.GetValue()
			case contains(desc, "processor_pcie_errors_correctable_count"):
				metrics["pcie_correctable"] = gauge.GetValue()
			case contains(desc, "processor_pcie_errors_fatal_count"):
				metrics["pcie_fatal"] = gauge.GetValue()
			case contains(desc, "processor_cache_lifetime_uncorrectable_ecc_error_count"):
				metrics["cache_uncorrectable"] = gauge.GetValue()
			case contains(desc, "processor_cache_lifetime_correctable_ecc_error_count"):
				metrics["cache_correctable"] = gauge.GetValue()
			}
		}
	}

	return metrics
}

// contains is a helper function for string matching
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// connectToTestServer creates a gofish client connected to the test server
func connectToTestServer(t *testing.T, server *httptest.Server) *gofish.APIClient {
	t.Helper()

	config := gofish.ClientConfig{
		Endpoint: server.URL,
		Username: "",
		Password: "",
		Insecure: true,
	}

	client, err := gofish.Connect(config)
	require.NoError(t, err, "Failed to connect to test server")

	return client
}

// setupTestServerClient spawns a httptest server which serves contents from a given basepath.
// It also sets up and connects a Gofish client to the httptest server.
// Both the server and client are returned for consumption in tests.
// A cleanup function is registered to close the server and logout the client.
func setupTestServerClient(t *testing.T, basepath string) (*TestServer, *gofish.APIClient) {
	t.Helper()
	osRoot, err := os.OpenRoot(basepath)
	require.NoError(t, err)

	server := newTestServer(t, osRoot, jsonContentTypeMiddleware)

	client := connectToTestServer(t, server.Server)
	t.Cleanup(func() {
		server.Close()
		client.Logout()
	})
	return server, client
}

// TestServer wraps httptest.Server to serve files from an os.Root
type TestServer struct {
	*httptest.Server
}

// newTestServer creates a test server that serves files from the given os.Root and with the given test server middlewares.
func newTestServer(t *testing.T, root *os.Root, middleware ...testMiddleware) *TestServer {
	t.Helper()
	// Create the file server handler
	handler := dirIndexFileServer(root.FS())

	// Apply middleware
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}

	// Create and return the test server
	return &TestServer{
		Server: httptest.NewServer(handler),
	}
}

// dirIndexFileServer is an HTTP handler that is similar to [http.FileServer], in that contents are read from a filesystem.
// It allows for a filesystem tree representative of a Redfish system.
// It differs from [http.FileServer] in that placing an 'index.json' file in a directory will serve that contents by default.
// That is, a Redfish request path like /redfish/v1 will look for and respond with contents from redfish/v1/index.json.
// This works well for Redfish APIs where paths are nested
// e.g. redfish/v1/Systems/HGX_Baseboard_0/Processors may serve a collection
// whereas redfish/v1/Systems/HGX_Baseboard_0/Processors/CPU_0 is a 'real' Redfish CPU
// in both cases, an index.json represnts whatever the system should return under each path.
func dirIndexFileServer(fsys fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		md5URL := md5.Sum([]byte(r.URL.String()))
		md5String := hex.EncodeToString(md5URL[:])
		hashedFile, err := fsys.Open(fmt.Sprintf("hashedresponses/%s.json", md5String))
		if err != nil {
			// Probably no hashed response file for this request, so traverse the fs
			raw := path.Clean(r.URL.EscapedPath())
			decoded, err := url.PathUnescape(raw)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			p := strings.TrimPrefix(decoded, "/")

			info, err := fs.Stat(fsys, p)
			if err != nil {
				http.NotFound(w, r)
				return
			}

			servePath := p
			if info.IsDir() {
				servePath = path.Join(p, "index.json")
				if _, derr := fs.Stat(fsys, servePath); derr != nil {
					http.NotFound(w, r)
					return
				}
			}

			b, err := fs.ReadFile(fsys, servePath)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(b)
			return
		}

		// Otherwise, we have a hashed file to respond with
		b, err := io.ReadAll(hashedFile)
		if err != nil {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	})
}

type testMiddleware func(http.Handler) http.Handler

// DelayMiddleware sleeps for some duration prior to response.
func delayMiddleware(delay time.Duration) testMiddleware { //nolint:unused
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(delay)
			next.ServeHTTP(w, r)
		})
	}
}

// JSONContentTypeMiddleware ensures all responses have JSON content type.
func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

type testWriter struct {
	t *testing.T
}

func (w testWriter) Write(p []byte) (n int, err error) {
	w.t.Log(strings.TrimSpace(string(p)))
	return len(p), nil
}

func NewTestLogger(t *testing.T, loglevel slog.Level) *slog.Logger {
	t.Helper()
	opts := &slog.HandlerOptions{
		Level: loglevel,
	}
	return slog.New(slog.NewJSONHandler(testWriter{t}, opts)).With("test_name", t.Name())
}
