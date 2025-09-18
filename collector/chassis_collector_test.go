package collector

import (
	"io"
	"log/slog"
	"sync"
	"testing"

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

				client := connectToTestServer(t, server)

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

				client := connectToTestServer(t, server)

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
			collector := NewChassisCollector(client, logger)
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

	client := connectToTestServer(t, server)
	t.Cleanup(func() {
		client.Logout()
		server.Close()
	})

	service := client.GetService()
	chassis, err := service.Chassis()
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector := NewChassisCollector(client, logger)
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
