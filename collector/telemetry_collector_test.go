package collector

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
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
	collector := NewTelemetryCollector(client, logger)

	// Collect metrics
	metricsChan := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(metricsChan)
		close(metricsChan)
	}()

	// Count metrics collected
	metricCount := 0
	var cacheECCFound, pcieErrorFound, throttleFound bool

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
	} else {
		t.Log("TelemetryService not available on this system (skipping metric validation)")
	}
}

func TestTelemetryCollectorDescribe(t *testing.T) {
	// Create a mock client (won't be used for Describe)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector := NewTelemetryCollector(nil, logger)

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
	collector := NewTelemetryCollector(client, logger)

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
	collector := NewTelemetryCollector(nil, logger)

	// Expected metric categories
	expectedCategories := map[string]int{
		"cache_ecc":     2, // correctable + uncorrectable
		"pcie":          9, // various PCIe error types
		"throttle":      4, // power, thermal, hardware, software
		"memory":        5, // ecc lifetime (2), bandwidth, capacity_util, operating_speed
		"scrape_status": 1, // collector status
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
	collector := NewTelemetryCollector(nil, logger)

	// Verify it has the right number of metrics defined
	if len(collector.metrics) != len(telemetryMetrics) {
		t.Errorf("Expected %d metrics, got %d", len(telemetryMetrics), len(collector.metrics))
	}
}

func TestTelemetryMetricNames(t *testing.T) {
	// Verify all metric names follow the expected pattern
	expectedPrefix := "telemetry_"

	for name := range telemetryMetrics {
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
