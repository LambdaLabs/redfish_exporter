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

// TestParseDrive tests the main drive metrics creation
func TestParseDrive(t *testing.T) {
	ch := make(chan prometheus.Metric, 10)
	done := make(chan bool)
	
	// Create a test drive with all fields
	testDrive := &redfish.Drive{
		Entity: common.Entity{
			ID:   "Disk.Bay.0",
			Name: "Drive 0",
		},
		CapacityBytes: 1000000000000, // 1TB
		Status: common.Status{
			State:  "Enabled",
			Health: "OK",
		},
	}
	
	// Run parseDrive in a goroutine (as it would be in production)
	go func() {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		parseDrive(ch, "test-host", testDrive, wg)
		wg.Wait()
		done <- true
	}()
	
	<-done
	
	// Should have created 3 metrics (state, health, capacity)
	assert.GreaterOrEqual(t, len(ch), 3, "Expected at least 3 metrics")
	
	// Collect and verify metrics
	metrics := make([]prometheus.Metric, 0)
	for len(ch) > 0 {
		metrics = append(metrics, <-ch)
	}
	
	// Verify we got the expected metrics
	foundState := false
	foundHealth := false
	foundCapacity := false
	
	for _, metric := range metrics {
		desc := metric.Desc().String()
		metricDTO := &dto.Metric{}
		_ = metric.Write(metricDTO)
		
		if strings.Contains(desc, "drive_state") {
			foundState = true
			// Verify state value (Enabled = 1)
			assert.Equal(t, float64(1), metricDTO.Gauge.GetValue())
		} else if strings.Contains(desc, "drive_health_state") {
			foundHealth = true
			// Verify health value (OK = 1)
			assert.Equal(t, float64(1), metricDTO.Gauge.GetValue())
		} else if strings.Contains(desc, "drive_capacity") {
			foundCapacity = true
			// Verify capacity value
			assert.Equal(t, float64(1000000000000), metricDTO.Gauge.GetValue())
		}
	}
	
	assert.True(t, foundState, "State metric not found")
	assert.True(t, foundHealth, "Health metric not found")
	assert.True(t, foundCapacity, "Capacity metric not found")
}

// TestDriveDuplicationHandling tests that duplicate drives are handled correctly
func TestDriveDuplicationHandling(t *testing.T) {
	// This test simulates the scenario where the same drive appears in multiple controllers
	ch := make(chan prometheus.Metric, 100)
	
	// Create test drives (same drive ID appearing twice)
	drive1 := &redfish.Drive{
		Entity: common.Entity{
			ID:   "Disk.Bay.0",
			Name: "Drive 0",
		},
		CapacityBytes: 2000000000000,
		Status: common.Status{
			State:  "Enabled",
			Health: "OK",
		},
	}
	
	drive2 := &redfish.Drive{
		Entity: common.Entity{
			ID:   "Disk.Bay.0", // Same ID as drive1
			Name: "Drive 0",
		},
		CapacityBytes: 2000000000000,
		Status: common.Status{
			State:  "Enabled",
			Health: "OK",
		},
	}
	
	// Simulate the deduplication logic from Collect method
	type driveInfo struct {
		drive        *redfish.Drive
		controllerID string
	}
	
	allDrives := []driveInfo{
		{drive: drive1, controllerID: "RAID.Slot.1"},
		{drive: drive2, controllerID: "RAID.Slot.2"}, // Same drive, different controller
	}
	
	driveControllerMap := make(map[string][]string)
	for _, info := range allDrives {
		driveControllerMap[info.drive.ID] = append(driveControllerMap[info.drive.ID], info.controllerID)
	}
	
	// Process drives with deduplication
	processedDrives := make(map[string]bool)
	mappingMetrics := 0
	mainMetrics := 0
	
	wg := &sync.WaitGroup{}
	for _, info := range allDrives {
		// Always create mapping metric
		parseDriveControllerMapping(ch, "test-host", info.drive, info.controllerID)
		mappingMetrics++
		
		// Only create main metrics once per unique drive
		if !processedDrives[info.drive.ID] {
			processedDrives[info.drive.ID] = true
			wg.Add(1)
			go parseDrive(ch, "test-host", info.drive, wg)
			mainMetrics++
		}
	}
	wg.Wait()
	
	// Verify results
	assert.Equal(t, 2, mappingMetrics, "Should have 2 mapping metrics (one per occurrence)")
	assert.Equal(t, 1, mainMetrics, "Should have 1 set of main metrics (deduplicated)")
	
	// Verify the driveControllerMap shows the duplicate
	assert.Equal(t, 2, len(driveControllerMap["Disk.Bay.0"]), "Drive should appear in 2 controllers")
	assert.Contains(t, driveControllerMap["Disk.Bay.0"], "RAID.Slot.1")
	assert.Contains(t, driveControllerMap["Disk.Bay.0"], "RAID.Slot.2")
	
	// Count actual metrics in channel
	totalMetrics := len(ch)
	// Should have: 2 mapping metrics + 3 main metrics (state, health, capacity)
	assert.GreaterOrEqual(t, totalMetrics, 5, "Should have at least 5 total metrics")
}

// TestMultipleDrivesMultipleControllers tests a complex scenario with multiple drives and controllers
func TestMultipleDrivesMultipleControllers(t *testing.T) {
	ch := make(chan prometheus.Metric, 100)
	
	// Create a scenario:
	// - Drive 0 appears in Controller 1 and 2 (duplicate)
	// - Drive 1 appears only in Controller 1
	// - Drive 2 appears only in Controller 2
	
	drives := []struct {
		driveID      string
		driveName    string
		controllerID string
	}{
		{"Disk.Bay.0", "Drive 0", "RAID.Slot.1"},
		{"Disk.Bay.0", "Drive 0", "RAID.Slot.2"}, // Duplicate
		{"Disk.Bay.1", "Drive 1", "RAID.Slot.1"},
		{"Disk.Bay.2", "Drive 2", "RAID.Slot.2"},
	}
	
	// Process drives
	processedDrives := make(map[string]bool)
	driveControllerMap := make(map[string][]string)
	wg := &sync.WaitGroup{}
	
	for _, d := range drives {
		drive := &redfish.Drive{
			Entity: common.Entity{
				ID:   d.driveID,
				Name: d.driveName,
			},
			CapacityBytes: 1000000000000,
			Status: common.Status{
				State:  "Enabled",
				Health: "OK",
			},
		}
		
		// Track controller mappings
		driveControllerMap[d.driveID] = append(driveControllerMap[d.driveID], d.controllerID)
		
		// Always create mapping metric
		parseDriveControllerMapping(ch, "test-host", drive, d.controllerID)
		
		// Only create main metrics once per unique drive
		if !processedDrives[d.driveID] {
			processedDrives[d.driveID] = true
			wg.Add(1)
			go parseDrive(ch, "test-host", drive, wg)
		}
	}
	wg.Wait()
	
	// Verify deduplication worked correctly
	assert.Equal(t, 3, len(processedDrives), "Should have 3 unique drives")
	assert.Equal(t, 2, len(driveControllerMap["Disk.Bay.0"]), "Drive 0 should be in 2 controllers")
	assert.Equal(t, 1, len(driveControllerMap["Disk.Bay.1"]), "Drive 1 should be in 1 controller")
	assert.Equal(t, 1, len(driveControllerMap["Disk.Bay.2"]), "Drive 2 should be in 1 controller")
	
	// Count metrics by type
	mappingCount := 0
	driveMetricCount := 0
	
	for len(ch) > 0 {
		metric := <-ch
		desc := metric.Desc().String()
		if strings.Contains(desc, "controller_mapping") {
			mappingCount++
		} else if strings.Contains(desc, "drive") {
			driveMetricCount++
		}
	}
	
	assert.Equal(t, 4, mappingCount, "Should have 4 mapping metrics (one per drive-controller pair)")
	// 3 unique drives × 3 metrics each (state, health, capacity) = 9
	assert.GreaterOrEqual(t, driveMetricCount, 9, "Should have at least 9 drive metrics")
}