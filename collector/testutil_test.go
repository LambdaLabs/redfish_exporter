package collector

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stmcginnis/gofish"
	"github.com/stretchr/testify/require"
)

// mockRedfishServer creates a test server with configurable responses
type mockRedfishServer struct {
	*httptest.Server
	t         *testing.T
	responses map[string]interface{}
	requests  []string // Track requests for debugging
}

// newMockRedfishServer creates a new mock Redfish server for testing
func newMockRedfishServer(t *testing.T) *mockRedfishServer {
	t.Helper()
	
	mrs := &mockRedfishServer{
		t:         t,
		responses: make(map[string]interface{}),
		requests:  make([]string, 0),
	}
	
	mrs.Server = httptest.NewServer(http.HandlerFunc(mrs.handler))
	t.Cleanup(mrs.Close)
	
	// Set default responses
	mrs.setDefaultResponses()
	
	return mrs
}

// handler processes requests and returns mock responses
func (m *mockRedfishServer) handler(w http.ResponseWriter, r *http.Request) {
	m.requests = append(m.requests, r.Method+" "+r.URL.Path)
	
	w.Header().Set("Content-Type", "application/json")
	
	if response, ok := m.responses[r.URL.Path]; ok {
		if err := json.NewEncoder(w).Encode(response); err != nil {
			m.t.Errorf("Failed to encode response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

// setDefaultResponses sets up the minimum required Redfish responses
func (m *mockRedfishServer) setDefaultResponses() {
	m.responses["/redfish/v1/"] = map[string]interface{}{
		"@odata.type":    "#ServiceRoot.v1_15_0.ServiceRoot",
		"@odata.id":      "/redfish/v1/",
		"Id":             "RootService",
		"Name":           "Root Service",
		"RedfishVersion": "1.15.0",
		"Systems":        map[string]string{"@odata.id": "/redfish/v1/Systems"},
	}
	
	m.responses["/redfish/v1/Systems"] = map[string]interface{}{
		"@odata.type": "#ComputerSystemCollection.ComputerSystemCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Systems/System1"},
		},
	}
}

// addProcessorWithMetrics adds a processor with full ProcessorMetrics to the mock
func (m *mockRedfishServer) addProcessorWithMetrics(systemID, processorID string, pcieErrors, cacheErrors map[string]int) {
	systemPath := "/redfish/v1/Systems/" + systemID
	processorPath := systemPath + "/Processors/" + processorID
	metricsPath := processorPath + "/ProcessorMetrics"
	
	// Add system
	m.responses[systemPath] = map[string]interface{}{
		"@odata.type": "#ComputerSystem.v1_20_0.ComputerSystem",
		"@odata.id":   systemPath,
		"Id":          systemID,
		"Name":        "Test System",
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
		"Processors": map[string]string{
			"@odata.id": systemPath + "/Processors",
		},
	}
	
	// Add processor collection
	m.responses[systemPath+"/Processors"] = map[string]interface{}{
		"@odata.type": "#ProcessorCollection.ProcessorCollection",
		"Members": []map[string]string{
			{"@odata.id": processorPath},
		},
	}
	
	// Add processor
	m.responses[processorPath] = map[string]interface{}{
		"@odata.type":  "#Processor.v1_20_0.Processor",
		"@odata.id":    processorPath,
		"Id":           processorID,
		"Name":         "Test Processor",
		"TotalCores":   128,
		"TotalThreads": 256,
		"Status": map[string]string{
			"State":        "Enabled",
			"Health":       "OK",
			"HealthRollup": "OK",
		},
		"Metrics": map[string]string{
			"@odata.id": metricsPath,
		},
	}
	
	// Add processor metrics
	m.responses[metricsPath] = map[string]interface{}{
		"@odata.type": "#ProcessorMetrics.v1_6_1.ProcessorMetrics",
		"@odata.id":   metricsPath,
		"PCIeErrors":  pcieErrors,
		"CacheMetricsTotal": map[string]interface{}{
			"LifeTime": cacheErrors,
		},
	}
}

// connectToMockServer creates a gofish client connected to the mock server
func connectToMockServer(t *testing.T, server *mockRedfishServer) *gofish.APIClient {
	t.Helper()
	
	config := gofish.ClientConfig{
		Endpoint: server.URL,
		Username: "",
		Password: "",
		Insecure: true,
	}
	
	client, err := gofish.Connect(config)
	require.NoError(t, err, "Failed to connect to mock server")
	
	return client
}

// collectProcessorMetrics runs the collector and returns metrics as a map
func collectProcessorMetrics(t *testing.T, client *gofish.APIClient) map[string]float64 {
	t.Helper()
	
	// Create a test logger
	logger := slog.Default()
	
	collector := NewSystemCollector(client, logger)
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
			// Extract metric name from description
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
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && strings.Contains(s, substr))
}