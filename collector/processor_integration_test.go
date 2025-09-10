package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessorMetricsIntegration(t *testing.T) {
	catalog := NewTestDataCatalog(t)
	
	tests := []struct {
		name        string
		setupMock   func(*mockRedfishServer)
		wantMetrics map[string]float64
		wantErr     bool
	}{
		{
			name: "processor with full PCIe and cache metrics from testdata",
			setupMock: func(m *mockRedfishServer) {
				// Load test data from files
				processor := catalog.ProcessorWithMetrics()
				metrics := catalog.ProcessorMetricsFull()
				
				// Set up mock server with loaded data
				m.responses["/redfish/v1/Systems/System1"] = map[string]interface{}{
					"@odata.type": "#ComputerSystem.v1_20_0.ComputerSystem",
					"@odata.id":   "/redfish/v1/Systems/System1",
					"Id":          "System1",
					"Name":        "Test System",
					"Status": map[string]string{
						"State":  "Enabled",
						"Health": "OK",
					},
					"Processors": map[string]string{
						"@odata.id": "/redfish/v1/Systems/System1/Processors",
					},
				}
				m.responses["/redfish/v1/Systems/System1/Processors"] = map[string]interface{}{
					"@odata.type": "#ProcessorCollection.ProcessorCollection",
					"Members": []map[string]string{
						{"@odata.id": "/redfish/v1/Systems/System1/Processors/GPU_2"},
					},
				}
				m.responses["/redfish/v1/Systems/System1/Processors/GPU_2"] = processor
				m.responses["/redfish/v1/Systems/System1/Processors/GPU_2/ProcessorMetrics"] = metrics
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
			setupMock: func(m *mockRedfishServer) {
				m.addProcessorWithMetrics(
					"System1",
					"CPU_0",
					map[string]int{
						"L0ToRecoveryCount":      0,
						"CorrectableErrorCount":  0,
						"FatalErrorCount":        0,
						"NonFatalErrorCount":     0,
						"NAKReceivedCount":       0,
						"NAKSentCount":           0,
						"ReplayCount":            0,
						"ReplayRolloverCount":    0,
					},
					map[string]int{
						"CorrectableECCErrorCount":   0,
						"UncorrectableECCErrorCount": 0,
					},
				)
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
			setupMock: func(m *mockRedfishServer) {
				// Add processor without Metrics field
				m.responses["/redfish/v1/Systems/System1"] = map[string]interface{}{
					"@odata.type": "#ComputerSystem.v1_20_0.ComputerSystem",
					"@odata.id":   "/redfish/v1/Systems/System1",
					"Id":          "System1",
					"Name":        "Test System",
					"Status": map[string]string{
						"State":  "Enabled",
						"Health": "OK",
					},
					"Processors": map[string]string{
						"@odata.id": "/redfish/v1/Systems/System1/Processors",
					},
				}
				
				m.responses["/redfish/v1/Systems/System1/Processors"] = map[string]interface{}{
					"@odata.type": "#ProcessorCollection.ProcessorCollection",
					"Members": []map[string]string{
						{"@odata.id": "/redfish/v1/Systems/System1/Processors/CPU_1"},
					},
				}
				
				m.responses["/redfish/v1/Systems/System1/Processors/CPU_1"] = map[string]interface{}{
					"@odata.type":  "#Processor.v1_0_0.Processor", // Old version without Metrics
					"@odata.id":    "/redfish/v1/Systems/System1/Processors/CPU_1",
					"Id":           "CPU_1",
					"Name":         "CPU without metrics",
					"TotalCores":   8,
					"TotalThreads": 16,
					"Status": map[string]string{
						"State":  "Enabled",
						"Health": "OK",
					},
					// No Metrics field
				}
			},
			wantMetrics: map[string]float64{
				// Should have no PCIe or cache metrics
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			server := newMockRedfishServer(t)
			tt.setupMock(server)
			
			client := connectToMockServer(t, server)
			
			// Act
			metrics := collectProcessorMetrics(t, client)
			
			// Assert
			for metricName, expectedValue := range tt.wantMetrics {
				actualValue, found := metrics[metricName]
				require.True(t, found, "Metric %s not found", metricName)
				assert.Equal(t, expectedValue, actualValue, "Metric %s has wrong value", metricName)
			}
			
			// Verify no unexpected metrics for the "without metrics" case
			if tt.name == "processor without metrics link" {
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
		setupMock     func(*mockRedfishServer)
		expectMetrics bool
	}{
		{
			name:          "Schema v1.0.0 - No ProcessorMetrics",
			schemaVersion: "1.0.0",
			setupMock: func(m *mockRedfishServer) {
				// Processor.v1_0_0 doesn't have Metrics link
				m.responses["/redfish/v1/Systems/System1/Processors/CPU_1"] = map[string]interface{}{
					"@odata.type": "#Processor.v1_0_0.Processor",
					"Id":          "CPU_1",
					"Name":        "Legacy CPU",
					"Status": map[string]string{
						"State":  "Enabled",
						"Health": "OK",
					},
				}
			},
			expectMetrics: false,
		},
		{
			name:          "Schema v1.4.0 - ProcessorMetrics introduced",
			schemaVersion: "1.4.0",
			setupMock: func(m *mockRedfishServer) {
				// Processor.v1_4_0 has Metrics link
				m.responses["/redfish/v1/Systems/System1/Processors/CPU_1"] = map[string]interface{}{
					"@odata.type": "#Processor.v1_4_0.Processor",
					"Id":          "CPU_1",
					"Name":        "CPU with basic metrics",
					"Metrics": map[string]string{
						"@odata.id": "/redfish/v1/Systems/System1/Processors/CPU_1/ProcessorMetrics",
					},
				}
				// ProcessorMetrics.v1_0_0 has cache metrics but no PCIe errors yet
				m.responses["/redfish/v1/Systems/System1/Processors/CPU_1/ProcessorMetrics"] = map[string]interface{}{
					"@odata.type": "#ProcessorMetrics.v1_0_0.ProcessorMetrics",
					"CacheMetricsTotal": map[string]interface{}{
						"LifeTime": map[string]int{
							"CorrectableECCErrorCount":   5,
							"UncorrectableECCErrorCount": 1,
						},
					},
					// No PCIeErrors in v1.0.0
				}
			},
			expectMetrics: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newMockRedfishServer(t)
			
			// Setup basic system structure
			server.responses["/redfish/v1/Systems/System1"] = map[string]interface{}{
				"@odata.type": "#ComputerSystem.v1_0_0.ComputerSystem",
				"Id":          "System1",
				"Processors": map[string]string{
					"@odata.id": "/redfish/v1/Systems/System1/Processors",
				},
			}
			server.responses["/redfish/v1/Systems/System1/Processors"] = map[string]interface{}{
				"Members": []map[string]string{
					{"@odata.id": "/redfish/v1/Systems/System1/Processors/CPU_1"},
				},
			}
			
			// Apply version-specific setup
			tt.setupMock(server)
			
			client := connectToMockServer(t, server)
			metrics := collectProcessorMetrics(t, client)
			
			if tt.expectMetrics {
				assert.NotEmpty(t, metrics, "Expected metrics for schema %s", tt.schemaVersion)
			} else {
				assert.Empty(t, metrics, "Expected no metrics for schema %s", tt.schemaVersion)
			}
		})
	}
}