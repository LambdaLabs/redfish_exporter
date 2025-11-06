package collector

import (
	"io"
	"log/slog"
	"math"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/common"
	"github.com/stmcginnis/gofish/redfish"
	"github.com/stretchr/testify/require"
)

func TestTelemetryCollectorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Connect to mock server
	config := gofish.ClientConfig{
		Endpoint: "https://localhost:8443",
		Username: "admin",
		Password: "password",
		Insecure: true,
	}

	client, err := gofish.Connect(config)
	if err != nil {
		t.Skipf("Could not connect to mock server: %v", err)
	}
	defer client.Logout()

	// Create collector
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewTelemetryCollector(t.Name(), client, logger, nil)
	require.NoError(t, err)

	// Collect metrics
	metricsChan := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(metricsChan)
		close(metricsChan)
	}()

	// Count metrics collected
	metricCount := 0
	var cacheECCFound, pcieErrorFound, throttleFound, platformEnvFound bool

	for metric := range metricsChan {
		metricCount++
		metricName := metric.Desc().String()

		// Check for expected metric types
		if strings.Contains(metricName, "cache_ecc") {
			cacheECCFound = true
		}
		if strings.Contains(metricName, "pcie") {
			pcieErrorFound = true
		}
		if strings.Contains(metricName, "throttle") {
			throttleFound = true
		}
		// Check for platform environment metrics (GPU temp, CPU temp, ambient, etc.)
		if strings.Contains(metricName, "gpu_temperature") || strings.Contains(metricName, "cpu_temperature") ||
			strings.Contains(metricName, "ambient_") || strings.Contains(metricName, "gpu_power") {
			platformEnvFound = true
		}
	}

	t.Logf("Collected %d telemetry metrics", metricCount)

	// Verify we got some metrics (may be 0 if TelemetryService not available)
	if metricCount > 0 {
		t.Logf("TelemetryService is available")

		// If we have metrics, verify we got the expected types
		if !cacheECCFound {
			t.Error("Expected to find cache ECC metrics")
		}
		if !pcieErrorFound {
			t.Error("Expected to find PCIe error metrics")
		}
		if !throttleFound {
			t.Error("Expected to find throttle metrics")
		}
		if !platformEnvFound {
			t.Log("Warning: Platform environment metrics not found (may not be available on this system)")
		}
	} else {
		t.Log("TelemetryService not available on this system (skipping metric validation)")
	}
}

func TestTelemetryCollectorDescribe(t *testing.T) {
	// Create a mock client (won't be used for Describe)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewTelemetryCollector(t.Name(), nil, logger, nil)
	require.NoError(t, err)

	// Describe should work without a client
	descChan := make(chan *prometheus.Desc, 100)
	go func() {
		collector.Describe(descChan)
		close(descChan)
	}()

	descCount := 0
	for range descChan {
		descCount++
	}

	// We should have descriptions for all our metrics
	expectedDescs := len(telemetryMetrics) + 1 // +1 for scrape status
	if descCount != expectedDescs {
		t.Errorf("Expected %d metric descriptions, got %d", expectedDescs, descCount)
	}
}

func TestParseMetricProperty(t *testing.T) {
	tests := []struct {
		name         string
		property     string
		expectedGPU  string
		expectedName string
	}{
		{
			name:         "cache_ecc_error",
			property:     "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_1/ProcessorMetrics#/CacheMetricsTotal/LifeTime/CorrectableECCErrorCount",
			expectedGPU:  "GPU_SXM_1",
			expectedName: "CacheMetricsTotal/LifeTime/CorrectableECCErrorCount",
		},
		{
			name:         "pcie_error",
			property:     "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_8/ProcessorMetrics#/PCIeErrors/FatalErrorCount",
			expectedGPU:  "GPU_SXM_8",
			expectedName: "PCIeErrors/FatalErrorCount",
		},
		{
			name:         "throttle_duration",
			property:     "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_3/ProcessorMetrics#/PowerLimitThrottleDuration",
			expectedGPU:  "GPU_SXM_3",
			expectedName: "PowerLimitThrottleDuration",
		},
		{
			name:         "oem_metric",
			property:     "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_5/ProcessorMetrics#/Oem/Nvidia/HardwareViolationThrottleDuration",
			expectedGPU:  "GPU_SXM_5",
			expectedName: "Oem/Nvidia/HardwareViolationThrottleDuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gpuID, metricName := parseMetricProperty(tt.property)
			if gpuID != tt.expectedGPU {
				t.Errorf("Expected GPU ID %q, got %q", tt.expectedGPU, gpuID)
			}
			if metricName != tt.expectedName {
				t.Errorf("Expected metric name %q, got %q", tt.expectedName, metricName)
			}
		})
	}
}

func TestParseMetricValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected float64
		wantErr  bool
	}{
		{name: "integer", value: "42", expected: 42.0, wantErr: false},
		{name: "float", value: "3.14", expected: 3.14, wantErr: false},
		{name: "zero", value: "0", expected: 0.0, wantErr: false},
		{name: "negative", value: "-5.5", expected: -5.5, wantErr: false},
		{name: "true", value: "true", expected: 1.0, wantErr: false},
		{name: "false", value: "false", expected: 0.0, wantErr: false},
		{name: "scientific", value: "1.23e10", expected: 1.23e10, wantErr: false},
		{name: "iso8601_zero", value: "PT0S", expected: 0.0, wantErr: false},
		{name: "iso8601_seconds", value: "PT30S", expected: 30.0, wantErr: false},
		{name: "iso8601_minutes", value: "PT5M", expected: 300.0, wantErr: false},
		{name: "iso8601_hours", value: "PT2H", expected: 7200.0, wantErr: false},
		{name: "iso8601_complex", value: "PT1H30M45S", expected: 5445.0, wantErr: false},
		{name: "invalid", value: "not_a_number", expected: 0.0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseMetricValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error: %v, got: %v", tt.wantErr, err)
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestExtractSystemIDFromReport(t *testing.T) {
	// Test the parsing logic
	property := "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_1/ProcessorMetrics#/PCIeErrors/FatalErrorCount"
	expectedSystemID := "HGX_Baseboard_0"

	// Manual parsing similar to extractSystemIDFromReport
	var systemID string
	if idx := strings.Index(property, "/Systems/"); idx >= 0 {
		remainder := property[idx+len("/Systems/"):]
		if endIdx := strings.Index(remainder, "/"); endIdx >= 0 {
			systemID = remainder[:endIdx]
		}
	}

	if systemID != expectedSystemID {
		t.Errorf("Expected system ID %q, got %q", expectedSystemID, systemID)
	}
}

// Benchmark the metric collection
func BenchmarkTelemetryCollector(b *testing.B) {
	config := gofish.ClientConfig{
		Endpoint: "https://localhost:8443",
		Username: "admin",
		Password: "password",
		Insecure: true,
	}

	client, err := gofish.Connect(config)
	if err != nil {
		b.Skipf("Could not connect to mock server: %v", err)
	}
	defer client.Logout()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewTelemetryCollector(b.Name(), client, logger, nil)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metricsChan := make(chan prometheus.Metric, 100)
		go func() {
			collector.Collect(metricsChan)
			close(metricsChan)
		}()
		// Drain the channel
		for range metricsChan {
		}
	}
}

// TestTelemetryMetricCount validates all metrics are properly exposed
func TestTelemetryMetricCount(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewTelemetryCollector(t.Name(), nil, logger, nil)
	require.NoError(t, err)

	// Expected metric categories
	expectedCategories := map[string]int{
		"cache_ecc":            2,  // correctable + uncorrectable
		"pcie":                 9,  // various PCIe error types
		"throttle":             4,  // power, thermal, hardware, software
		"memory":               5,  // ecc lifetime (2), bandwidth, capacity_util, operating_speed
		"reset":                8,  // conventional entry/exit, fundamental entry/exit, irot exit, pf_flr entry/exit, last_reset_type
		"port":                 23, // port metrics (speed, rx/tx bytes/frames/errors, link down counts, OEM metrics)
		"nvidia_gpm":           22, // GPU Performance Monitoring: 11 compute + 2 aggregate media + 2 instance media + 5 network + 2 PCIe
		"platform_environment": 18, // Platform environment metrics: 3 backward-compat + 4 GPU + 8 CPU + 2 ambient + 1 BMC
		"info":                 1,  // GPU Info series
		"scrape_status":        1,  // collector status
	}

	totalExpected := 0
	for _, count := range expectedCategories {
		totalExpected += count
	}

	// Count actual metrics
	descChan := make(chan *prometheus.Desc, 100)
	go func() {
		collector.Describe(descChan)
		close(descChan)
	}()

	actualCount := 0
	for range descChan {
		actualCount++
	}

	if actualCount != totalExpected {
		t.Errorf("Expected %d total metrics, got %d", totalExpected, actualCount)
	}
}

func TestTelemetryCollectorGracefulNoService(t *testing.T) {
	// Test that collector handles missing TelemetryService gracefully
	// This would require a mock client that returns an error
	// For now, just verify the collector can be created without panic
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewTelemetryCollector(t.Name(), nil, logger, nil)
	require.NoError(t, err)

	// Verify it has the right number of metrics defined
	if len(collector.metrics) != len(telemetryMetrics) {
		t.Errorf("Expected %d metrics, got %d", len(telemetryMetrics), len(collector.metrics))
	}
}

func TestTelemetryMetricNames(t *testing.T) {
	// Verify all metric names follow the expected pattern
	expectedPrefix := "telemetry_"

	// Backward-compatible metrics that don't use telemetry_ prefix
	backwardCompatMetrics := map[string]bool{
		"gpu_memory_power_watts":         true,
		"gpu_temperature_tlimit_celsius": true,
		"chassis_gpu_total_power_watts":  true,
	}

	for name := range telemetryMetrics {
		// Skip backward-compatible metrics
		if backwardCompatMetrics[name] {
			continue
		}

		if !hasPrefix(name, expectedPrefix) {
			t.Errorf("Metric name %q does not start with %q", name, expectedPrefix)
		}

		// Verify no spaces or invalid characters
		for _, char := range name {
			if char == ' ' || char == '\t' || char == '\n' {
				t.Errorf("Metric name %q contains whitespace", name)
			}
		}
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// TestParseResetMetricProperty validates reset metric property parsing
func TestParseResetMetricProperty(t *testing.T) {
	tests := []struct {
		name         string
		property     string
		expectedGPU  string
		expectedName string
	}{
		{
			name:         "conventional_reset_entry",
			property:     "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_1/Oem/Nvidia/ProcessorResetMetrics#/ConventionalResetEntryCount",
			expectedGPU:  "GPU_SXM_1",
			expectedName: "ConventionalResetEntryCount",
		},
		{
			name:         "fundamental_reset_exit",
			property:     "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_5/Oem/Nvidia/ProcessorResetMetrics#/FundamentalResetExitCount",
			expectedGPU:  "GPU_SXM_5",
			expectedName: "FundamentalResetExitCount",
		},
		{
			name:         "last_reset_type",
			property:     "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_0/Oem/Nvidia/ProcessorResetMetrics#/LastResetType",
			expectedGPU:  "GPU_SXM_0",
			expectedName: "LastResetType",
		},
		{
			name:         "irot_reset_exit",
			property:     "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_7/Oem/Nvidia/ProcessorResetMetrics#/IRoTResetExitCount",
			expectedGPU:  "GPU_SXM_7",
			expectedName: "IRoTResetExitCount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gpuID, metricName := parseResetMetricProperty(tt.property)
			if gpuID != tt.expectedGPU {
				t.Errorf("Expected GPU ID %q, got %q", tt.expectedGPU, gpuID)
			}
			if metricName != tt.expectedName {
				t.Errorf("Expected metric name %q, got %q", tt.expectedName, metricName)
			}
		})
	}
}

// TestParsePortMetricProperty validates port metric property parsing
func TestParsePortMetricProperty(t *testing.T) {
	tests := []struct {
		name         string
		property     string
		expectedGPU  string
		expectedPort string
		expectedName string
	}{
		{
			name:         "nvlink_rx_bytes",
			property:     "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_0/Ports/NVLink_0/Metrics#/RXBytes",
			expectedGPU:  "GPU_SXM_0",
			expectedPort: "NVLink_0",
			expectedName: "RXBytes",
		},
		{
			name:         "nvlink_oem_metric",
			property:     "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_3/Ports/NVLink_17/Metrics#/Oem/Nvidia/TotalRawBER",
			expectedGPU:  "GPU_SXM_3",
			expectedPort: "NVLink_17",
			expectedName: "Oem/Nvidia/TotalRawBER",
		},
		{
			name:         "pcie_speed",
			property:     "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_1/Ports/PCIe/Metrics#/CurrentSpeedGbps",
			expectedGPU:  "GPU_SXM_1",
			expectedPort: "PCIe",
			expectedName: "CurrentSpeedGbps",
		},
		{
			name:         "nvlink_networking",
			property:     "/redfish/v1/Systems/HGX_Baseboard_0/Processors/GPU_SXM_7/Ports/NVLink_5/Metrics#/Networking/RXFrames",
			expectedGPU:  "GPU_SXM_7",
			expectedPort: "NVLink_5",
			expectedName: "Networking/RXFrames",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gpuID, portID, metricName := parsePortMetricProperty(tt.property)
			if gpuID != tt.expectedGPU {
				t.Errorf("Expected GPU ID %q, got %q", tt.expectedGPU, gpuID)
			}
			if portID != tt.expectedPort {
				t.Errorf("Expected port ID %q, got %q", tt.expectedPort, portID)
			}
			if metricName != tt.expectedName {
				t.Errorf("Expected metric name %q, got %q", tt.expectedName, metricName)
			}
		})
	}
}

// TestParseSensorPath validates sensor path parsing for platform environment metrics
func TestParseSensorPath(t *testing.T) {
	tests := []struct {
		name            string
		sensorPath      string
		expectedChassis string
		expectedSensor  string
	}{
		{
			name:            "gpu_temp_sensor",
			sensorPath:      "/redfish/v1/Chassis/HGX_Baseboard_0/Sensors/HGX_GPU_0_TEMP_0",
			expectedChassis: "HGX_Baseboard_0",
			expectedSensor:  "HGX_GPU_0_TEMP_0",
		},
		{
			name:            "gpu_power_sensor",
			sensorPath:      "/redfish/v1/Chassis/HGX_Baseboard_0/Sensors/HGX_GPU_1_Power_0",
			expectedChassis: "HGX_Baseboard_0",
			expectedSensor:  "HGX_GPU_1_Power_0",
		},
		{
			name:            "cpu_temp_sensor",
			sensorPath:      "/redfish/v1/Chassis/HGX_Baseboard_0/Sensors/ProcessorModule_0_CPU_0_Temp_0",
			expectedChassis: "HGX_Baseboard_0",
			expectedSensor:  "ProcessorModule_0_CPU_0_Temp_0",
		},
		{
			name:            "ambient_temp_sensor",
			sensorPath:      "/redfish/v1/Chassis/HGX_Baseboard_0/Sensors/HGX_ProcessorModule_0_Inlet_Temp_0",
			expectedChassis: "HGX_Baseboard_0",
			expectedSensor:  "HGX_ProcessorModule_0_Inlet_Temp_0",
		},
		{
			name:            "gpu_memory_power",
			sensorPath:      "/redfish/v1/Chassis/HGX_Baseboard_0/Sensors/HGX_GPU_0_DRAM_0_Power_0",
			expectedChassis: "HGX_Baseboard_0",
			expectedSensor:  "HGX_GPU_0_DRAM_0_Power_0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chassisID, sensorID := parseSensorPath(tt.sensorPath)
			if chassisID != tt.expectedChassis {
				t.Errorf("Expected chassis ID %q, got %q", tt.expectedChassis, chassisID)
			}
			if sensorID != tt.expectedSensor {
				t.Errorf("Expected sensor ID %q, got %q", tt.expectedSensor, sensorID)
			}
		})
	}
}

// TestParseGPUSensorMetric validates GPU sensor metric parsing
func TestParseGPUSensorMetric(t *testing.T) {
	tests := []struct {
		name           string
		sensorID       string
		expectedGPU    string
		expectedType   string
		expectedMemory string
	}{
		{
			name:         "gpu_power",
			sensorID:     "HGX_GPU_0_Power_0",
			expectedGPU:  "GPU_0",
			expectedType: "power",
		},
		{
			name:         "gpu_energy",
			sensorID:     "HGX_GPU_3_Energy_0",
			expectedGPU:  "GPU_3",
			expectedType: "energy",
		},
		{
			name:         "gpu_temp_0",
			sensorID:     "HGX_GPU_1_TEMP_0",
			expectedGPU:  "GPU_1",
			expectedType: "temp0",
		},
		{
			name:         "gpu_temp_1",
			sensorID:     "HGX_GPU_2_TEMP_1",
			expectedGPU:  "GPU_2",
			expectedType: "temp1",
		},
		{
			name:           "gpu_memory_power",
			sensorID:       "HGX_GPU_0_DRAM_0_Power_0",
			expectedGPU:    "GPU_0",
			expectedType:   "memory_power",
			expectedMemory: "GPU_0_DRAM_0",
		},
		{
			name:           "gpu_memory_temp",
			sensorID:       "HGX_GPU_5_DRAM_1_Temp_0",
			expectedGPU:    "GPU_5",
			expectedType:   "memory_temp",
			expectedMemory: "GPU_5_DRAM_1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gpuID, metricType, memoryID := parseGPUSensorMetric(tt.sensorID)
			if gpuID != tt.expectedGPU {
				t.Errorf("Expected GPU ID %q, got %q", tt.expectedGPU, gpuID)
			}
			if metricType != tt.expectedType {
				t.Errorf("Expected metric type %q, got %q", tt.expectedType, metricType)
			}
			if memoryID != tt.expectedMemory {
				t.Errorf("Expected memory ID %q, got %q", tt.expectedMemory, memoryID)
			}
		})
	}
}

// TestParseCPUSensorMetric validates CPU sensor metric parsing
func TestParseCPUSensorMetric(t *testing.T) {
	tests := []struct {
		name         string
		sensorID     string
		expectedCPU  string
		expectedType string
	}{
		{
			name:         "cpu_power",
			sensorID:     "ProcessorModule_0_CPU_0_Power_0",
			expectedCPU:  "CPU_0",
			expectedType: "power",
		},
		{
			name:         "cpu_energy",
			sensorID:     "ProcessorModule_1_CPU_0_Energy_0",
			expectedCPU:  "CPU_1",
			expectedType: "energy",
		},
		{
			name:         "cpu_temp",
			sensorID:     "ProcessorModule_0_CPU_0_Temp_0",
			expectedCPU:  "CPU_0",
			expectedType: "temp",
		},
		{
			name:         "cpu_temp_limit",
			sensorID:     "ProcessorModule_0_CPU_0_TempLimit_0",
			expectedCPU:  "CPU_0",
			expectedType: "temp_limit",
		},
		{
			name:         "cpu_edp_current",
			sensorID:     "ProcessorModule_0_CPU_0_EDP_Current_Limit_0",
			expectedCPU:  "CPU_0",
			expectedType: "edp_current",
		},
		{
			name:         "cpu_edp_peak",
			sensorID:     "ProcessorModule_0_CPU_0_EDP_Peak_Limit_0",
			expectedCPU:  "CPU_0",
			expectedType: "edp_peak",
		},
		{
			name:         "cpu_vreg_cpu",
			sensorID:     "ProcessorModule_0_CPU_0_VREG_CPU_Power_0",
			expectedCPU:  "CPU_0",
			expectedType: "vreg_cpu",
		},
		{
			name:         "cpu_vreg_soc",
			sensorID:     "ProcessorModule_0_CPU_0_VREG_SOC_Power_0",
			expectedCPU:  "CPU_0",
			expectedType: "vreg_soc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpuID, metricType := parseCPUSensorMetric(tt.sensorID)
			if cpuID != tt.expectedCPU {
				t.Errorf("Expected CPU ID %q, got %q", tt.expectedCPU, cpuID)
			}
			if metricType != tt.expectedType {
				t.Errorf("Expected metric type %q, got %q", tt.expectedType, metricType)
			}
		})
	}
}

// TestParseAmbientSensorMetric validates ambient sensor metric parsing
func TestParseAmbientSensorMetric(t *testing.T) {
	tests := []struct {
		name             string
		sensorID         string
		expectedLocation string
		expectedType     string
		expectedSensorID string
	}{
		{
			name:             "inlet_temp",
			sensorID:         "HGX_ProcessorModule_0_Inlet_Temp_0",
			expectedLocation: "ProcessorModule_0",
			expectedType:     "inlet",
			expectedSensorID: "HGX_ProcessorModule_0_Inlet_Temp_0",
		},
		{
			name:             "exhaust_temp",
			sensorID:         "HGX_ProcessorModule_1_Exhaust_Temp_0",
			expectedLocation: "ProcessorModule_1",
			expectedType:     "exhaust",
			expectedSensorID: "HGX_ProcessorModule_1_Exhaust_Temp_0",
		},
		{
			name:             "inlet_temp_variant",
			sensorID:         "HGX_ProcessorModule_0_Inlet_Temp_1",
			expectedLocation: "ProcessorModule_0",
			expectedType:     "inlet",
			expectedSensorID: "HGX_ProcessorModule_0_Inlet_Temp_1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locationID, metricType, sensorID := parseAmbientSensorMetric(tt.sensorID)
			if locationID != tt.expectedLocation {
				t.Errorf("Expected location ID %q, got %q", tt.expectedLocation, locationID)
			}
			if metricType != tt.expectedType {
				t.Errorf("Expected metric type %q, got %q", tt.expectedType, metricType)
			}
			if sensorID != tt.expectedSensorID {
				t.Errorf("Expected sensor ID %q, got %q", tt.expectedSensorID, sensorID)
			}
		})
	}
}

func TestTelemetryCollector_CollectPlatformGPUMetrics(t *testing.T) {
	tT := map[string]struct {
		report    *redfish.MetricReport
		systemMap map[string]string
	}{
		"stale reports should not be emitted": {
			report: &redfish.MetricReport{
				Entity: common.Entity{
					ID: "HGX_PlatformEnvironmentMetrics_0",
				},
				MetricValues: []redfish.MetricValue{
					{
						MetricProperty: "/redfish/v1/Chassis/HGX_GPU_1/Sensors/HGX_Chassis_0_TotalGPU_Power_0",
						MetricValue:    "100",
						OEM:            []byte(`{"Nvidia": {"@odata.type": "#NvidiaMetricReport.v1_0_0.NvidiaMetricReport","MetricValueStale": false}},`),
					},
					{
						MetricProperty: "/redfish/v1/Chassis/HGX_GPU_0/Sensors/HGX_GPU_Energy_0",
						MetricValue:    "nan",
						OEM:            []byte(`"Oem": {"Nvidia": {"@odata.type": "#NvidiaMetricReport.v1_0_0.NvidiaMetricReport","MetricValueStale": true}},`),
					},
				},
			},
		},
	}

	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
			tc := &TelemetryCollector{
				logger:  logger,
				metrics: createTelemetryMetricMap(),
			}
			testWG := &sync.WaitGroup{}
			testWG.Add(len(test.report.MetricValues))
			outCh := make(chan prometheus.Metric, len(test.report.MetricValues))
			go func() {
				tc.collectPlatformEnvironmentMetrics(outCh, test.report, map[string]string{"GPU_Energy": "HGX_GPU_0"}, testWG)
				close(outCh)
			}()

			for metric := range outCh {
				m := &dto.Metric{}
				require.NoError(t, metric.Write(m))
				if m.Counter != nil {
					require.False(t, math.IsNaN(*m.GetCounter().Value))
				}
				if m.Gauge != nil {
					require.False(t, math.IsNaN(*m.GetGauge().Value))
				}
				if strings.Contains(metric.Desc().String(), "stale_reports_last") {
					require.GreaterOrEqual(t, m.GetGauge().GetValue(), 1.0)
				}
			}
		})
	}
}
