package collector

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriveLocationOrdinal(t *testing.T) {
	for _, tt := range []struct {
		name, payload, want string
	}{
		{"no location", `{}`, ""},
		{"missing ordinal", `{"PhysicalLocation":{"PartLocation":{"ServiceLabel":"Bay 7"}}}`, ""},
		{"null ordinal", `{"PhysicalLocation":{"PartLocation":{"LocationOrdinalValue":null}}}`, ""},
		{"zero ordinal", `{"PhysicalLocation":{"PartLocation":{"LocationOrdinalValue":0}}}`, "0"},
		{"quoted zero", `{"PhysicalLocation":{"PartLocation":{"LocationOrdinalValue":"0"}}}`, "0"},
		{"nonzero ordinal", `{"PhysicalLocation":{"PartLocation":{"LocationOrdinalValue":4}}}`, "4"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var drive schemas.Drive
			require.NoError(t, json.Unmarshal([]byte(tt.payload), &drive))
			assert.Equal(t, tt.want, driveLocationOrdinal(&drive))
		})
	}
}

// TestDriveMetrics tests drive metric generation with controller dimension
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
		name        string
		drives      []driveScenario
		wantMetrics int
		wantValues  map[string]driveMetrics // driveID -> metrics
	}{
		{
			name: "single drive",
			drives: []driveScenario{
				{driveID: "Disk.Bay.0", driveName: "Drive 0", controllerID: "RAID.Slot.1", capacity: 1000000000000, state: "Enabled", health: "OK"},
			},
			wantMetrics: 3,
			wantValues: map[string]driveMetrics{
				"Disk.Bay.0:RAID.Slot.1": {capacity: 1000000000000, state: stateEnabled, health: healthOK},
			},
		},
		{
			name: "duplicate drive in two controllers - same status",
			drives: []driveScenario{
				{driveID: "Disk.Bay.0", driveName: "Drive 0", controllerID: "RAID.Slot.1", capacity: 1000000000000, state: "Enabled", health: "OK"},
				{driveID: "Disk.Bay.0", driveName: "Drive 0", controllerID: "RAID.Slot.2", capacity: 1000000000000, state: "Enabled", health: "OK"},
			},
			wantMetrics: 6, // 3 metrics × 2 controllers
			wantValues: map[string]driveMetrics{
				"Disk.Bay.0:RAID.Slot.1": {capacity: 1000000000000, state: stateEnabled, health: healthOK},
				"Disk.Bay.0:RAID.Slot.2": {capacity: 1000000000000, state: stateEnabled, health: healthOK},
			},
		},
		{
			name: "duplicate drive - different health per controller",
			drives: []driveScenario{
				{driveID: "Disk.Bay.1", driveName: "Drive 1", controllerID: "RAID.Slot.1", capacity: 2000000000000, state: "Enabled", health: "OK"},
				{driveID: "Disk.Bay.1", driveName: "Drive 1", controllerID: "RAID.Slot.2", capacity: 2000000000000, state: "Enabled", health: "Warning"},
			},
			wantMetrics: 6, // 3 metrics × 2 controllers
			wantValues: map[string]driveMetrics{
				"Disk.Bay.1:RAID.Slot.1": {capacity: 2000000000000, state: stateEnabled, health: healthOK},
				"Disk.Bay.1:RAID.Slot.2": {capacity: 2000000000000, state: stateEnabled, health: healthWarning},
			},
		},
		{
			name: "duplicate drive - different health across three controllers",
			drives: []driveScenario{
				{driveID: "Disk.Bay.2", driveName: "Drive 2", controllerID: "RAID.Slot.1", capacity: 3000000000000, state: "Enabled", health: "Warning"},
				{driveID: "Disk.Bay.2", driveName: "Drive 2", controllerID: "RAID.Slot.2", capacity: 3000000000000, state: "Enabled", health: "Critical"},
				{driveID: "Disk.Bay.2", driveName: "Drive 2", controllerID: "RAID.Slot.3", capacity: 3000000000000, state: "Enabled", health: "OK"},
			},
			wantMetrics: 9, // 3 metrics × 3 controllers
			wantValues: map[string]driveMetrics{
				"Disk.Bay.2:RAID.Slot.1": {capacity: 3000000000000, state: stateEnabled, health: healthWarning},
				"Disk.Bay.2:RAID.Slot.2": {capacity: 3000000000000, state: stateEnabled, health: healthCritical},
				"Disk.Bay.2:RAID.Slot.3": {capacity: 3000000000000, state: stateEnabled, health: healthOK},
			},
		},
		{
			name: "duplicate drive - different state per controller",
			drives: []driveScenario{
				{driveID: "Disk.Bay.3", driveName: "Drive 3", controllerID: "RAID.Slot.1", capacity: 4000000000000, state: "Enabled", health: "OK"},
				{driveID: "Disk.Bay.3", driveName: "Drive 3", controllerID: "RAID.Slot.2", capacity: 4000000000000, state: "Disabled", health: "OK"},
			},
			wantMetrics: 6, // 3 metrics × 2 controllers
			wantValues: map[string]driveMetrics{
				"Disk.Bay.3:RAID.Slot.1": {capacity: 4000000000000, state: stateEnabled, health: healthOK},
				"Disk.Bay.3:RAID.Slot.2": {capacity: 4000000000000, state: stateDisabled, health: healthOK},
			},
		},
		{
			name: "multiple unique drives",
			drives: []driveScenario{
				{driveID: "Disk.Bay.0", driveName: "Drive 0", controllerID: "RAID.Slot.1", capacity: 1000000000000, state: "Enabled", health: "OK"},
				{driveID: "Disk.Bay.1", driveName: "Drive 1", controllerID: "RAID.Slot.1", capacity: 2000000000000, state: "Enabled", health: "Warning"},
				{driveID: "Disk.Bay.2", driveName: "Drive 2", controllerID: "RAID.Slot.2", capacity: 500000000000, state: "Disabled", health: "Critical"},
			},
			wantMetrics: 9,
			wantValues: map[string]driveMetrics{
				"Disk.Bay.0:RAID.Slot.1": {capacity: 1000000000000, state: stateEnabled, health: healthOK},
				"Disk.Bay.1:RAID.Slot.1": {capacity: 2000000000000, state: stateEnabled, health: healthWarning},
				"Disk.Bay.2:RAID.Slot.2": {capacity: 500000000000, state: stateDisabled, health: healthCritical},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan prometheus.Metric, 100)
			eg := newRecoverGroup(context.Background())

			// Process all drive+controller combinations
			for _, d := range tt.drives {
				drive := &schemas.Drive{
					Entity: schemas.Entity{
						ID:   d.driveID,
						Name: d.driveName,
					},
					CapacityBytes: gofish.ToRef(int(d.capacity)),
					Status: schemas.Status{
						State:  schemas.State(d.state),
						Health: schemas.Health(d.health),
					},
				}

				// Create drive metrics with controller ID
				dr := drive
				cID := d.controllerID
				eg.Go(func() error {
					parseDrive(ch, dr, cID)
					return nil
				})
			}
			eg.Wait() //nolint:errcheck

			// Collect metrics from channel
			driveMetricCount := 0
			actualValues := make(map[string]driveMetrics)

			for len(ch) > 0 {
				metric := <-ch
				desc := metric.Desc().String()
				metricDTO := &dto.Metric{}
				err := metric.Write(metricDTO)
				require.NoError(t, err)

				if strings.Contains(desc, "drive") {
					driveMetricCount++

					// Get drive ID and controller ID from labels
					var driveID, controllerID string
					for _, label := range metricDTO.GetLabel() {
						switch *label.Name {
						case "drive_id":
							driveID = *label.Value
						case "storage_controller_id":
							controllerID = *label.Value
						}
					}

					// Create composite key for drive+controller
					key := driveID + ":" + controllerID

					// Initialize metrics for drive+controller if needed
					if _, exists := actualValues[key]; !exists {
						actualValues[key] = driveMetrics{}
					}

					// Store metric value by type
					value := metricDTO.Gauge.GetValue()
					metrics := actualValues[key]
					if strings.Contains(desc, "drive_capacity") {
						metrics.capacity = value
					} else if strings.Contains(desc, "drive_state") {
						metrics.state = value
					} else if strings.Contains(desc, "drive_health_state") {
						metrics.health = value
					}
					actualValues[key] = metrics
				}
			}

			// Validate counts
			assert.Equal(t, tt.wantMetrics, driveMetricCount)

			// Validate all metrics
			for key, want := range tt.wantValues {
				actual := actualValues[key]
				assert.Equal(t, want.capacity, actual.capacity, "Drive+Controller %s capacity", key)
				assert.Equal(t, want.state, actual.state, "Drive+Controller %s state", key)
				assert.Equal(t, want.health, actual.health, "Drive+Controller %s health", key)
			}
		})
	}
}

// TestEthernetInterfaceSpeedLabel checks that the speed label carries the reported speed.
// It once carried the address of the decoded value, which changed on every scrape.
func TestEthernetInterfaceSpeedLabel(t *testing.T) {
	for _, tt := range []struct {
		name, payload, want string
	}{
		{"reported speed", `{"Id":"1","Name":"LAN 1","SpeedMbps":200000}`, "200000 Mbps"},
		{"link down", `{"Id":"1","Name":"LAN 1","SpeedMbps":0}`, "0 Mbps"},
		{"speed not reported", `{"Id":"1","Name":"LAN 1"}`, "0 Mbps"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Two scrapes decode the same response into two separate values.
			for scrape := 0; scrape < 2; scrape++ {
				var ethernetInterface schemas.EthernetInterface
				require.NoError(t, json.Unmarshal([]byte(tt.payload), &ethernetInterface))

				ch := make(chan prometheus.Metric, 10)
				parseEthernetInterface(ch, &ethernetInterface)
				close(ch)

				for metric := range ch {
					m := &dto.Metric{}
					require.NoError(t, metric.Write(m))
					for _, label := range m.GetLabel() {
						if label.GetName() == "ethernet_interface_speed" {
							assert.Equal(t, tt.want, label.GetValue())
						}
					}
				}
			}
		})
	}
}

// TestProcessorWithoutMetrics tests that processor collection works when ProcessorMetrics is not available
func TestProcessorWithoutMetrics(t *testing.T) {
	// Mock processor without metrics (no metrics field means Metrics() returns nil)
	mockProcessor := &schemas.Processor{
		Entity: schemas.Entity{
			ID:   "CPU_1",
			Name: "CPU 1",
		},
		TotalCores:   gofish.ToRef(8),
		TotalThreads: gofish.ToRef(16),
		Status: schemas.Status{
			State:        "Enabled",
			Health:       "OK",
			HealthRollup: "OK",
		},
	}

	// Test metric collection
	ch := make(chan prometheus.Metric, 100)

	// Create a test logger
	logger := slog.Default()

	parseProcessor(ch, mockProcessor, logger)

	// Collect metrics from channel
	processorMetricCount := 0
	pcieErrorMetricCount := 0
	cacheMetricCount := 0
	basicMetrics := make(map[string]bool)

	for len(ch) > 0 {
		metric := <-ch
		desc := metric.Desc().String()

		if strings.Contains(desc, "processor") {
			processorMetricCount++
			if strings.Contains(desc, "pcie_errors") {
				pcieErrorMetricCount++
			} else if strings.Contains(desc, "cache_lifetime") {
				cacheMetricCount++
			}
			// Track which basic metrics we got
			if strings.Contains(desc, "processor_state") {
				basicMetrics["state"] = true
			} else if strings.Contains(desc, "processor_health_state") {
				basicMetrics["health"] = true
			} else if strings.Contains(desc, "processor_health_rollup") {
				basicMetrics["health_rollup"] = true
			} else if strings.Contains(desc, "processor_total_threads") {
				basicMetrics["threads"] = true
			} else if strings.Contains(desc, "processor_total_cores") {
				basicMetrics["cores"] = true
			}
		}
	}

	// Verify basic metrics were collected
	assert.Equal(t, 5, processorMetricCount, "Should have exactly 5 basic processor metrics")
	assert.True(t, basicMetrics["state"], "Should have processor state metric")
	assert.True(t, basicMetrics["health"], "Should have processor health metric")
	assert.True(t, basicMetrics["health_rollup"], "Should have processor health rollup metric")
	assert.True(t, basicMetrics["threads"], "Should have processor threads metric")
	assert.True(t, basicMetrics["cores"], "Should have processor cores metric")

	// Verify no PCIe error metrics were collected when ProcessorMetrics is unavailable
	assert.Equal(t, 0, pcieErrorMetricCount, "Should have no PCIe error metrics when ProcessorMetrics is unavailable")

	// Verify no cache metrics were collected when ProcessorMetrics is unavailable
	assert.Equal(t, 0, cacheMetricCount, "Should have no cache metrics when ProcessorMetrics is unavailable")
}

// TestSystemCollectorBiosInfo verifies that the system collector emits
// redfish_system_bios_info for the host system (with bios_version + model
// labels) and skips members that report no BiosVersion (e.g. HGX baseboards).
func TestSystemCollectorBiosInfo(t *testing.T) {
	_, client := setupTestServerClient(t, "testdata/gpu_info_unknown_multi")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewSystemCollector(t.Name(), client, logger, config.DefaultSystemCollector)
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 1000)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	var biosInfoLabels []map[string]string
	for metric := range ch {
		if !strings.Contains(metric.Desc().String(), "system_bios_info") {
			continue
		}
		d := &dto.Metric{}
		require.NoError(t, metric.Write(d))
		require.Equal(t, float64(1), d.GetGauge().GetValue())
		labelMap := make(map[string]string)
		for _, label := range d.Label {
			labelMap[label.GetName()] = label.GetValue()
		}
		biosInfoLabels = append(biosInfoLabels, labelMap)
	}

	// Exactly one series: the host system. HGX_Baseboard_0 has no BiosVersion
	// and must be filtered out.
	require.Len(t, biosInfoLabels, 1, "expected exactly one host bios_info series")
	require.Equal(t, "2.4b", biosInfoLabels[0]["bios_version"])
	require.Equal(t, "EXAMPLE-MODEL-01", biosInfoLabels[0]["model"])
}

// TestDuplicateDriveListingGathers reproduces BMC firmwares that list the
// same drive more than once under a single storage controller. Without
// dedupe, the duplicate label sets fail the entire registry gather ("was
// collected before with the same name and label values"), which turns every
// scrape of such a host into an HTTP 500.
func TestDuplicateDriveListingGathers(t *testing.T) {
	srv := newTestRedfishServer(t)

	systemPath := "/redfish/v1/Systems/System1"
	storagePath := systemPath + "/Storage/Storage.1"
	drivePath := storagePath + "/Drives/Drive.Bay.1"

	system := srv.loadFixture("system1.json")
	system["Storage"] = map[string]string{"@odata.id": systemPath + "/Storage"}
	delete(system, "Processors") // keep the mock focused on storage
	srv.addRoute(systemPath, system)

	srv.addRoute(systemPath+"/Storage", map[string]any{
		"@odata.type": "#StorageCollection.StorageCollection",
		"Members": []map[string]string{
			{"@odata.id": storagePath},
		},
	})

	// The BMC references the same drive twice within one storage subsystem.
	srv.addRoute(storagePath, map[string]any{
		"@odata.type": "#Storage.v1_13_0.Storage",
		"@odata.id":   storagePath,
		"Id":          "Storage.1",
		"Name":        "Storage.1",
		"Drives": []map[string]string{
			{"@odata.id": drivePath},
			{"@odata.id": drivePath},
		},
	})

	srv.addRoute(drivePath, map[string]any{
		"@odata.type":   "#Drive.v1_16_0.Drive",
		"@odata.id":     drivePath,
		"Id":            "Drive.Bay.1",
		"Name":          "Drive.Bay.1",
		"CapacityBytes": 1920398934016,
		"Status":        map[string]string{"State": "Enabled", "Health": "OK"},
	})

	client := connectToTestServer(t, srv.Server)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewSystemCollector(t.Name(), client, logger, config.DefaultSystemCollector)
	require.NoError(t, err)

	registry := prometheus.NewPedanticRegistry()
	require.NoError(t, registry.Register(collector))

	families, err := registry.Gather()
	require.NoError(t, err, "gather must not fail on duplicate drive listings")

	got := map[string]int{}
	for _, mf := range families {
		if strings.HasPrefix(mf.GetName(), "redfish_system_storage_drive_") {
			got[mf.GetName()] = len(mf.GetMetric())
		}
	}
	assert.Equal(t, map[string]int{
		"redfish_system_storage_drive_state":        1,
		"redfish_system_storage_drive_health_state": 1,
		"redfish_system_storage_drive_capacity":     1,
	}, got, "each drive metric family should carry exactly one deduped series")
}

// TestSameDriveIDInTwoEnclosuresGathers reproduces a BMC that numbers the bays
// of each enclosure from zero under one storage resource, so two drives share
// an Id and a Name. Both must be emitted, told apart by their path.
func TestSameDriveIDInTwoEnclosuresGathers(t *testing.T) {
	srv := newTestRedfishServer(t)

	systemPath := "/redfish/v1/Systems/System1"
	storagePath := systemPath + "/Storage/Storage.1"
	drivePaths := []string{
		"/redfish/v1/Chassis/Enclosure.0/Drives/Disk.Bay.0",
		"/redfish/v1/Chassis/Enclosure.1/Drives/Disk.Bay.0",
	}
	ordinals := []int{0, 4}

	system := srv.loadFixture("system1.json")
	system["Storage"] = map[string]string{"@odata.id": systemPath + "/Storage"}
	delete(system, "Processors") // keep the mock focused on storage
	srv.addRoute(systemPath, system)

	srv.addRoute(systemPath+"/Storage", map[string]any{
		"@odata.type": "#StorageCollection.StorageCollection",
		"Members": []map[string]string{
			{"@odata.id": storagePath},
		},
	})

	srv.addRoute(storagePath, map[string]any{
		"@odata.type": "#Storage.v1_13_0.Storage",
		"@odata.id":   storagePath,
		"Id":          "Storage.1",
		"Name":        "Storage.1",
		"Drives": []map[string]string{
			{"@odata.id": drivePaths[0]},
			{"@odata.id": drivePaths[1]},
		},
	})

	for i, drivePath := range drivePaths {
		srv.addRoute(drivePath, map[string]any{
			"@odata.type":   "#Drive.v1_16_0.Drive",
			"@odata.id":     drivePath,
			"Id":            "Disk.Bay.0",
			"Name":          "Disk.Bay.0",
			"CapacityBytes": 1920398934016,
			"Status":        map[string]string{"State": "Enabled", "Health": "OK"},
			"PhysicalLocation": map[string]any{
				"PartLocation": map[string]any{"LocationType": "Bay", "LocationOrdinalValue": ordinals[i]},
			},
		})
	}

	client := connectToTestServer(t, srv.Server)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewSystemCollector(t.Name(), client, logger, config.DefaultSystemCollector)
	require.NoError(t, err)

	registry := prometheus.NewPedanticRegistry()
	require.NoError(t, registry.Register(collector))

	families, err := registry.Gather()
	require.NoError(t, err, "gather must not fail when two drives share an Id")

	got := map[string]string{} // drive_path -> location_ordinal
	for _, mf := range families {
		if mf.GetName() != "redfish_system_storage_drive_health_state" {
			continue
		}
		for _, m := range mf.GetMetric() {
			labels := map[string]string{}
			for _, label := range m.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			assert.Equal(t, "Disk.Bay.0", labels["drive_id"])
			got[labels["drive_path"]] = labels["location_ordinal"]
		}
	}
	assert.Equal(t, map[string]string{
		drivePaths[0]: "0",
		drivePaths[1]: "4",
	}, got, "each drive should have its own series, labelled with its path and location ordinal")
}
