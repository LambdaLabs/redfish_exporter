package collector

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stmcginnis/gofish/redfish"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServerWithGPU creates a test server with GPU hardware
func setupTestServerWithGPU(t *testing.T) *testRedfishServer {
	server := &testRedfishServer{
		t:        t,
		mux:      http.NewServeMux(),
		requests: make([]string, 0),
	}
	server.Server = httptest.NewServer(server.mux)
	t.Cleanup(server.Close)
	
	// Add service root
	server.addRouteFromFixture("/redfish/v1/", "service_root.json")
	
	// Add systems collection
	server.addRoute("/redfish/v1/Systems", map[string]interface{}{
		"@odata.type": "#ComputerSystemCollection.ComputerSystemCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0"},
		},
		"Members@odata.count": 1,
	})
	
	setupGPUSystem(server)
	setupGPUMemory(server)
	
	return server
}

// setupGPUSystem adds the HGX system configuration
func setupGPUSystem(server *testRedfishServer) {
	server.addRoute("/redfish/v1/Systems/HGX_Baseboard_0", map[string]interface{}{
		"@odata.type": "#ComputerSystem.v1_14_0.ComputerSystem",
		"@odata.id":   "/redfish/v1/Systems/HGX_Baseboard_0",
		"Id":          "HGX_Baseboard_0",
		"Name":        "HGX System",
		"SystemType":  "Physical",
		"Manufacturer": "NVIDIA",
		"Model":       "HGX",
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
		"Memory": map[string]string{
			"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Memory",
		},
		"Processors": map[string]string{
			"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Processors",
		},
	})
}

// setupGPUMemory adds GPU memory configuration
func setupGPUMemory(server *testRedfishServer) {
	// Memory collection
	server.addRoute("/redfish/v1/Systems/HGX_Baseboard_0/Memory", map[string]interface{}{
		"@odata.type": "#MemoryCollection.MemoryCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_0_DRAM_0"},
			{"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_1_DRAM_0"},
			{"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Memory/DIMM_0"},
		},
		"Members@odata.count": 3,
	})
	
	// GPU_0 memory with fixtures
	server.addRouteFromFixture("/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_0_DRAM_0", 
		"nvidia_gpu_memory.json")
	server.addRouteFromFixture("/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_0_DRAM_0/MemoryMetrics",
		"nvidia_gpu_memory_metrics.json")
	
	// GPU_1 memory
	server.addRoute("/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_1_DRAM_0", map[string]interface{}{
		"@odata.type": "#Memory.v1_17_0.Memory",
		"@odata.id":   "/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_1_DRAM_0",
		"Id":          "GPU_1_DRAM_0",
		"Name":        "GPU_1_DRAM",
		"CapacityMiB": 98304,
		"MemoryDeviceType": "HBM2E",
		"Manufacturer": "NVIDIA",
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
		"Oem": map[string]interface{}{
			"Nvidia": map[string]interface{}{
				"RowRemappingFailed":  true,
				"RowRemappingPending": false,
			},
		},
	})
	
	// Regular DIMM
	server.addRoute("/redfish/v1/Systems/HGX_Baseboard_0/Memory/DIMM_0", map[string]interface{}{
		"@odata.type": "#Memory.v1_17_0.Memory",
		"@odata.id":   "/redfish/v1/Systems/HGX_Baseboard_0/Memory/DIMM_0",
		"Id":          "DIMM_0",
		"Name":        "DIMM_0",
		"CapacityMiB": 32768,
		"MemoryDeviceType": "DDR4",
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
	})
}

// TODO: Add these setup functions when processor and NVLink collection is fully implemented
// setupGPUProcessors and setupNVLinkPorts are commented out until needed

// collectAndCategorizeMetrics collects metrics and categorizes them
func collectAndCategorizeMetrics(t *testing.T, collector *GPUCollector) (map[string]float64, map[string]float64, map[string]float64, int) {
	ch := make(chan prometheus.Metric, 200)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()
	
	gpuMemoryMetrics := make(map[string]float64)
	gpuProcessorMetrics := make(map[string]float64)
	nvlinkMetrics := make(map[string]float64)
	metricsFound := 0
	
	for metric := range ch {
		dto := &dto.Metric{}
		if err := metric.Write(dto); err != nil {
			t.Errorf("Failed to write metric: %v", err)
			continue
		}
		
		metricsFound++
		categorizeMetric(metric, dto, gpuMemoryMetrics, gpuProcessorMetrics, nvlinkMetrics)
	}
	
	return gpuMemoryMetrics, gpuProcessorMetrics, nvlinkMetrics, metricsFound
}

// categorizeMetric categorizes a metric into the appropriate map
func categorizeMetric(metric prometheus.Metric, dto *dto.Metric, gpuMemory, gpuProcessor, nvlink map[string]float64) {
	desc := metric.Desc()
	descString := desc.String()
	
	var memoryID, processorID, portID string
	for _, label := range dto.Label {
		switch label.GetName() {
		case "memory_id":
			memoryID = label.GetValue()
		case "processor_id":
			processorID = label.GetValue()
		case "port_id":
			portID = label.GetValue()
		}
	}
	
	// Categorize GPU memory metrics
	if strings.Contains(memoryID, "GPU") {
		categorizeMemoryMetric(descString, memoryID, dto, gpuMemory)
	}
	
	// Categorize GPU processor metrics
	if processorID == "GPU_0" {
		categorizeProcessorMetric(descString, dto, gpuProcessor)
	}
	
	// Categorize NVLink port metrics
	if strings.Contains(portID, "NVLink") {
		categorizeNVLinkMetric(descString, dto, nvlink)
	}
}

// categorizeMemoryMetric categorizes memory metrics
func categorizeMemoryMetric(descString, memoryID string, dto *dto.Metric, metrics map[string]float64) {
	if strings.Contains(descString, "row_remapping_failed") {
		metrics["row_remapping_failed_"+memoryID] = dto.Gauge.GetValue()
	} else if strings.Contains(descString, "row_remapping_pending") {
		metrics["row_remapping_pending_"+memoryID] = dto.Gauge.GetValue()
	} else if strings.Contains(descString, "correctable_row_remapping_count") {
		metrics["correctable_row_remapping_count"] = dto.Gauge.GetValue()
	} else if strings.Contains(descString, "max_availability_bank_count") {
		metrics["max_availability_bank_count"] = dto.Gauge.GetValue()
	} else if strings.Contains(descString, "gpu_memory_capacity") {
		metrics["capacity_"+memoryID] = dto.Gauge.GetValue()
	}
}

// categorizeProcessorMetric categorizes processor metrics
func categorizeProcessorMetric(descString string, dto *dto.Metric, metrics map[string]float64) {
	if strings.Contains(descString, "sm_utilization_percent") {
		metrics["sm_utilization"] = dto.Gauge.GetValue()
	} else if strings.Contains(descString, "tensor_core_activity_percent") {
		metrics["tensor_core_activity"] = dto.Gauge.GetValue()
	} else if strings.Contains(descString, "fp32_activity_percent") {
		metrics["fp32_activity"] = dto.Gauge.GetValue()
	} else if strings.Contains(descString, "nvlink_data_rx_bandwidth_gbps") {
		metrics["nvlink_rx_bandwidth"] = dto.Gauge.GetValue()
	} else if strings.Contains(descString, "pcie_tx_bytes") {
		metrics["pcie_tx_bytes"] = dto.Gauge.GetValue()
	}
}

// categorizeNVLinkMetric categorizes NVLink metrics
func categorizeNVLinkMetric(descString string, dto *dto.Metric, metrics map[string]float64) {
	if strings.Contains(descString, "nvlink_runtime_error") {
		metrics["runtime_error"] = dto.Gauge.GetValue()
	} else if strings.Contains(descString, "nvlink_training_error") {
		metrics["training_error"] = dto.Gauge.GetValue()
	} else if strings.Contains(descString, "link_error_recovery_count") {
		metrics["error_recovery_count"] = dto.Gauge.GetValue()
	} else if strings.Contains(descString, "symbol_errors") {
		metrics["symbol_errors"] = dto.Gauge.GetValue()
	} else if strings.Contains(descString, "bit_error_rate") {
		metrics["bit_error_rate"] = dto.Gauge.GetValue()
	}
}

// verifyGPUMemoryMetrics verifies GPU memory metrics
func verifyGPUMemoryMetrics(t *testing.T, metrics map[string]float64) {
	if len(metrics) == 0 {
		t.Error("No GPU memory metrics were collected")
	}
	
	// Check GPU_0 metrics
	if val, ok := metrics["row_remapping_failed_GPU_0_DRAM_0"]; ok {
		if val != 0 {
			t.Errorf("Expected GPU_0 row_remapping_failed = 0, got %f", val)
		}
	} else {
		t.Error("GPU_0 row_remapping_failed metric not collected")
	}
	
	// Check GPU_1 metrics
	if val, ok := metrics["row_remapping_failed_GPU_1_DRAM_0"]; ok {
		if val != 1 {
			t.Errorf("Expected GPU_1 row_remapping_failed = 1, got %f", val)
		}
	} else {
		t.Error("GPU_1 row_remapping_failed metric not collected")
	}
	
	// Check OEM data
	if val, ok := metrics["max_availability_bank_count"]; ok {
		if val != 5952 {
			t.Errorf("Expected max_availability_bank_count = 5952, got %f", val)
		}
	}
}


// TestGPUCollectorWithNvidiaGPU tests the GPU collector with Nvidia GPU hardware
func TestGPUCollectorWithNvidiaGPU(t *testing.T) {
	server := setupTestServerWithGPU(t)
	client := connectToTestServer(t, server)
	defer client.Logout()
	
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector := NewGPUCollector(client, logger)
	
	gpuMemoryMetrics, _, _, metricsFound := collectAndCategorizeMetrics(t, collector)
	
	if metricsFound == 0 {
		t.Error("No metrics were collected")
	}
	
	verifyGPUMemoryMetrics(t, gpuMemoryMetrics)
	
	t.Logf("Successfully collected %d total metrics", metricsFound)
	t.Logf("GPU memory metrics: %d", len(gpuMemoryMetrics))
}

// TestGPUCollectorWithNoGPUs tests the GPU collector when no GPUs are present
func TestGPUCollectorWithNoGPUs(t *testing.T) {
	// Create a test server with no GPU hardware
	server := &testRedfishServer{
		t:        t,
		mux:      http.NewServeMux(),
		requests: make([]string, 0),
	}
	server.Server = httptest.NewServer(server.mux)
	t.Cleanup(server.Close)
	
	// Add service root
	server.addRouteFromFixture("/redfish/v1/", "service_root.json")
	
	// Add systems collection with regular system
	server.addRoute("/redfish/v1/Systems", map[string]interface{}{
		"@odata.type": "#ComputerSystemCollection.ComputerSystemCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Systems/System1"},
		},
		"Members@odata.count": 1,
	})
	
	// Add regular system without GPU components
	server.addRoute("/redfish/v1/Systems/System1", map[string]interface{}{
		"@odata.type": "#ComputerSystem.v1_14_0.ComputerSystem",
		"@odata.id":   "/redfish/v1/Systems/System1",
		"Id":          "System1",
		"Name":        "Regular System",
		"SystemType":  "Physical",
		"Manufacturer": "Dell",
		"Model":       "PowerEdge",
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
		"Memory": map[string]string{
			"@odata.id": "/redfish/v1/Systems/System1/Memory",
		},
		"Processors": map[string]string{
			"@odata.id": "/redfish/v1/Systems/System1/Processors",
		},
	})
	
	// Add Memory collection with only regular memory
	server.addRoute("/redfish/v1/Systems/System1/Memory", map[string]interface{}{
		"@odata.type": "#MemoryCollection.MemoryCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Systems/System1/Memory/DIMM_0"},
			{"@odata.id": "/redfish/v1/Systems/System1/Memory/DIMM_1"},
		},
		"Members@odata.count": 2,
	})
	
	// Add regular DIMMs
	server.addRoute("/redfish/v1/Systems/System1/Memory/DIMM_0", map[string]interface{}{
		"@odata.type": "#Memory.v1_17_0.Memory",
		"@odata.id":   "/redfish/v1/Systems/System1/Memory/DIMM_0",
		"Id":          "DIMM_0",
		"Name":        "DIMM_0",
		"CapacityMiB": 32768,
		"MemoryDeviceType": "DDR4",
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
	})
	
	// Add Processor collection with only CPUs
	server.addRoute("/redfish/v1/Systems/System1/Processors", map[string]interface{}{
		"@odata.type": "#ProcessorCollection.ProcessorCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Systems/System1/Processors/CPU_0"},
			{"@odata.id": "/redfish/v1/Systems/System1/Processors/CPU_1"},
		},
		"Members@odata.count": 2,
	})
	
	// Add CPU processors
	server.addRoute("/redfish/v1/Systems/System1/Processors/CPU_0", map[string]interface{}{
		"@odata.type": "#Processor.v1_14_0.Processor",
		"@odata.id":   "/redfish/v1/Systems/System1/Processors/CPU_0",
		"Id":          "CPU_0",
		"Name":        "CPU_0",
		"ProcessorType": "CPU",
		"Manufacturer": "Intel",
		"Model":       "Xeon",
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
	})
	
	// Connect to the test server
	client := connectToTestServer(t, server)
	defer client.Logout()
	
	// Create GPU collector
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector := NewGPUCollector(client, logger)
	
	// Collect metrics
	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()
	
	// Check that no GPU-specific metrics are collected
	gpuMetricsFound := 0
	for metric := range ch {
		dto := &dto.Metric{}
		if err := metric.Write(dto); err != nil {
			continue
		}
		
		desc := metric.Desc()
		descString := desc.String()
		
		// Check if this is a GPU-specific metric (should not find any)
		if strings.Contains(descString, "gpu_") || 
		   strings.Contains(descString, "nvlink") ||
		   strings.Contains(descString, "tensor_core") ||
		   strings.Contains(descString, "sm_utilization") {
			gpuMetricsFound++
			t.Errorf("Unexpected GPU metric found: %s", descString)
		}
	}
	
	if gpuMetricsFound > 0 {
		t.Errorf("Found %d GPU metrics when none were expected", gpuMetricsFound)
	}
	
	t.Log("Successfully verified no GPU metrics collected for non-GPU system")
}

// TestCollectGPUProcessorMetrics tests collection of GPU processor metrics with various health states
func TestCollectGPUProcessorMetrics(t *testing.T) {
	tests := map[string]struct {
		processorID    string
		processorName  string
		processorType  string
		health         string
		state          string
		expectMetric   bool
		expectedHealth float64
		expectedState  float64
	}{
		"healthy GPU": {
			processorID:    "GPU_0",
			processorName:  "NVIDIA GB300",
			processorType:  "GPU",
			health:         "OK",
			state:          "Enabled",
			expectMetric:   true,
			expectedHealth: 1,
			expectedState:  1,
		},
		"GPU with warning": {
			processorID:    "GPU_1",
			processorName:  "NVIDIA GB300",
			processorType:  "GPU",
			health:         "Warning",
			state:          "Enabled",
			expectMetric:   true,
			expectedHealth: 2,
			expectedState:  1,
		},
		"GPU with critical status": {
			processorID:    "GPU_2",
			processorName:  "NVIDIA GB300",
			processorType:  "GPU",
			health:         "Critical",
			state:          "Enabled",
			expectMetric:   true,
			expectedHealth: 3,
			expectedState:  1,
		},
		"disabled GPU": {
			processorID:    "GPU_3",
			processorName:  "NVIDIA GB300",
			processorType:  "GPU",
			health:         "OK",
			state:          "Disabled",
			expectMetric:   true,
			expectedHealth: 1,
			expectedState:  2,
		},
		"GPU processor identified by ID pattern": {
			processorID:    "GPU_4",
			processorName:  "Custom GPU",
			processorType:  "CPU", // Wrong type but ID contains GPU_
			health:         "OK",
			state:          "Enabled",
			expectMetric:   false, // Won't be collected since ProcessorType is not GPU
			expectedHealth: 1,
			expectedState:  1,
		},
		"non-GPU processor": {
			processorID:   "CPU_0",
			processorName: "Intel Xeon",
			processorType: "CPU",
			health:        "OK",
			state:         "Enabled",
			expectMetric:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create test server
			server := &testRedfishServer{
				t:        t,
				mux:      http.NewServeMux(),
				requests: make([]string, 0),
			}
			server.Server = httptest.NewServer(server.mux)
			defer server.Close()

			// Setup service root
			server.addRouteFromFixture("/redfish/v1/", "service_root.json")

			// Setup systems collection
			server.addRoute("/redfish/v1/Systems", map[string]interface{}{
				"@odata.type": "#ComputerSystemCollection.ComputerSystemCollection",
				"Members": []map[string]string{
					{"@odata.id": "/redfish/v1/Systems/System1"},
				},
				"Members@odata.count": 1,
			})

			// Setup system with processor
			server.addRoute("/redfish/v1/Systems/System1", map[string]interface{}{
				"@odata.type": "#ComputerSystem.v1_14_0.ComputerSystem",
				"Id":          "System1",
				"Name":        "Test System",
				"Processors": map[string]string{
					"@odata.id": "/redfish/v1/Systems/System1/Processors",
				},
			})

			// Setup processors collection
			server.addRoute("/redfish/v1/Systems/System1/Processors", map[string]interface{}{
				"@odata.type": "#ProcessorCollection.ProcessorCollection",
				"Members": []map[string]string{
					{"@odata.id": "/redfish/v1/Systems/System1/Processors/" + tt.processorID},
				},
				"Members@odata.count": 1,
			})

			// Setup individual processor
			server.addRoute("/redfish/v1/Systems/System1/Processors/"+tt.processorID, map[string]interface{}{
				"@odata.type":   "#Processor.v1_4_0.Processor",
				"Id":            tt.processorID,
				"Name":          tt.processorName,
				"ProcessorType": tt.processorType,
				"Status": map[string]string{
					"Health": tt.health,
					"State":  tt.state,
				},
				"TotalCores":   128,
				"TotalThreads": 128,
			})

			// Create collector and collect metrics
			client := connectToTestServer(t, server)

			collector := NewGPUCollector(client, slog.Default())
			ch := make(chan prometheus.Metric, 100)
			go func() {
				collector.Collect(ch)
				close(ch)
			}()

			// Check metrics
			foundHealth := false
			foundState := false

			for metric := range ch {
				desc := metric.Desc()
				descString := desc.String()
				
				dto := &dto.Metric{}
				if err := metric.Write(dto); err != nil {
					t.Errorf("failed to write metric: %v", err)
					continue
				}

				// Check if this is a GPU processor metric
				if strings.Contains(descString, "gpu_processor_health") {
					foundHealth = true
					if tt.expectMetric && dto.Gauge.GetValue() != tt.expectedHealth {
						t.Errorf("expected health value %f, got %f", tt.expectedHealth, dto.Gauge.GetValue())
					}
				}

				if strings.Contains(descString, "gpu_processor_state") {
					foundState = true
					if tt.expectMetric && dto.Gauge.GetValue() != tt.expectedState {
						t.Errorf("expected state value %f, got %f", tt.expectedState, dto.Gauge.GetValue())
					}
				}
			}

			if tt.expectMetric && !foundHealth {
				t.Error("expected GPU health metric but not found")
			}
			if tt.expectMetric && !foundState {
				t.Error("expected GPU state metric but not found")
			}
			if !tt.expectMetric && (foundHealth || foundState) {
				t.Error("found GPU metrics for non-GPU processor")
			}
		})
	}
}

// mockMemoryWithMetrics implements MemoryWithMetrics interface for testing
type mockMemoryWithMetrics struct {
	id                         string
	correctableECCErrorCount   int
	uncorrectableECCErrorCount int
	shouldError                bool
}

func (m *mockMemoryWithMetrics) GetID() string {
	return m.id
}

func (m *mockMemoryWithMetrics) Metrics() (*redfish.MemoryMetrics, error) {
	if m.shouldError {
		return nil, errors.New("metrics retrieval failed")
	}

	return &redfish.MemoryMetrics{
		CurrentPeriod: redfish.CurrentPeriod{
			CorrectableECCErrorCount:   m.correctableECCErrorCount,
			UncorrectableECCErrorCount: m.uncorrectableECCErrorCount,
		},
	}, nil
}

func TestEmitGPUECCMetrics(t *testing.T) {
	tT := map[string]struct {
		memories             []MemoryWithMetrics
		systemName           string
		expectedMetricCount  int
		expectedMetricChecks []struct {
			nameContains  string
			expectedValue float64
		}
	}{
		"happy path": {
			memories: []MemoryWithMetrics{
				&mockMemoryWithMetrics{
					id:                         "mockMem",
					correctableECCErrorCount:   100,
					uncorrectableECCErrorCount: 0,
				},
			},
			systemName:          "test",
			expectedMetricCount: 2,
			expectedMetricChecks: []struct {
				nameContains  string
				expectedValue float64
			}{
				{nameContains: "memory_ecc_correctable", expectedValue: 100},
				{nameContains: "memory_ecc_uncorrectable", expectedValue: 0},
			},
		},
		// should simply error from within functon-under-test, no returned metrics
		"no memory metrics": {
			memories: []MemoryWithMetrics{
				&mockMemoryWithMetrics{
					id:          "mockMem",
					shouldError: true,
				},
			},
			systemName:          "test",
			expectedMetricCount: 0,
		},
	}

	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			outCh := make(chan prometheus.Metric, 100)
			go func() {
				emitGPUECCMetrics(outCh, test.memories, slog.Default(), []string{"testGPU", test.systemName, "testGPUId"}, gpuMetrics)
				close(outCh)
			}()

			var metrics []prometheus.Metric
			for metric := range outCh {
				metrics = append(metrics, metric)
			}

			require.Equal(t, test.expectedMetricCount, len(metrics), "unexpected number of metrics")

			for _, check := range test.expectedMetricChecks {
				found := false
				for _, metric := range metrics {
					if descContains(metric, check.nameContains) {
						found = true
						dtoMetric := &dto.Metric{}
						require.NoError(t, metric.Write(dtoMetric), "unexpected error writing DTO metric")
						requireCounterWithValue(t, dtoMetric, check.expectedValue)
						break
					}
				}
				assert.True(t, found, "expected metric containing '%s' not found", check.nameContains)
			}
		})
	}
}

func requireGaugeWithValue(t *testing.T, metric *dto.Metric, expected float64) {
	t.Helper()
	require.NotNil(t, metric.Gauge, "required a gauge")
	require.Equal(t, expected, *metric.Gauge.Value)
}

func requireCounterWithValue(t *testing.T, metric *dto.Metric, expected float64) {
	t.Helper()
	require.NotNil(t, metric.Counter, "required a counter")
	require.Equal(t, expected, *metric.Counter.Value)
}

func requireMetricDescContains(t *testing.T, m prometheus.Metric, contains string) {
	t.Helper()
	desc := m.Desc()
	assert.NotNil(t, desc)
	require.Contains(t, desc.String(), contains)
}

func descContains(m prometheus.Metric, contains string) bool {
	desc := m.Desc()
	if desc == nil {
		return false
	}
	return strings.Contains(desc.String(), contains)
}
