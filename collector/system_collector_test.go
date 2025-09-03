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

	testDrive := &redfish.Drive{
		Entity: common.Entity{
			ID:   "Disk.Bay.0",
			Name: "Drive 0",
		},
	}

	parseDriveControllerMapping(ch, "test-host", testDrive, "RAID.Slot.1")

	assert.Equal(t, 1, len(ch))

	metric := <-ch
	metricDTO := &dto.Metric{}
	err := metric.Write(metricDTO)
	require.NoError(t, err)

	labels := metricDTO.GetLabel()
	assert.Equal(t, 4, len(labels))

	labelMap := make(map[string]string)
	for _, label := range labels {
		labelMap[*label.Name] = *label.Value
	}

	assert.Equal(t, "test-host", labelMap["hostname"])
	assert.Equal(t, "Disk.Bay.0", labelMap["drive_id"])
	assert.Equal(t, "Drive 0", labelMap["drive_name"])
	assert.Equal(t, "RAID.Slot.1", labelMap["storage_controller_id"])
	assert.Equal(t, float64(1), metricDTO.Gauge.GetValue())
}

// TestDriveMetrics tests drive metric generation and deduplication
func TestDriveMetrics(t *testing.T) {
	// Define metric value mappings
	stateEnabled := float64(1)
	stateDisabled := float64(2)
	healthOK := float64(1)
	healthWarning := float64(2)
	healthCritical := float64(3)

	type driveScenario struct {
		driveID      string
		driveName    string
		controllerID string
		capacity     int64
		state        string
		health       string
	}

	type driveMetrics struct {
		capacity float64
		state    float64
		health   float64
	}

	tests := []struct {
		name         string
		drives       []driveScenario
		wantMappings int
		wantMetrics  int
		wantValues   map[string]driveMetrics // driveID -> metrics
	}{
		{
			name: "single drive",
			drives: []driveScenario{
				{driveID: "Disk.Bay.0", driveName: "Drive 0", controllerID: "RAID.Slot.1", capacity: 1000000000000, state: "Enabled", health: "OK"},
			},
			wantMappings: 1,
			wantMetrics:  3,
			wantValues: map[string]driveMetrics{
				"Disk.Bay.0": {capacity: 1000000000000, state: stateEnabled, health: healthOK},
			},
		},
		{
			name: "duplicate drive in two controllers",
			drives: []driveScenario{
				{driveID: "Disk.Bay.0", driveName: "Drive 0", controllerID: "RAID.Slot.1", capacity: 1000000000000, state: "Enabled", health: "OK"},
				{driveID: "Disk.Bay.0", driveName: "Drive 0", controllerID: "RAID.Slot.2", capacity: 1000000000000, state: "Enabled", health: "OK"},
			},
			wantMappings: 2,
			wantMetrics:  3,
			wantValues: map[string]driveMetrics{
				"Disk.Bay.0": {capacity: 1000000000000, state: stateEnabled, health: healthOK},
			},
		},
		{
			name: "multiple unique drives",
			drives: []driveScenario{
				{driveID: "Disk.Bay.0", driveName: "Drive 0", controllerID: "RAID.Slot.1", capacity: 1000000000000, state: "Enabled", health: "OK"},
				{driveID: "Disk.Bay.1", driveName: "Drive 1", controllerID: "RAID.Slot.1", capacity: 2000000000000, state: "Enabled", health: "Warning"},
				{driveID: "Disk.Bay.2", driveName: "Drive 2", controllerID: "RAID.Slot.2", capacity: 500000000000, state: "Disabled", health: "Critical"},
			},
			wantMappings: 3,
			wantMetrics:  9,
			wantValues: map[string]driveMetrics{
				"Disk.Bay.0": {capacity: 1000000000000, state: stateEnabled, health: healthOK},
				"Disk.Bay.1": {capacity: 2000000000000, state: stateEnabled, health: healthWarning},
				"Disk.Bay.2": {capacity: 500000000000, state: stateDisabled, health: healthCritical},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan prometheus.Metric, 100)
			processedDrives := make(map[string]bool)
			wg := &sync.WaitGroup{}

			// Create drives and mappings
			for _, d := range tt.drives {
				drive := &redfish.Drive{
					Entity: common.Entity{
						ID:   d.driveID,
						Name: d.driveName,
					},
					CapacityBytes: d.capacity,
					Status: common.Status{
						State:  common.State(d.state),
						Health: common.Health(d.health),
					},
				}

				parseDriveControllerMapping(ch, "test-host", drive, d.controllerID)

				if !processedDrives[d.driveID] {
					processedDrives[d.driveID] = true
					wg.Add(1)
					go parseDrive(ch, "test-host", drive, wg)
				}
			}
			wg.Wait()

			// Collect metrics from channel
			mappingCount := 0
			driveMetricCount := 0
			actualValues := make(map[string]driveMetrics)

			for len(ch) > 0 {
				metric := <-ch
				desc := metric.Desc().String()
				metricDTO := &dto.Metric{}
				err := metric.Write(metricDTO)
				require.NoError(t, err)

				if strings.Contains(desc, "controller_mapping") {
					mappingCount++
				} else if strings.Contains(desc, "drive") {
					driveMetricCount++

					// Get drive ID from labels
					var driveID string
					for _, label := range metricDTO.GetLabel() {
						if *label.Name == "drive_id" {
							driveID = *label.Value
							break
						}
					}

					// Initialize metrics for drive if needed
					if _, exists := actualValues[driveID]; !exists {
						actualValues[driveID] = driveMetrics{}
					}

					// Store metric value by type
					value := metricDTO.Gauge.GetValue()
					metrics := actualValues[driveID]
					if strings.Contains(desc, "drive_capacity") {
						metrics.capacity = value
					} else if strings.Contains(desc, "drive_state") {
						metrics.state = value
					} else if strings.Contains(desc, "drive_health_state") {
						metrics.health = value
					}
					actualValues[driveID] = metrics
				}
			}

			// Validate counts
			assert.Equal(t, tt.wantMappings, mappingCount)
			assert.Equal(t, tt.wantMetrics, driveMetricCount)

			// Validate all metrics
			for driveID, want := range tt.wantValues {
				actual := actualValues[driveID]
				assert.Equal(t, want.capacity, actual.capacity, "Drive %s capacity", driveID)
				assert.Equal(t, want.state, actual.state, "Drive %s state", driveID)
				assert.Equal(t, want.health, actual.health, "Drive %s health", driveID)
			}
		})
	}
}
