package collector

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
func (m *testRedfishServer) makeJSONHandler(response interface{}) http.HandlerFunc {
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
func (m *testRedfishServer) addRoute(path string, response interface{}) {
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

// connectToTestServer creates a gofish client connected to the test server
func connectToTestServer(t *testing.T, server *testRedfishServer) *gofish.APIClient {
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
