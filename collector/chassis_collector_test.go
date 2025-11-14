package collector

import (
	"io"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stmcginnis/gofish"
	"github.com/stretchr/testify/require"
)

func TestGetLeakDetectors(t *testing.T) {
	tT := map[string]struct {
		mockSetupFn func() (*testRedfishServer, *gofish.APIClient)
		expectedLDs int
	}{
		"happy path - data was returned in a gofish-expected manner": {
			mockSetupFn: func() (*testRedfishServer, *gofish.APIClient) {
				server := newTestRedfishServer(t)
				server.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "chassis_main.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem", "thermal_subsystem.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection", "leak_detection.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors", "leak_detectors_collection.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors/Chassis_0_LeakDetector_0_ColdPlate", "leak_detector_ok.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors/Chassis_0_LeakDetector_0_Manifold", "leak_detector_ok.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors/Chassis_0_LeakDetector_1_ColdPlate", "leak_detector_ok.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors/Chassis_0_LeakDetector_1_Manifold", "leak_detector_leak.json")

				client := connectToTestServer(t, server.Server)

				return server, client
			},
			expectedLDs: 4,
		},
		"happy path - OEM returned single LeakDetection in ThermalSubsystem": {
			mockSetupFn: func() (*testRedfishServer, *gofish.APIClient) {
				server := newTestRedfishServer(t)
				server.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "chassis_main.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem", "thermal_subsystem.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection", "single_leak_detection.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors", "leak_detectors_collection.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors/Chassis_0_LeakDetector_0_ColdPlate", "leak_detector_ok.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors/Chassis_0_LeakDetector_0_Manifold", "leak_detector_ok.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors/Chassis_0_LeakDetector_1_ColdPlate", "leak_detector_ok.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors/Chassis_0_LeakDetector_1_Manifold", "leak_detector_leak.json")

				client := connectToTestServer(t, server.Server)

				return server, client
			},
			expectedLDs: 4,
		},
	}

	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			srv, client := test.mockSetupFn()
			require.NotNil(t, srv)
			require.NotNil(t, client)
			t.Cleanup(func() {
				client.Logout()
				srv.Close()
			})

			service := client.GetService()
			chassis, err := service.Chassis()
			require.NoError(t, err)
			require.NotEmpty(t, chassis, "Expected at least one chassis")

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			collector, err := NewChassisCollector(t.Name(), client, logger, config.DefaultChassisCollector)
			require.NoError(t, err)
			thermalSubsystem, err := chassis[0].ThermalSubsystem()
			require.NoError(t, err)

			detectors := collector.getLeakDetectors(thermalSubsystem, logger)

			require.Equal(t, test.expectedLDs, len(detectors))
		})
	}
}

func TestParseLeakDetector(t *testing.T) {
	server := newTestRedfishServer(t)
	server.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "chassis_main.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem", "thermal_subsystem.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection", "single_leak_detection.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors", "leak_detectors_single.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors/Chassis_0_LeakDetector_0_ColdPlate", "leak_detector_ok.json")

	client := connectToTestServer(t, server.Server)
	t.Cleanup(func() {
		client.Logout()
		server.Close()
	})

	service := client.GetService()
	chassis, err := service.Chassis()
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewChassisCollector(t.Name(), client, logger, config.DefaultChassisCollector)
	require.NoError(t, err)
	thermalSubsystem, err := chassis[0].ThermalSubsystem()
	require.NoError(t, err)

	detectors := collector.getLeakDetectors(thermalSubsystem, logger)
	require.Greater(t, len(detectors), 0)

	metricsCh := make(chan prometheus.Metric, 10)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	parseLeakDetector(metricsCh, "test_chassis", detectors[0], wg)
	close(metricsCh)
	wg.Wait()

	for metric := range metricsCh {
		dto := &dto.Metric{}
		require.NoError(t, metric.Write(dto))

		// Verify the metric has the expected labels and value
		require.Len(t, dto.Label, 4, "Expected 4 labels")

		// Check labels
		labelMap := make(map[string]string)
		for _, label := range dto.Label {
			labelMap[label.GetName()] = label.GetValue()
		}

		require.Equal(t, "test_chassis", labelMap["chassis_id"])
		require.Equal(t, "LeakDetection", labelMap["leak_detection_id"])
		require.Equal(t, "Chassis_0_LeakDetector_0_ColdPlate", labelMap["leak_detector_id"])
		require.Equal(t, "leak_detector", labelMap["resource"])

		// Check gauge value
		require.NotNil(t, dto.Gauge, "Expected gauge metric")
		require.Equal(t, float64(1), dto.Gauge.GetValue())
	}
}

// TestCollectTotalGPUPower tests the collection of total GPU power metric
// Note: This metric is now collected via TelemetryService (HGX_PlatformEnvironmentMetrics_0)
func TestCollectTotalGPUPower(t *testing.T) {
	t.Skip("chassis_gpu_total_power_watts is now collected via TelemetryCollector from HGX_PlatformEnvironmentMetrics_0")
	server := newTestRedfishServer(t)

	// Add chassis collection
	server.addRoute("/redfish/v1/Chassis", map[string]interface{}{
		"@odata.type": "#ChassisCollection.ChassisCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Chassis/HGX_Chassis_0"},
		},
		"Members@odata.count": 1,
	})

	// Add main chassis
	server.addRoute("/redfish/v1/Chassis/HGX_Chassis_0", map[string]interface{}{
		"@odata.type": "#Chassis.v1_20_0.Chassis",
		"@odata.id":   "/redfish/v1/Chassis/HGX_Chassis_0",
		"Id":          "HGX_Chassis_0",
		"Name":        "HGX Chassis",
		"ChassisType": "RackMount",
		"Controls": map[string]interface{}{
			"@odata.id": "/redfish/v1/Chassis/HGX_Chassis_0/Controls",
		},
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
	})

	// Add controls collection
	server.addRoute("/redfish/v1/Chassis/HGX_Chassis_0/Controls", map[string]interface{}{
		"@odata.type": "#ControlCollection.ControlCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Chassis/HGX_Chassis_0/Controls/TotalGPU_Power_0"},
		},
		"Members@odata.count": 1,
	})

	// Add total GPU power control
	server.addRoute("/redfish/v1/Chassis/HGX_Chassis_0/Controls/TotalGPU_Power_0", map[string]interface{}{
		"@odata.type":   "#Control.v1_5_0.Control",
		"@odata.id":     "/redfish/v1/Chassis/HGX_Chassis_0/Controls/TotalGPU_Power_0",
		"Id":            "TotalGPU_Power_0",
		"Name":          "Total GPU Power",
		"ControlType":   "Power",
		"SetPointUnits": "W",
		"Sensor": map[string]interface{}{
			"Reading":       673.8720092773438,
			"DataSourceUri": "/redfish/v1/Chassis/HGX_Chassis_0/Sensors/TotalGPU_Power",
		},
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
	})

	client := connectToTestServer(t, server.Server)
	defer client.Logout()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewChassisCollector(t.Name(), client, logger, config.DefaultChassisCollector)
	require.NoError(t, err)

	// Collect metrics
	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	// Check for total GPU power metric
	foundTotalGPUPower := false
	var totalGPUPowerValue float64

	for metric := range ch {
		dto := &dto.Metric{}
		require.NoError(t, metric.Write(dto))

		desc := metric.Desc()
		descString := desc.String()

		if strings.Contains(descString, "gpu_total_power_watts") {
			foundTotalGPUPower = true
			totalGPUPowerValue = dto.Gauge.GetValue()

			// Check labels
			labelMap := make(map[string]string)
			for _, label := range dto.Label {
				labelMap[label.GetName()] = label.GetValue()
			}
			require.Equal(t, "HGX_Chassis_0", labelMap["chassis_id"])
			require.Equal(t, "chassis", labelMap["resource"])
			break
		}
	}

	require.True(t, foundTotalGPUPower, "Total GPU power metric should be collected")
	require.InDelta(t, 673.872, totalGPUPowerValue, 0.01, "Total GPU power value should match")
}

// TestCollectTotalGPUPowerMultipleChassis tests total GPU power collection with multiple chassis
// where only some have the control endpoint
// Note: This metric is now collected via TelemetryService (HGX_PlatformEnvironmentMetrics_0)
func TestCollectTotalGPUPowerMultipleChassis(t *testing.T) {
	t.Skip("chassis_gpu_total_power_watts is now collected via TelemetryCollector from HGX_PlatformEnvironmentMetrics_0")
	server := newTestRedfishServer(t)

	// Add chassis collection with multiple chassis
	server.addRoute("/redfish/v1/Chassis", map[string]interface{}{
		"@odata.type": "#ChassisCollection.ChassisCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Chassis/HGX_Chassis_0"},
			{"@odata.id": "/redfish/v1/Chassis/System_Chassis_1"},
			{"@odata.id": "/redfish/v1/Chassis/HGX_ProcessorModule_0"},
		},
		"Members@odata.count": 3,
	})

	// Add main HGX chassis with GPU power control
	server.addRoute("/redfish/v1/Chassis/HGX_Chassis_0", map[string]interface{}{
		"@odata.type": "#Chassis.v1_20_0.Chassis",
		"@odata.id":   "/redfish/v1/Chassis/HGX_Chassis_0",
		"Id":          "HGX_Chassis_0",
		"Name":        "HGX Chassis",
		"ChassisType": "RackMount",
		"Controls": map[string]interface{}{
			"@odata.id": "/redfish/v1/Chassis/HGX_Chassis_0/Controls",
		},
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
	})

	// Add controls collection for HGX_Chassis_0 with multiple GPU power controls
	server.addRoute("/redfish/v1/Chassis/HGX_Chassis_0/Controls", map[string]interface{}{
		"@odata.type": "#ControlCollection.ControlCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Chassis/HGX_Chassis_0/Controls/TotalGPU_Power_0"},
			{"@odata.id": "/redfish/v1/Chassis/HGX_Chassis_0/Controls/TotalGPU_Power_1"},
		},
		"Members@odata.count": 2,
	})

	// Add first GPU power control for HGX_Chassis_0
	server.addRoute("/redfish/v1/Chassis/HGX_Chassis_0/Controls/TotalGPU_Power_0", map[string]interface{}{
		"@odata.type":   "#Control.v1_5_0.Control",
		"@odata.id":     "/redfish/v1/Chassis/HGX_Chassis_0/Controls/TotalGPU_Power_0",
		"Id":            "TotalGPU_Power_0",
		"Name":          "Total GPU Power",
		"ControlType":   "Power",
		"SetPointUnits": "W",
		"Sensor": map[string]interface{}{
			"Reading":       673.8720092773438,
			"DataSourceUri": "/redfish/v1/Chassis/HGX_Chassis_0/Sensors/TotalGPU_Power",
		},
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
	})

	// Add second GPU power control for HGX_Chassis_0
	server.addRoute("/redfish/v1/Chassis/HGX_Chassis_0/Controls/TotalGPU_Power_1", map[string]interface{}{
		"@odata.type":   "#Control.v1_5_0.Control",
		"@odata.id":     "/redfish/v1/Chassis/HGX_Chassis_0/Controls/TotalGPU_Power_1",
		"Id":            "TotalGPU_Power_1",
		"Name":          "Total GPU Power Group 2",
		"ControlType":   "Power",
		"SetPointUnits": "W",
		"Sensor": map[string]interface{}{
			"Reading":       450.25,
			"DataSourceUri": "/redfish/v1/Chassis/HGX_Chassis_0/Sensors/TotalGPU_Power_1",
		},
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
	})

	// Add regular system chassis without GPU power control
	server.addRoute("/redfish/v1/Chassis/System_Chassis_1", map[string]interface{}{
		"@odata.type": "#Chassis.v1_20_0.Chassis",
		"@odata.id":   "/redfish/v1/Chassis/System_Chassis_1",
		"Id":          "System_Chassis_1",
		"Name":        "System Chassis",
		"ChassisType": "RackMount",
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
	})

	// Add processor module chassis without GPU power control
	server.addRoute("/redfish/v1/Chassis/HGX_ProcessorModule_0", map[string]interface{}{
		"@odata.type": "#Chassis.v1_20_0.Chassis",
		"@odata.id":   "/redfish/v1/Chassis/HGX_ProcessorModule_0",
		"Id":          "HGX_ProcessorModule_0",
		"Name":        "HGX Processor Module",
		"ChassisType": "Module",
		"Status": map[string]string{
			"State":  "Enabled",
			"Health": "OK",
		},
	})

	client := connectToTestServer(t, server.Server)
	defer client.Logout()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewChassisCollector(t.Name(), client, logger, config.DefaultChassisCollector)
	require.NoError(t, err)

	// Collect metrics
	ch := make(chan prometheus.Metric, 200)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	// Track which chassis have GPU power metrics
	gpuPowerMetrics := []float64{}
	chassisHealthMetrics := make(map[string]bool)

	for metric := range ch {
		dto := &dto.Metric{}
		require.NoError(t, metric.Write(dto))

		desc := metric.Desc()
		descString := desc.String()

		// Get chassis_id from labels
		var chassisID string
		for _, label := range dto.Label {
			if label.GetName() == "chassis_id" {
				chassisID = label.GetValue()
				break
			}
		}

		if strings.Contains(descString, "gpu_total_power_watts") {
			gpuPowerMetrics = append(gpuPowerMetrics, dto.Gauge.GetValue())
		}

		if strings.Contains(descString, "chassis_health") && !strings.Contains(descString, "rollup") {
			chassisHealthMetrics[chassisID] = true
		}
	}

	// Verify all three chassis were processed
	require.Len(t, chassisHealthMetrics, 3, "Should have collected health metrics for all 3 chassis")

	// Verify we collected both GPU power metrics from HGX_Chassis_0
	require.Len(t, gpuPowerMetrics, 2, "Should have collected 2 GPU power metrics from HGX_Chassis_0")

	// Sort to ensure consistent ordering
	sort.Float64s(gpuPowerMetrics)
	require.InDelta(t, 450.25, gpuPowerMetrics[0], 0.01, "First GPU power value should match")
	require.InDelta(t, 673.872, gpuPowerMetrics[1], 0.01, "Second GPU power value should match")
}

// TestCollectTotalGPUPowerErrorHandling tests error handling for GPU power collection
// Note: This metric is now collected via TelemetryService (HGX_PlatformEnvironmentMetrics_0)
func TestCollectTotalGPUPowerErrorHandling(t *testing.T) {
	t.Skip("chassis_gpu_total_power_watts is now collected via TelemetryCollector from HGX_PlatformEnvironmentMetrics_0")
	testCases := []struct {
		name            string
		controlResponse map[string]interface{}
		expectMetric    bool
		expectedValue   float64
	}{
		{
			name: "control with zero reading should still emit metric",
			controlResponse: map[string]interface{}{
				"@odata.type":   "#Control.v1_5_0.Control",
				"Id":            "TotalGPU_Power_0",
				"Name":          "Total GPU Power",
				"ControlType":   "Power",
				"SetPointUnits": "W",
				"Sensor": map[string]interface{}{
					"Reading":       0,
					"DataSourceUri": "/redfish/v1/Chassis/HGX_Chassis_0/Sensors/TotalGPU_Power",
				},
			},
			expectMetric:  true,
			expectedValue: 0,
		},
		{
			name: "control with wrong type",
			controlResponse: map[string]interface{}{
				"@odata.type":   "#Control.v1_5_0.Control",
				"Id":            "TotalGPU_Power_0",
				"Name":          "Total GPU Power",
				"ControlType":   "Temperature", // Wrong type
				"SetPointUnits": "Cel",
				"Sensor": map[string]interface{}{
					"Reading":       100.5,
					"DataSourceUri": "/redfish/v1/Chassis/HGX_Chassis_0/Sensors/Something",
				},
			},
			expectMetric: false,
		},
		{
			name: "valid control with power reading",
			controlResponse: map[string]interface{}{
				"@odata.type":   "#Control.v1_5_0.Control",
				"Id":            "TotalGPU_Power_0",
				"Name":          "Total GPU Power",
				"ControlType":   "Power",
				"SetPointUnits": "W",
				"Sensor": map[string]interface{}{
					"Reading":       500.25,
					"DataSourceUri": "/redfish/v1/Chassis/HGX_Chassis_0/Sensors/TotalGPU_Power",
				},
			},
			expectMetric:  true,
			expectedValue: 500.25,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := newTestRedfishServer(t)

			// Add chassis collection
			server.addRoute("/redfish/v1/Chassis", map[string]interface{}{
				"@odata.type": "#ChassisCollection.ChassisCollection",
				"Members": []map[string]string{
					{"@odata.id": "/redfish/v1/Chassis/HGX_Chassis_0"},
				},
				"Members@odata.count": 1,
			})

			// Add chassis
			server.addRoute("/redfish/v1/Chassis/HGX_Chassis_0", map[string]interface{}{
				"@odata.type": "#Chassis.v1_20_0.Chassis",
				"@odata.id":   "/redfish/v1/Chassis/HGX_Chassis_0",
				"Id":          "HGX_Chassis_0",
				"Name":        "HGX Chassis",
				"ChassisType": "RackMount",
				"Controls": map[string]interface{}{
					"@odata.id": "/redfish/v1/Chassis/HGX_Chassis_0/Controls",
				},
				"Status": map[string]string{
					"State":  "Enabled",
					"Health": "OK",
				},
			})

			// Add controls collection
			server.addRoute("/redfish/v1/Chassis/HGX_Chassis_0/Controls", map[string]interface{}{
				"@odata.type": "#ControlCollection.ControlCollection",
				"Members": []map[string]string{
					{"@odata.id": "/redfish/v1/Chassis/HGX_Chassis_0/Controls/TotalGPU_Power_0"},
				},
				"Members@odata.count": 1,
			})

			// Add control with test-specific response
			tc.controlResponse["@odata.id"] = "/redfish/v1/Chassis/HGX_Chassis_0/Controls/TotalGPU_Power_0"
			server.addRoute("/redfish/v1/Chassis/HGX_Chassis_0/Controls/TotalGPU_Power_0", tc.controlResponse)

			client := connectToTestServer(t, server.Server)
			defer client.Logout()

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			collector, err := NewChassisCollector(t.Name(), client, logger, config.DefaultChassisCollector)
			require.NoError(t, err)

			// Collect metrics
			ch := make(chan prometheus.Metric, 100)
			go func() {
				collector.Collect(ch)
				close(ch)
			}()

			// Check for GPU power metric
			foundGPUPower := false
			var gpuPowerValue float64

			for metric := range ch {
				desc := metric.Desc()
				if strings.Contains(desc.String(), "gpu_total_power_watts") {
					foundGPUPower = true
					dto := &dto.Metric{}
					require.NoError(t, metric.Write(dto))
					gpuPowerValue = dto.Gauge.GetValue()
					break
				}
			}

			if tc.expectMetric {
				require.True(t, foundGPUPower, "Expected GPU power metric to be collected")
				require.InDelta(t, tc.expectedValue, gpuPowerValue, 0.01, "GPU power value should match")
			} else {
				require.False(t, foundGPUPower, "GPU power metric should not be collected")
			}
		})
	}
}
