package collector

import (
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stmcginnis/gofish/common"
	"github.com/stmcginnis/gofish/redfish"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseDriveControllerMapping tests the drive-to-controller mapping metric creation
func TestParseDriveControllerMapping(t *testing.T) {
	ch := make(chan prometheus.Metric, 10)
	
	// Create a test drive
	testDrive := &redfish.Drive{
		Entity: common.Entity{
			ID:   "Disk.Bay.0",
			Name: "Drive 0",
		},
	}
	
	// Test the mapping metric creation
	parseDriveControllerMapping(ch, "test-host", testDrive, "RAID.Slot.1")
	
	// Verify metric was created
	assert.Equal(t, 1, len(ch), "Expected one metric to be created")
	
	// Verify metric details
	metric := <-ch
	metricDTO := &dto.Metric{}
	err := metric.Write(metricDTO)
	require.NoError(t, err)
	
	// Check labels
	labels := metricDTO.GetLabel()
	assert.Equal(t, 4, len(labels), "Expected 4 labels")
	
	labelMap := make(map[string]string)
	for _, label := range labels {
		labelMap[*label.Name] = *label.Value
	}
	
	assert.Equal(t, "test-host", labelMap["hostname"])
	assert.Equal(t, "Disk.Bay.0", labelMap["drive_id"])
	assert.Equal(t, "Drive 0", labelMap["drive_name"])
	assert.Equal(t, "RAID.Slot.1", labelMap["storage_controller_id"])
	
	// Check value (should always be 1)
	assert.Equal(t, float64(1), metricDTO.Gauge.GetValue())
}

// Common test structure for drive-related tests
type driveTestCase struct {
	name                    string
	drives                  []driveScenario
	expectedUniqueDrives    int
	expectedMappingMetrics  int
	expectedDriveMetrics    int // state + health + capacity per unique drive
	expectedControllerMap   map[string]int // driveID -> controller count
	validateMetrics         bool // whether to validate individual metric values
}

type driveScenario struct {
	driveID      string
	driveName    string
	controllerID string
	capacity     int64
	state        string
	health       string
}

// TestDriveMetrics tests both individual drive parsing and deduplication scenarios
func TestDriveMetrics(t *testing.T) {
	tests := []driveTestCase{
		{
			name: "single healthy drive",
			drives: []driveScenario{
				{
					driveID:      "Disk.Bay.0",
					driveName:    "Drive 0",
					controllerID: "RAID.Slot.1",
					capacity:     1000000000000,
					state:        "Enabled",
					health:       "OK",
				},
			},
			expectedUniqueDrives:   1,
			expectedMappingMetrics: 1,
			expectedDriveMetrics:   3,
			expectedControllerMap:  map[string]int{"Disk.Bay.0": 1},
			validateMetrics:        true,
		},
		{
			name: "drive with warning in single controller",
			drives: []driveScenario{
				{
					driveID:      "Disk.Bay.1",
					driveName:    "Drive 1",
					controllerID: "RAID.Slot.1",
					capacity:     2000000000000,
					state:        "Enabled",
					health:       "Warning",
				},
			},
			expectedUniqueDrives:   1,
			expectedMappingMetrics: 1,
			expectedDriveMetrics:   3,
			expectedControllerMap:  map[string]int{"Disk.Bay.1": 1},
			validateMetrics:        true,
		},
		{
			name: "critical drive",
			drives: []driveScenario{
				{
					driveID:      "Disk.Bay.2",
					driveName:    "Drive 2",
					controllerID: "RAID.Slot.1",
					capacity:     500000000000,
					state:        "Disabled",
					health:       "Critical",
				},
			},
			expectedUniqueDrives:   1,
			expectedMappingMetrics: 1,
			expectedDriveMetrics:   3,
			expectedControllerMap:  map[string]int{"Disk.Bay.2": 1},
			validateMetrics:        true,
		},
		{
			name: "duplicate drive in two controllers",
			drives: []driveScenario{
				{
					driveID:      "Disk.Bay.0",
					driveName:    "Drive 0",
					controllerID: "RAID.Slot.1",
					capacity:     1000000000000,
					state:        "Enabled",
					health:       "OK",
				},
				{
					driveID:      "Disk.Bay.0",
					driveName:    "Drive 0",
					controllerID: "RAID.Slot.2",
					capacity:     1000000000000,
					state:        "Enabled",
					health:       "OK",
				},
			},
			expectedUniqueDrives:   1,
			expectedMappingMetrics: 2,
			expectedDriveMetrics:   3,
			expectedControllerMap:  map[string]int{"Disk.Bay.0": 2},
		},
		{
			name: "multiple drives with one duplicate",
			drives: []driveScenario{
				{
					driveID:      "Disk.Bay.0",
					driveName:    "Drive 0",
					controllerID: "RAID.Slot.1",
					capacity:     1000000000000,
					state:        "Enabled",
					health:       "OK",
				},
				{
					driveID:      "Disk.Bay.0",
					driveName:    "Drive 0",
					controllerID: "RAID.Slot.2",
					capacity:     1000000000000,
					state:        "Enabled",
					health:       "OK",
				},
				{
					driveID:      "Disk.Bay.1",
					driveName:    "Drive 1",
					controllerID: "RAID.Slot.1",
					capacity:     2000000000000,
					state:        "Enabled",
					health:       "OK",
				},
				{
					driveID:      "Disk.Bay.2",
					driveName:    "Drive 2",
					controllerID: "RAID.Slot.2",
					capacity:     500000000000,
					state:        "Disabled",
					health:       "Critical",
				},
			},
			expectedUniqueDrives:   3,
			expectedMappingMetrics: 4,
			expectedDriveMetrics:   9,
			expectedControllerMap: map[string]int{
				"Disk.Bay.0": 2,
				"Disk.Bay.1": 1,
				"Disk.Bay.2": 1,
			},
		},
		{
			name: "all drives duplicated",
			drives: []driveScenario{
				{
					driveID:      "Disk.Bay.0",
					driveName:    "Drive 0",
					controllerID: "RAID.Slot.1",
					capacity:     1000000000000,
					state:        "Enabled",
					health:       "OK",
				},
				{
					driveID:      "Disk.Bay.0",
					driveName:    "Drive 0",
					controllerID: "RAID.Slot.2",
					capacity:     1000000000000,
					state:        "Enabled",
					health:       "OK",
				},
				{
					driveID:      "Disk.Bay.1",
					driveName:    "Drive 1",
					controllerID: "RAID.Slot.1",
					capacity:     2000000000000,
					state:        "Enabled",
					health:       "Warning",
				},
				{
					driveID:      "Disk.Bay.1",
					driveName:    "Drive 1",
					controllerID: "RAID.Slot.2",
					capacity:     2000000000000,
					state:        "Enabled",
					health:       "Warning",
				},
			},
			expectedUniqueDrives:   2,
			expectedMappingMetrics: 4,
			expectedDriveMetrics:   6,
			expectedControllerMap: map[string]int{
				"Disk.Bay.0": 2,
				"Disk.Bay.1": 2,
			},
		},
		{
			name: "drive in three controllers",
			drives: []driveScenario{
				{
					driveID:      "Disk.Bay.0",
					driveName:    "Drive 0",
					controllerID: "RAID.Slot.1",
					capacity:     1000000000000,
					state:        "Enabled",
					health:       "OK",
				},
				{
					driveID:      "Disk.Bay.0",
					driveName:    "Drive 0",
					controllerID: "RAID.Slot.2",
					capacity:     1000000000000,
					state:        "Enabled",
					health:       "OK",
				},
				{
					driveID:      "Disk.Bay.0",
					driveName:    "Drive 0",
					controllerID: "RAID.Slot.3",
					capacity:     1000000000000,
					state:        "Enabled",
					health:       "OK",
				},
			},
			expectedUniqueDrives:   1,
			expectedMappingMetrics: 3,
			expectedDriveMetrics:   3,
			expectedControllerMap:  map[string]int{"Disk.Bay.0": 3},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan prometheus.Metric, 100)
			
			// Build drives and controller map
			driveControllerMap := make(map[string][]string)
			processedDrives := make(map[string]bool)
			wg := &sync.WaitGroup{}
			
			// Expected metric values for validation (if enabled)
			expectedMetricValues := make(map[string]map[string]float64) // driveID -> metric type -> value
			
			for _, scenario := range tt.drives {
				drive := &redfish.Drive{
					Entity: common.Entity{
						ID:   scenario.driveID,
						Name: scenario.driveName,
					},
					CapacityBytes: scenario.capacity,
					Status: common.Status{
						State:  common.State(scenario.state),
						Health: common.Health(scenario.health),
					},
				}
				
				// Track controller mappings
				driveControllerMap[scenario.driveID] = append(driveControllerMap[scenario.driveID], scenario.controllerID)
				
				// Always create mapping metric
				parseDriveControllerMapping(ch, "test-host", drive, scenario.controllerID)
				
				// Only create main metrics once per unique drive
				if !processedDrives[scenario.driveID] {
					processedDrives[scenario.driveID] = true
					wg.Add(1)
					go parseDrive(ch, "test-host", drive, wg)
					
					// Store expected values for validation
					if tt.validateMetrics {
						expectedMetricValues[scenario.driveID] = map[string]float64{
							"capacity": float64(scenario.capacity),
						}
						// Map state values
						switch scenario.state {
						case "Enabled":
							expectedMetricValues[scenario.driveID]["state"] = 1
						case "Disabled":
							expectedMetricValues[scenario.driveID]["state"] = 2
						}
						// Map health values
						switch scenario.health {
						case "OK":
							expectedMetricValues[scenario.driveID]["health"] = 1
						case "Warning":
							expectedMetricValues[scenario.driveID]["health"] = 2
						case "Critical":
							expectedMetricValues[scenario.driveID]["health"] = 3
						}
					}
				}
			}
			wg.Wait()
			
			// Verify unique drive count
			assert.Equal(t, tt.expectedUniqueDrives, len(processedDrives))
			
			// Verify controller mappings
			for driveID, expectedCount := range tt.expectedControllerMap {
				assert.Equal(t, expectedCount, len(driveControllerMap[driveID]), 
					"Drive %s should appear in %d controllers", driveID, expectedCount)
			}
			
			// Count metrics by type and validate values if needed
			mappingCount := 0
			driveMetricCount := 0
			actualMetricValues := make(map[string]map[string]float64)
			
			for len(ch) > 0 {
				metric := <-ch
				desc := metric.Desc().String()
				metricDTO := &dto.Metric{}
				_ = metric.Write(metricDTO)
				
				if strings.Contains(desc, "controller_mapping") {
					mappingCount++
				} else if strings.Contains(desc, "drive") {
					driveMetricCount++
					
					// Extract drive ID from labels for validation
					if tt.validateMetrics {
						var driveID string
						for _, label := range metricDTO.GetLabel() {
							if *label.Name == "drive_id" {
								driveID = *label.Value
								break
							}
						}
						
						if driveID != "" {
							if actualMetricValues[driveID] == nil {
								actualMetricValues[driveID] = make(map[string]float64)
							}
							
							value := metricDTO.Gauge.GetValue()
							if strings.Contains(desc, "drive_state") {
								actualMetricValues[driveID]["state"] = value
							} else if strings.Contains(desc, "drive_health_state") {
								actualMetricValues[driveID]["health"] = value
							} else if strings.Contains(desc, "drive_capacity") {
								actualMetricValues[driveID]["capacity"] = value
							}
						}
					}
				}
			}
			
			assert.Equal(t, tt.expectedMappingMetrics, mappingCount)
			assert.Equal(t, tt.expectedDriveMetrics, driveMetricCount)
			
			// Validate metric values if enabled
			if tt.validateMetrics {
				for driveID, expectedValues := range expectedMetricValues {
					actualValues := actualMetricValues[driveID]
					for metricType, expectedValue := range expectedValues {
						assert.Equal(t, expectedValue, actualValues[metricType], 
							"Drive %s %s metric mismatch", driveID, metricType)
					}
				}
			}
		})
	}
}