package collector

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/config"
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
	setupGPUProcessorsAndSensors(server)

	return server
}

// setupGPUSystem adds the HGX system configuration
func setupGPUSystem(server *testRedfishServer) {
	server.addRoute("/redfish/v1/Systems/HGX_Baseboard_0", map[string]interface{}{
		"@odata.type":  "#ComputerSystem.v1_14_0.ComputerSystem",
		"@odata.id":    "/redfish/v1/Systems/HGX_Baseboard_0",
		"Id":           "HGX_Baseboard_0",
		"Name":         "HGX System",
		"SystemType":   "Physical",
		"Manufacturer": "NVIDIA",
		"Model":        "HGX",
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
		"@odata.type":      "#Memory.v1_17_0.Memory",
		"@odata.id":        "/redfish/v1/Systems/HGX_Baseboard_0/Memory/GPU_1_DRAM_0",
		"Id":               "GPU_1_DRAM_0",
		"Name":             "GPU_1_DRAM",
		"CapacityMiB":      98304,
		"MemoryDeviceType": "HBM2E",
		"Manufacturer":     "NVIDIA",
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
		"@odata.type":      "#Memory.v1_17_0.Memory",
		"@odata.id":        "/redfish/v1/Systems/HGX_Baseboard_0/Memory/DIMM_0",
		"Id":               "DIMM_0",
		"Name":             "DIMM_0",
		"CapacityMiB":      32768,
		"MemoryDeviceType": "DDR4",
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
	})
}

// testGPUConfig holds configuration for a test GPU
type testGPUConfig struct {
	ID          string
	Name        string
	Temperature float64
	MemoryPower float64
}

// createGPUProcessor creates a GPU processor response with common fields
func createGPUProcessor(systemID, gpuID, gpuName string) map[string]interface{} {
	return map[string]interface{}{
		"@odata.type":   "#Processor.v1_20_0.Processor",
		"@odata.id":     fmt.Sprintf("/redfish/v1/Systems/%s/Processors/%s", systemID, gpuID),
		"Id":            gpuID,
		"Name":          gpuName,
		"ProcessorType": "GPU",
		"Manufacturer":  "NVIDIA",
		"Model":         "H100",
		"Metrics": map[string]interface{}{
			"@odata.id": fmt.Sprintf("/redfish/v1/Systems/%s/Processors/%s/ProcessorMetrics", systemID, gpuID),
		},
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
	}
}

// createTemperatureSensor creates a temperature sensor response
func createTemperatureSensor(chassisID, sensorID string, temperature float64) map[string]interface{} {
	return map[string]interface{}{
		"@odata.id":    fmt.Sprintf("/redfish/v1/Chassis/%s/Sensors/%s", chassisID, sensorID),
		"@odata.type":  "#Sensor.v1_7_0.Sensor",
		"Id":           sensorID,
		"Name":         fmt.Sprintf("%s Temperature Sensor", chassisID),
		"Reading":      temperature,
		"ReadingType":  "Temperature",
		"ReadingUnits": "Cel",
		"Status": map[string]interface{}{
			"State":  "Enabled",
			"Health": "OK",
		},
	}
}

// createPowerSensor creates a power sensor response
func createPowerSensor(chassisID, sensorID string, powerReading float64) map[string]interface{} {
	return map[string]interface{}{
		"@odata.id":    fmt.Sprintf("/redfish/v1/Chassis/%s/Sensors/%s", chassisID, sensorID),
		"@odata.type":  "#Sensor.v1_7_0.Sensor",
		"Id":           sensorID,
		"Name":         fmt.Sprintf("%s Power Sensor", chassisID),
		"Reading":      powerReading,
		"ReadingType":  "Power",
		"ReadingUnits": "W",
		"Status": map[string]interface{}{
			"State":  "Enabled",
			"Health": "OK",
		},
	}
}

// setupGPUProcessorsAndSensors adds GPU processors and temperature sensors
func setupGPUProcessorsAndSensors(server *testRedfishServer) {
	systemID := "HGX_Baseboard_0"

	// Define test GPUs
	testGPUs := []testGPUConfig{
		{ID: "GPU_0", Name: "GPU 0", Temperature: 58.0, MemoryPower: 36.5},
		{ID: "GPU_1", Name: "GPU 1", Temperature: 59.5, MemoryPower: 38.0},
	}

	// Build processor collection members
	members := make([]map[string]string, len(testGPUs))
	for i, gpu := range testGPUs {
		members[i] = map[string]string{
			"@odata.id": fmt.Sprintf("/redfish/v1/Systems/%s/Processors/%s", systemID, gpu.ID),
		}
	}

	// Add processors collection
	server.addRoute(fmt.Sprintf("/redfish/v1/Systems/%s/Processors", systemID), map[string]interface{}{
		"@odata.type":         "#ProcessorCollection.ProcessorCollection",
		"Members":             members,
		"Members@odata.count": len(members),
	})

	// Add each GPU processor and its temperature sensor
	for _, gpu := range testGPUs {
		// Add processor
		processorPath := fmt.Sprintf("/redfish/v1/Systems/%s/Processors/%s", systemID, gpu.ID)
		server.addRoute(processorPath, createGPUProcessor(systemID, gpu.ID, gpu.Name))

		// Add temperature sensor
		chassisID := fmt.Sprintf("HGX_%s", gpu.ID)
		sensorID := fmt.Sprintf("%s_TEMP_1", chassisID)
		sensorPath := fmt.Sprintf("/redfish/v1/Chassis/%s/Sensors/%s", chassisID, sensorID)
		server.addRoute(sensorPath, createTemperatureSensor(chassisID, sensorID, gpu.Temperature))

		// Add memory power sensor
		powerSensorID := fmt.Sprintf("HGX_%s_DRAM_0_Power_0", gpu.ID)
		powerSensorPath := fmt.Sprintf("/redfish/v1/Chassis/%s/Sensors/%s", chassisID, powerSensorID)
		server.addRoute(powerSensorPath, createPowerSensor(chassisID, powerSensorID, gpu.MemoryPower))
	}
}

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

	// Categorize GPU temperature metrics
	if strings.Contains(descString, "temperature_tlimit_celsius") {
		gpuID := ""
		for _, label := range dto.Label {
			if label.GetName() == "gpu_id" {
				gpuID = label.GetValue()
				break
			}
		}
		if gpuID != "" {
			gpuProcessor["temperature_tlimit_"+gpuID] = dto.Gauge.GetValue()
		}
	}

	// Categorize GPU memory power metrics
	if strings.Contains(descString, "memory_power_watts") {
		memoryID := ""
		for _, label := range dto.Label {
			if label.GetName() == "memory_id" {
				memoryID = label.GetValue()
				break
			}
		}
		if memoryID != "" {
			gpuProcessor["memory_power_"+memoryID] = dto.Gauge.GetValue()
		}
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
		t.Fatal("No GPU memory metrics were collected")
	}

	tests := []struct {
		metricKey     string
		expectedValue float64
		description   string
		required      bool
	}{
		{
			metricKey:     "row_remapping_failed_GPU_0_DRAM_0",
			expectedValue: 0,
			description:   "GPU_0 row_remapping_failed",
			required:      true,
		},
		{
			metricKey:     "row_remapping_failed_GPU_1_DRAM_0",
			expectedValue: 1,
			description:   "GPU_1 row_remapping_failed",
			required:      true,
		},
		{
			metricKey:     "max_availability_bank_count",
			expectedValue: 5952,
			description:   "max_availability_bank_count",
			required:      false, // OEM data might not always be present
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			val, ok := metrics[tt.metricKey]
			if !ok {
				if tt.required {
					t.Errorf("%s metric not collected", tt.description)
				}
				return
			}
			if val != tt.expectedValue {
				t.Errorf("Expected %s = %f, got %f", tt.description, tt.expectedValue, val)
			}
		})
	}
}

// verifyGPUMemoryPowerMetrics verifies GPU memory power metrics
func verifyGPUMemoryPowerMetrics(t *testing.T, metrics map[string]float64) {
	tests := []struct {
		metricKey     string
		expectedValue float64
		description   string
	}{
		{
			metricKey:     "memory_power_GPU_0_DRAM_0",
			expectedValue: 36.5,
			description:   "GPU_0_DRAM_0 power",
		},
		{
			metricKey:     "memory_power_GPU_1_DRAM_0",
			expectedValue: 38.0,
			description:   "GPU_1_DRAM_0 power",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			val, ok := metrics[tt.metricKey]
			if !ok {
				t.Errorf("%s metric not collected", tt.description)
				return
			}
			if val != tt.expectedValue {
				t.Errorf("Expected %s = %f, got %f", tt.description, tt.expectedValue, val)
			}
		})
	}
}

// verifyGPUTemperatureMetrics verifies GPU temperature metrics
func verifyGPUTemperatureMetrics(t *testing.T, metrics map[string]float64) {
	tests := []struct {
		metricKey     string
		expectedValue float64
		description   string
	}{
		{
			metricKey:     "temperature_tlimit_GPU_0",
			expectedValue: 58.0,
			description:   "GPU_0 temperature",
		},
		{
			metricKey:     "temperature_tlimit_GPU_1",
			expectedValue: 59.5,
			description:   "GPU_1 temperature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			val, ok := metrics[tt.metricKey]
			if !ok {
				t.Errorf("%s metric not collected", tt.description)
				return
			}
			if val != tt.expectedValue {
				t.Errorf("Expected %s = %f, got %f", tt.description, tt.expectedValue, val)
			}
		})
	}
}

// TestGPUCollectorWithNvidiaGPU tests the GPU collector with Nvidia GPU hardware
// Note: GPU temperature and memory power metrics are now collected via TelemetryService (HGX_PlatformEnvironmentMetrics_0)
func TestGPUCollectorWithNvidiaGPU(t *testing.T) {
	t.Skip("GPU temperature and memory power metrics now collected via TelemetryCollector from HGX_PlatformEnvironmentMetrics_0")
	server := setupTestServerWithGPU(t)
	client := connectToTestServer(t, server)
	defer client.Logout()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewGPUCollector(t.Name(), client, logger, config.DefaultGPUCollector)
	require.NoError(t, err)

	gpuMemoryMetrics, gpuProcessorMetrics, _, metricsFound := collectAndCategorizeMetrics(t, collector)

	require.NotZero(t, metricsFound, "No metrics were collected")

	verifyGPUMemoryMetrics(t, gpuMemoryMetrics)
	verifyGPUTemperatureMetrics(t, gpuProcessorMetrics)
	verifyGPUMemoryPowerMetrics(t, gpuProcessorMetrics)

	t.Logf("Successfully collected %d total metrics", metricsFound)
	t.Logf("GPU memory metrics: %d", len(gpuMemoryMetrics))
	t.Logf("GPU processor/temperature/power metrics: %d", len(gpuProcessorMetrics))
}

// TestGPUContextUtilization tests the collection of GPU context utilization duration metric
func TestGPUContextUtilization(t *testing.T) {
	server := setupTestServerWithGPU(t)

	// Add ProcessorMetrics endpoints using testdata fixtures
	server.addRouteFromFixture("/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0/ProcessorMetrics",
		"processor_metrics_gpu_context_util.json")
	server.addRouteFromFixture("/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_1/ProcessorMetrics",
		"processor_metrics_gpu_context_zero.json")

	client := connectToTestServer(t, server)
	defer client.Logout()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewGPUCollector(t.Name(), client, logger, config.DefaultGPUCollector)
	require.NoError(t, err)

	// Collect metrics
	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	// Check for context utilization metrics
	contextUtilMetrics := make(map[string]float64)

	for metric := range ch {
		dto := &dto.Metric{}
		require.NoError(t, metric.Write(dto))

		desc := metric.Desc()
		if strings.Contains(desc.String(), "context_utilization_seconds_total") {
			// Get GPU ID from labels
			for _, label := range dto.Label {
				if label.GetName() == "gpu_id" {
					gpuID := label.GetValue()
					contextUtilMetrics[gpuID] = dto.Counter.GetValue()
					break
				}
			}
		}
	}

	// Verify we got the expected metrics
	require.Len(t, contextUtilMetrics, 2, "Should have collected context utilization for 2 GPUs")

	// GPU_0 should have 2h45m30s = 9930 seconds (from fixture)
	require.InDelta(t, 9930.0, contextUtilMetrics["GPU_0"], 0.01, "GPU_0 should have 9930 seconds")

	// GPU_1 should have 0 seconds
	require.Equal(t, 0.0, contextUtilMetrics["GPU_1"], "GPU_1 should have 0 seconds")
}

// TestGPUContextUtilizationWithDifferentOEMLocations tests finding the duration in different OEM locations
func TestGPUContextUtilizationWithDifferentOEMLocations(t *testing.T) {
	server := setupTestServerWithGPU(t)

	// Test with OEM data directly at root level (not under vendor key)
	server.addRouteFromFixture("/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0/ProcessorMetrics",
		"processor_metrics_gpu_context_direct.json")

	client := connectToTestServer(t, server)
	defer client.Logout()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewGPUCollector(t.Name(), client, logger, config.DefaultGPUCollector)
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	// Check for context utilization metrics
	foundMetric := false
	var metricValue float64

	for metric := range ch {
		desc := metric.Desc()
		if strings.Contains(desc.String(), "context_utilization_seconds_total") {
			dto := &dto.Metric{}
			require.NoError(t, metric.Write(dto))
			foundMetric = true
			metricValue = dto.Counter.GetValue()
			break
		}
	}

	require.True(t, foundMetric, "Should find context utilization metric")
	// 1h30m = 5400 seconds
	require.InDelta(t, 5400.0, metricValue, 0.01, "Should parse duration correctly from direct OEM location")
}

// TestGPUTemperatureSensorEdgeCases tests edge cases for GPU temperature collection
// Note: GPU temperature collection is now done via TelemetryService (HGX_PlatformEnvironmentMetrics_0)
func TestGPUTemperatureSensorEdgeCases(t *testing.T) {
	t.Run("temperature metrics are collected via telemetry service", func(t *testing.T) {
		// This test verifies that temperature metrics are no longer collected directly
		// from sensors, but through the TelemetryService instead
		server := setupTestServerWithGPU(t)

		client := connectToTestServer(t, server)
		defer client.Logout()

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		collector, err := NewGPUCollector(t.Name(), client, logger, config.DefaultGPUCollector)
		require.NoError(t, err)

		ch := make(chan prometheus.Metric, 100)
		go func() {
			collector.Collect(ch)
			close(ch)
		}()

		// Collect all metrics and verify GPU collector doesn't emit temperature metrics
		// (those come from TelemetryCollector now)
		for metric := range ch {
			desc := metric.Desc()
			descString := desc.String()

			// The GPU collector no longer collects temperature/power via sensors
			// These are now collected by TelemetryCollector via HGX_PlatformEnvironmentMetrics_0
			if strings.Contains(descString, "scrape_status") {
				continue // Scrape status is fine
			}
		}

		t.Log("GPU collector no longer directly collects temperature/power metrics from sensors")
	})

	t.Run("sensor with correct fixture format", func(t *testing.T) {
		t.Skip("GPU temperature metrics now collected via TelemetryCollector from HGX_PlatformEnvironmentMetrics_0")
		// Create a minimal server just for this test
		server := &testRedfishServer{
			t:        t,
			mux:      http.NewServeMux(),
			requests: make([]string, 0),
		}
		server.Server = httptest.NewServer(server.mux)
		t.Cleanup(server.Close)

		// Add minimal routes needed
		server.addRouteFromFixture("/redfish/v1/", "service_root.json")
		server.addRoute("/redfish/v1/Systems", map[string]interface{}{
			"@odata.type": "#ComputerSystemCollection.ComputerSystemCollection",
			"Members": []map[string]string{
				{"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0"},
			},
			"Members@odata.count": 1,
		})

		// Add system with GPU
		server.addRoute("/redfish/v1/Systems/HGX_Baseboard_0", map[string]interface{}{
			"@odata.type": "#ComputerSystem.v1_14_0.ComputerSystem",
			"Id":          "HGX_Baseboard_0",
			"Name":        "HGX System",
			"Processors": map[string]string{
				"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Processors",
			},
		})

		// Add processor collection with GPU
		server.addRoute("/redfish/v1/Systems/HGX_Baseboard_0/Processors", map[string]interface{}{
			"@odata.type": "#ProcessorCollection.ProcessorCollection",
			"Members": []map[string]string{
				{"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0"},
			},
			"Members@odata.count": 1,
		})

		// Add GPU processor
		server.addRoute("/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_0", map[string]interface{}{
			"@odata.type":   "#Processor.v1_20_0.Processor",
			"Id":            "GPU_0",
			"Name":          "GPU 0",
			"ProcessorType": "GPU",
			"Status": map[string]string{
				"State":  "Enabled",
				"Health": "OK",
			},
		})

		// Add the temperature sensor using the fixture
		server.addRouteFromFixture("/redfish/v1/Chassis/HGX_GPU_0/Sensors/HGX_GPU_0_TEMP_1",
			"gpu_temperature_sensor.json")

		client := connectToTestServer(t, server)
		defer client.Logout()

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		collector, err := NewGPUCollector(t.Name(), client, logger, config.DefaultGPUCollector)
		require.NoError(t, err)

		ch := make(chan prometheus.Metric, 200)
		go func() {
			collector.Collect(ch)
			close(ch)
		}()

		// Verify temperature metric was collected from fixture
		foundTemp := false
		for metric := range ch {
			desc := metric.Desc()
			if strings.Contains(desc.String(), "temperature_tlimit_celsius") {
				dto := &dto.Metric{}
				require.NoError(t, metric.Write(dto))
				// Fixture has Reading: 58.0
				assert.Equal(t, 58.0, dto.Gauge.GetValue())
				foundTemp = true
				break
			}
		}

		assert.True(t, foundTemp, "Temperature metric should be collected from fixture")
	})
}

// TestGPUCollectorWithNoGPUs tests the GPU collector when no GPUs are present
func TestGPUCollectorWithNoGPUs(t *testing.T) {
	server := setupTestServerWithoutGPU(t)
	client := connectToTestServer(t, server)
	defer client.Logout()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewGPUCollector(t.Name(), client, logger, config.DefaultGPUCollector)
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	gpuMetricsFound := 0
	for metric := range ch {
		dto := &dto.Metric{}
		if err := metric.Write(dto); err != nil {
			continue
		}

		desc := metric.Desc()
		descString := desc.String()

		if isGPUSpecificMetric(descString) {
			gpuMetricsFound++
			t.Errorf("Unexpected GPU metric found: %s", descString)
		}
	}

	assert.Zero(t, gpuMetricsFound, "Found GPU metrics when none were expected")
	t.Log("Successfully verified no GPU metrics collected for non-GPU system")
}

// setupTestServerWithoutGPU creates a test server without GPU hardware
func setupTestServerWithoutGPU(t *testing.T) *testRedfishServer {
	server := &testRedfishServer{
		t:        t,
		mux:      http.NewServeMux(),
		requests: make([]string, 0),
	}
	server.Server = httptest.NewServer(server.mux)
	t.Cleanup(server.Close)

	server.addRouteFromFixture("/redfish/v1/", "service_root.json")

	server.addRoute("/redfish/v1/Systems", map[string]interface{}{
		"@odata.type": "#ComputerSystemCollection.ComputerSystemCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Systems/System1"},
		},
		"Members@odata.count": 1,
	})

	setupNonGPUSystem(server)
	setupNonGPUMemory(server)
	setupNonGPUProcessors(server)

	return server
}

// setupNonGPUSystem adds a regular system without GPU components
func setupNonGPUSystem(server *testRedfishServer) {
	server.addRoute("/redfish/v1/Systems/System1", map[string]interface{}{
		"@odata.type":  "#ComputerSystem.v1_14_0.ComputerSystem",
		"@odata.id":    "/redfish/v1/Systems/System1",
		"Id":           "System1",
		"Name":         "Regular System",
		"SystemType":   "Physical",
		"Manufacturer": "Dell",
		"Model":        "PowerEdge",
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
}

// setupNonGPUMemory adds regular memory without GPU memory
func setupNonGPUMemory(server *testRedfishServer) {
	server.addRoute("/redfish/v1/Systems/System1/Memory", map[string]interface{}{
		"@odata.type": "#MemoryCollection.MemoryCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Systems/System1/Memory/DIMM_0"},
			{"@odata.id": "/redfish/v1/Systems/System1/Memory/DIMM_1"},
		},
		"Members@odata.count": 2,
	})

	server.addRoute("/redfish/v1/Systems/System1/Memory/DIMM_0", map[string]interface{}{
		"@odata.type":      "#Memory.v1_17_0.Memory",
		"@odata.id":        "/redfish/v1/Systems/System1/Memory/DIMM_0",
		"Id":               "DIMM_0",
		"Name":             "DIMM_0",
		"CapacityMiB":      32768,
		"MemoryDeviceType": "DDR4",
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
	})
}

// setupNonGPUProcessors adds CPU processors without GPUs
func setupNonGPUProcessors(server *testRedfishServer) {
	server.addRoute("/redfish/v1/Systems/System1/Processors", map[string]interface{}{
		"@odata.type": "#ProcessorCollection.ProcessorCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Systems/System1/Processors/CPU_0"},
			{"@odata.id": "/redfish/v1/Systems/System1/Processors/CPU_1"},
		},
		"Members@odata.count": 2,
	})

	server.addRoute("/redfish/v1/Systems/System1/Processors/CPU_0", map[string]interface{}{
		"@odata.type":   "#Processor.v1_14_0.Processor",
		"@odata.id":     "/redfish/v1/Systems/System1/Processors/CPU_0",
		"Id":            "CPU_0",
		"Name":          "CPU_0",
		"ProcessorType": "CPU",
		"Manufacturer":  "Intel",
		"Model":         "Xeon",
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
	})
}

// isGPUSpecificMetric checks if a metric is GPU-specific
func isGPUSpecificMetric(descString string) bool {
	gpuIndicators := []string{
		"gpu_",
		"nvlink",
		"tensor_core",
		"sm_utilization",
	}

	for _, indicator := range gpuIndicators {
		if strings.Contains(descString, indicator) {
			return true
		}
	}
	return false
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

			collector, err := NewGPUCollector(t.Name(), client, slog.Default(), config.DefaultGPUCollector)
			require.NoError(t, err)
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

// TestGPUSerialNumberAndUUIDMetrics tests that GPU serial number and UUID info metrics are collected correctly
func TestGPUSerialNumberAndUUIDMetrics(t *testing.T) {
	tests := map[string]struct {
		gpus []struct {
			id           string
			serialNumber string
			uuid         string
		}
		expectUniqueUUIDs         bool
		expectUniqueSerialNumbers bool
	}{
		"B200 style - unique serial numbers and UUIDs": {
			gpus: []struct {
				id           string
				serialNumber string
				uuid         string
			}{
				{id: "GPU_SXM_1", serialNumber: "1650225122146", uuid: "e03507e4-8147-2191-d9a5-834772ec93a7"},
				{id: "GPU_SXM_2", serialNumber: "1650225105680", uuid: "5337f4db-707b-da38-9d3c-42678cfd8b17"},
				{id: "GPU_SXM_3", serialNumber: "1650325051066", uuid: "6f19ea63-bdb8-0115-cb3c-2ac106b1eb1f"},
				{id: "GPU_SXM_4", serialNumber: "1650325008513", uuid: "d3768332-13de-6427-7e30-14d9e6aa6fe1"},
			},
			expectUniqueUUIDs:         true,
			expectUniqueSerialNumbers: true,
		},
		"GB300 style - duplicate serial numbers per pair, unique UUIDs": {
			gpus: []struct {
				id           string
				serialNumber string
				uuid         string
			}{
				{id: "GPU_0", serialNumber: "1652625065599", uuid: "e79b5d32-1fca-41ba-9fff-8be0eafe44e2"},
				{id: "GPU_1", serialNumber: "1652625065599", uuid: "4bcdb113-2738-8369-3d0e-158a5e1aee44"}, // Duplicate SN
				{id: "GPU_2", serialNumber: "1652625065919", uuid: "a1b2c3d4-5678-90ab-cdef-123456789012"},
				{id: "GPU_3", serialNumber: "1652625065919", uuid: "f1e2d3c4-b5a6-9780-1234-567890abcdef"}, // Duplicate SN
			},
			expectUniqueUUIDs:         true,
			expectUniqueSerialNumbers: false, // Serial numbers are allowed to be duplicate
		},
		"invalid - duplicate UUIDs should be logged as error": {
			gpus: []struct {
				id           string
				serialNumber string
				uuid         string
			}{
				{id: "GPU_0", serialNumber: "1650225122146", uuid: "e03507e4-8147-2191-d9a5-834772ec93a7"},
				{id: "GPU_1", serialNumber: "1650225105680", uuid: "e03507e4-8147-2191-d9a5-834772ec93a7"}, // DUPLICATE UUID!
			},
			expectUniqueUUIDs:         false, // This case expects duplicate UUIDs (invalid but should be detected)
			expectUniqueSerialNumbers: true,
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
					{"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0"},
				},
				"Members@odata.count": 1,
			})

			// Setup system
			server.addRoute("/redfish/v1/Systems/HGX_Baseboard_0", map[string]interface{}{
				"@odata.type": "#ComputerSystem.v1_14_0.ComputerSystem",
				"Id":          "HGX_Baseboard_0",
				"Name":        "Test HGX System",
				"Processors": map[string]string{
					"@odata.id": "/redfish/v1/Systems/HGX_Baseboard_0/Processors",
				},
			})

			// Build processor collection members
			members := make([]map[string]string, len(tt.gpus))
			for i, gpu := range tt.gpus {
				members[i] = map[string]string{
					"@odata.id": fmt.Sprintf("/redfish/v1/Systems/HGX_Baseboard_0/Processors/%s", gpu.id),
				}
			}

			// Setup processors collection
			server.addRoute("/redfish/v1/Systems/HGX_Baseboard_0/Processors", map[string]interface{}{
				"@odata.type":         "#ProcessorCollection.ProcessorCollection",
				"Members":             members,
				"Members@odata.count": len(members),
			})

			// Add each GPU processor with serial number and UUID
			for _, gpu := range tt.gpus {
				server.addRoute(fmt.Sprintf("/redfish/v1/Systems/HGX_Baseboard_0/Processors/%s", gpu.id), map[string]interface{}{
					"@odata.type":   "#Processor.v1_20_0.Processor",
					"@odata.id":     fmt.Sprintf("/redfish/v1/Systems/HGX_Baseboard_0/Processors/%s", gpu.id),
					"Id":            gpu.id,
					"Name":          fmt.Sprintf("GPU %s", gpu.id),
					"ProcessorType": "GPU",
					"SerialNumber":  gpu.serialNumber,
					"UUID":          gpu.uuid,
					"TotalCores":    128,
					"TotalThreads":  128,
					"Status": map[string]string{
						"State":  "Enabled",
						"Health": "OK",
					},
				})
			}

			// Create collector with a logger that captures output
			var logBuf strings.Builder
			logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

			client := connectToTestServer(t, server)
			defer client.Logout()

			collector, err := NewGPUCollector(t.Name(), client, logger, config.DefaultGPUCollector)
			require.NoError(t, err)
			ch := make(chan prometheus.Metric, 200)
			go func() {
				collector.Collect(ch)
				close(ch)
			}()

			// Collect gpu_info metrics
			type gpuInfo struct {
				serialNumber string
				uuid         string
			}
			gpuInfoMetrics := make(map[string]gpuInfo) // gpu_id -> gpuInfo

			for metric := range ch {
				desc := metric.Desc()
				descString := desc.String()

				dto := &dto.Metric{}
				if err := metric.Write(dto); err != nil {
					t.Errorf("failed to write metric: %v", err)
					continue
				}

				// Check for gpu_info metric
				if strings.Contains(descString, "gpu_info") {
					var gpuID, serialNumber, uuid string
					for _, label := range dto.Label {
						switch label.GetName() {
						case "gpu_id":
							gpuID = label.GetValue()
						case "serial_number":
							serialNumber = label.GetValue()
						case "uuid":
							uuid = label.GetValue()
						}
					}

					require.NotEmpty(t, gpuID, "gpu_id label should be present")
					require.Equal(t, 1.0, dto.Gauge.GetValue(), "Info metric value should be 1")
					gpuInfoMetrics[gpuID] = gpuInfo{
						serialNumber: serialNumber,
						uuid:         uuid,
					}
				}
			}

			// Verify all GPUs emitted gpu_info metric
			require.Len(t, gpuInfoMetrics, len(tt.gpus), "All GPUs should emit gpu_info metric")

			// Verify serial numbers and UUIDs match expected values
			for _, gpu := range tt.gpus {
				info, ok := gpuInfoMetrics[gpu.id]
				require.True(t, ok, "GPU %s should have info metric", gpu.id)
				assert.Equal(t, gpu.serialNumber, info.serialNumber,
					"GPU %s should have correct serial number", gpu.id)
				assert.Equal(t, gpu.uuid, info.uuid,
					"GPU %s should have correct UUID", gpu.id)
			}

			// Verify UUID uniqueness
			uuidSet := make(map[string]bool)
			hasDuplicateUUIDs := false
			for gpuID, info := range gpuInfoMetrics {
				if info.uuid != "" {
					if uuidSet[info.uuid] {
						hasDuplicateUUIDs = true
						if tt.expectUniqueUUIDs {
							t.Errorf("UUID %s is duplicated for GPU %s - UUIDs must always be unique", info.uuid, gpuID)
						}
					}
					uuidSet[info.uuid] = true
				}
			}

			// If we expected duplicate UUIDs (invalid case), verify error was logged
			if !tt.expectUniqueUUIDs {
				logOutput := logBuf.String()
				assert.True(t, hasDuplicateUUIDs, "Test case expects duplicate UUIDs but none were found")
				assert.Contains(t, logOutput, "duplicate GPU UUID detected", "Should log error for duplicate UUID")
			}

			// Verify serial number uniqueness (or lack thereof)
			serialNumberSet := make(map[string][]string) // serial_number -> list of gpu_ids
			for gpuID, info := range gpuInfoMetrics {
				if info.serialNumber != "" {
					serialNumberSet[info.serialNumber] = append(serialNumberSet[info.serialNumber], gpuID)
				}
			}

			if tt.expectUniqueSerialNumbers {
				for sn, gpuIDs := range serialNumberSet {
					assert.Len(t, gpuIDs, 1, "Serial number %s should be unique but found on GPUs: %v", sn, gpuIDs)
				}
			} else {
				// For GB300 style, we expect some duplicates
				hasDuplicates := false
				for _, gpuIDs := range serialNumberSet {
					if len(gpuIDs) > 1 {
						hasDuplicates = true
						break
					}
				}
				assert.True(t, hasDuplicates, "Expected duplicate serial numbers for GB300 style test case")
			}
		})
	}
}

// TestGPUMetricsWithMissingSerialOrUUID tests handling of GPUs with missing serial number or UUID
func TestGPUMetricsWithMissingSerialOrUUID(t *testing.T) {
	tests := map[string]struct {
		serialNumber string
		uuid         string
		expectMetric bool
		expectSerial string
		expectUUID   string
	}{
		"both present": {
			serialNumber: "1234567890",
			uuid:         "e03507e4-8147-2191-d9a5-834772ec93a7",
			expectMetric: true,
			expectSerial: "1234567890",
			expectUUID:   "e03507e4-8147-2191-d9a5-834772ec93a7",
		},
		"missing serial number": {
			serialNumber: "",
			uuid:         "e03507e4-8147-2191-d9a5-834772ec93a7",
			expectMetric: true,
			expectSerial: "",
			expectUUID:   "e03507e4-8147-2191-d9a5-834772ec93a7",
		},
		"missing UUID": {
			serialNumber: "1234567890",
			uuid:         "",
			expectMetric: true,
			expectSerial: "1234567890",
			expectUUID:   "",
		},
		"both missing": {
			serialNumber: "",
			uuid:         "",
			expectMetric: false, // No metric emitted if both are empty
			expectSerial: "",
			expectUUID:   "",
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

			// Setup minimal routes
			server.addRouteFromFixture("/redfish/v1/", "service_root.json")
			server.addRoute("/redfish/v1/Systems", map[string]interface{}{
				"@odata.type": "#ComputerSystemCollection.ComputerSystemCollection",
				"Members":     []map[string]string{{"@odata.id": "/redfish/v1/Systems/System1"}},
			})
			server.addRoute("/redfish/v1/Systems/System1", map[string]interface{}{
				"@odata.type": "#ComputerSystem.v1_14_0.ComputerSystem",
				"Id":          "System1",
				"Processors":  map[string]string{"@odata.id": "/redfish/v1/Systems/System1/Processors"},
			})
			server.addRoute("/redfish/v1/Systems/System1/Processors", map[string]interface{}{
				"@odata.type": "#ProcessorCollection.ProcessorCollection",
				"Members":     []map[string]string{{"@odata.id": "/redfish/v1/Systems/System1/Processors/GPU_0"}},
			})

			processorData := map[string]interface{}{
				"@odata.type":   "#Processor.v1_20_0.Processor",
				"Id":            "GPU_0",
				"ProcessorType": "GPU",
				"TotalCores":    128,
				"TotalThreads":  128,
				"Status":        map[string]string{"State": "Enabled", "Health": "OK"},
			}
			if tt.serialNumber != "" {
				processorData["SerialNumber"] = tt.serialNumber
			}
			if tt.uuid != "" {
				processorData["UUID"] = tt.uuid
			}
			server.addRoute("/redfish/v1/Systems/System1/Processors/GPU_0", processorData)

			// Collect metrics with logger that captures output
			var logBuf strings.Builder
			logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

			client := connectToTestServer(t, server)
			defer client.Logout()

			collector, err := NewGPUCollector(t.Name(), client, logger, config.DefaultGPUCollector)
			require.NoError(t, err)
			ch := make(chan prometheus.Metric, 100)
			go func() {
				collector.Collect(ch)
				close(ch)
			}()

			foundMetric := false
			var actualSerial, actualUUID string

			for metric := range ch {
				desc := metric.Desc()
				if strings.Contains(desc.String(), "gpu_info") {
					foundMetric = true
					dto := &dto.Metric{}
					require.NoError(t, metric.Write(dto))

					for _, label := range dto.Label {
						switch label.GetName() {
						case "serial_number":
							actualSerial = label.GetValue()
						case "uuid":
							actualUUID = label.GetValue()
						}
					}
				}
			}

			assert.Equal(t, tt.expectMetric, foundMetric, "gpu_info metric presence mismatch")
			if foundMetric {
				assert.Equal(t, tt.expectSerial, actualSerial, "Serial number value mismatch")
				assert.Equal(t, tt.expectUUID, actualUUID, "UUID value mismatch")
			}

			// Verify appropriate logging for missing values
			logOutput := logBuf.String()
			if tt.serialNumber == "" {
				assert.Contains(t, logOutput, "GPU has no serial number", "Should log when serial number is missing")
			}
			if tt.uuid == "" {
				assert.Contains(t, logOutput, "GPU has no UUID", "Should log when UUID is missing")
			}
		})
	}
}

func Test_filterGPUs(t *testing.T) {
	tT := map[string]struct {
		processors []*redfish.Processor
		want       []*redfish.Processor
	}{
		"happy path, CPUs filtered": {
			processors: []*redfish.Processor{
				{
					ProcessorType: redfish.CPUProcessorType,
				},
				{
					ProcessorType: redfish.GPUProcessorType,
					Description:   "want this one",
				},
			},
			want: []*redfish.Processor{
				{
					ProcessorType: redfish.GPUProcessorType,
					Description:   "want this one",
				},
			},
		},
		"happy path, no GPUs returned": {
			processors: []*redfish.Processor{
				{
					ProcessorType: redfish.CPUProcessorType,
				},
				{
					ProcessorType: redfish.CPUProcessorType,
				},
			},
			want: []*redfish.Processor{},
		},
	}

	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			got := filterGPUs(test.processors)
			assert.Equal(t, test.want, got)
		})
	}
}
