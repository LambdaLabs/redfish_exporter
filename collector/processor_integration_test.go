package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessorMetricsIntegration(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*testRedfishServer)
		wantMetrics map[string]float64
		wantErr     bool
	}{
		{
			name: "processor with full PCIe and cache metrics from testdata",
			setupMock: func(m *testRedfishServer) {
				// Set up system and processor collection
				m.setupSystemWithProcessor("System1", "GPU_2")

				// Add processor and metrics from fixtures
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/GPU_2", "processor_with_metrics.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/GPU_2/ProcessorMetrics", "processor_metrics_full.json")
			},
			wantMetrics: map[string]float64{
				"pcie_l0_recovery":    42,
				"pcie_correctable":    10,
				"pcie_fatal":          1,
				"cache_correctable":   15,
				"cache_uncorrectable": 2,
			},
		},
		{
			name: "processor with zero error counts",
			setupMock: func(m *testRedfishServer) {
				// Set up system and processor collection
				m.setupSystemWithProcessor("System1", "CPU_0")

				// Add processor and metrics with zero errors
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU_0", "processor_zero_errors.json")
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU_0/ProcessorMetrics", "processor_metrics_zero.json")
			},
			wantMetrics: map[string]float64{
				"pcie_l0_recovery":    0,
				"pcie_correctable":    0,
				"pcie_fatal":          0,
				"cache_correctable":   0,
				"cache_uncorrectable": 0,
			},
		},
		{
			name: "processor without metrics link",
			setupMock: func(m *testRedfishServer) {
				// Set up system and processor collection
				m.setupSystemWithProcessor("System1", "CPU_1")

				// Add processor v1.0.0 (no Metrics field)
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU_1", "schemas/v1_0_0_processor.json")
			},
			wantMetrics: map[string]float64{
				// Should have no PCIe or cache metrics
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			server := newTestRedfishServer(t)
			tt.setupMock(server)

			client := connectToTestServer(t, server)

			// Act
			metrics := collectProcessorMetrics(t, client)

			// Assert
			for metricName, expectedValue := range tt.wantMetrics {
				actualValue, found := metrics[metricName]
				require.True(t, found, "Metric %s not found", metricName)
				assert.Equal(t, expectedValue, actualValue, "Metric %s has wrong value", metricName)
			}

			// Verify no unexpected metrics for the "without metrics" case
			if len(tt.wantMetrics) == 0 {
				assert.Empty(t, metrics, "Expected no PCIe or cache metrics when ProcessorMetrics unavailable")
			}
		})
	}
}

// TestProcessorMetricsBackwardsCompatibility tests different Redfish schema versions
func TestProcessorMetricsBackwardsCompatibility(t *testing.T) {
	tests := []struct {
		name          string
		schemaVersion string
		setupMock     func(*testRedfishServer)
		expectMetrics bool
	}{
		{
			name:          "Schema v1.0.0 - No ProcessorMetrics",
			schemaVersion: "1.0.0",
			setupMock: func(m *testRedfishServer) {
				// Processor.v1_0_0 doesn't have Metrics link
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU_1", "schemas/v1_0_0_processor.json")
			},
			expectMetrics: false,
		},
		{
			name:          "Schema v1.4.0 - ProcessorMetrics introduced",
			schemaVersion: "1.4.0",
			setupMock: func(m *testRedfishServer) {
				// Processor.v1_4_0 has Metrics link
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU_1", "schemas/v1_4_0_processor.json")
				// ProcessorMetrics.v1_0_0 has cache metrics but no PCIe errors yet
				m.addRouteFromFixture("/redfish/v1/Systems/System1/Processors/CPU_1/ProcessorMetrics", "schemas/v1_4_0_processor_metrics.json")
			},
			expectMetrics: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestRedfishServer(t)

			// Setup basic system structure
			server.setupSystemWithProcessor("System1", "CPU_1")

			// Apply version-specific setup
			tt.setupMock(server)

			client := connectToTestServer(t, server)
			metrics := collectProcessorMetrics(t, client)

			if tt.expectMetrics {
				assert.NotEmpty(t, metrics, "Expected metrics for schema %s", tt.schemaVersion)
			} else {
				assert.Empty(t, metrics, "Expected no metrics for schema %s", tt.schemaVersion)
			}
		})
	}
}
