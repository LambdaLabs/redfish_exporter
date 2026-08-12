package collector

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"path"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stmcginnis/gofish/schemas"
	"github.com/stretchr/testify/require"
)

// collectedMetric is a drained prometheus.Metric flattened for assertions.
type collectedMetric struct {
	fqName string
	labels map[string]string
	value  float64
}

// drainMetrics closes ch and flattens everything on it, keyed by metric name. A metric
// name may appear more than once when a family covers several sensors or detectors.
func drainMetrics(t *testing.T, ch chan prometheus.Metric) map[string][]collectedMetric {
	t.Helper()
	close(ch)

	out := map[string][]collectedMetric{}
	for metric := range ch {
		d := &dto.Metric{}
		require.NoError(t, metric.Write(d))

		match := fqNameRe.FindStringSubmatch(metric.Desc().String())
		require.Len(t, match, 2, "could not extract fqName from %s", metric.Desc().String())

		labels := map[string]string{}
		for _, label := range d.Label {
			labels[label.GetName()] = label.GetValue()
		}

		require.NotNil(t, d.Gauge, "expected gauge metric for %s", match[1])
		out[match[1]] = append(out[match[1]], collectedMetric{
			fqName: match[1],
			labels: labels,
			value:  d.Gauge.GetValue(),
		})
	}
	return out
}

// requireMetric asserts exactly one sample exists for name and returns it.
func requireMetric(t *testing.T, metrics map[string][]collectedMetric, name string) collectedMetric {
	t.Helper()
	samples, ok := metrics[name]
	require.True(t, ok, "expected metric %s to be emitted, got %v", name, slices.Sorted(maps.Keys(metrics)))
	require.Len(t, samples, 1, "expected exactly one sample for %s", name)
	return samples[0]
}

func TestGetLeakDetection(t *testing.T) {
	tT := map[string]struct {
		setupFn                 func(*testRedfishServer)
		expectedLDs             int
		expectedLeakDetectionID string
		expectNilLeakDetection  bool
	}{
		"canonical DMTF placement under ThermalSubsystem": {
			setupFn: func(server *testRedfishServer) {
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem", "thermal_subsystem.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection", "leak_detection.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors", "leak_detectors_collection.json")
				for _, id := range []string{"Chassis_0_LeakDetector_0_ColdPlate", "Chassis_0_LeakDetector_0_Manifold", "Chassis_0_LeakDetector_1_ColdPlate"} {
					server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors/"+id, "leak_detector_ok.json")
				}
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors/Chassis_0_LeakDetector_1_Manifold", "leak_detector_leak.json")
			},
			expectedLDs:             4,
			expectedLeakDetectionID: "LeakDetection",
		},
		// Regression test: the LeakDetection @odata.id is NOT <ThermalSubsystem>/LeakDetection.
		// An implementation that re-derives the URL by string concatenation rather than
		// following the advertised link finds nothing here.
		"LeakDetection at a vendor-specific @odata.id": {
			setupFn: func(server *testRedfishServer) {
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem", "thermal_subsystem_oem_leak_path.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Oem/Vendor/LeakDetectionSystem", "leak_detection_oem_path.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Oem/Vendor/LeakDetectionSystem/LeakDetectors", "leak_detectors_collection_oem_path.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Oem/Vendor/LeakDetectionSystem/LeakDetectors/Chassis_0_LeakDetector_0_ColdPlate", "leak_detector_ok.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Oem/Vendor/LeakDetectionSystem/LeakDetectors/Chassis_0_LeakDetector_1_Manifold", "leak_detector_leak.json")
			},
			expectedLDs:             2,
			expectedLeakDetectionID: "LeakDetectionSystem",
		},
		// Observed on the NVIDIA MGX NVSwitch tray (P3809), where 6 of its 7
		// LeakDetection-bearing chassis expose an empty LeakDetectors collection and only
		// one carries real detectors. The subsystem rollup is still meaningful.
		"LeakDetection present with an empty detector collection": {
			setupFn: func(server *testRedfishServer) {
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem", "thermal_subsystem.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection", "leak_detection.json")
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors", "leak_detectors_collection_empty.json")
			},
			expectedLDs:             0,
			expectedLeakDetectionID: "LeakDetection",
		},
		// Observed on SYS-A21GE-NBRT: ThermalSubsystem advertises LeakDetection but the
		// GET returns 404. This must degrade quietly rather than fail the scrape.
		"LeakDetection advertised but returns 404": {
			setupFn: func(server *testRedfishServer) {
				server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem", "thermal_subsystem.json")
				server.addErrorRoute("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection", http.StatusNotFound)
			},
			expectedLDs:            0,
			expectNilLeakDetection: true,
		},
	}

	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			server := newTestRedfishServer(t)
			server.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
			server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "chassis_main.json")
			test.setupFn(server)

			client := connectToTestServer(t, server.Server)
			t.Cleanup(func() {
				client.Logout()
				server.Close()
			})

			chassis, err := client.GetService().Chassis()
			require.NoError(t, err)
			require.NotEmpty(t, chassis, "Expected at least one chassis")

			logger := NewTestLogger(t, slog.LevelDebug)
			collector, err := NewChassisCollector(t.Name(), client, logger, config.DefaultChassisCollector)
			require.NoError(t, err)
			thermalSubsystem, err := chassis[0].ThermalSubsystem()
			require.NoError(t, err)

			leakDetection, detectors := collector.getLeakDetection(thermalSubsystem, logger)

			require.Len(t, detectors, test.expectedLDs)
			if test.expectNilLeakDetection {
				require.Nil(t, leakDetection)
				return
			}
			require.NotNil(t, leakDetection)
			require.Equal(t, test.expectedLeakDetectionID, leakDetection.ID)
		})
	}
}

func TestParseLeakDetector(t *testing.T) {
	tT := map[string]struct {
		fixture               string
		expectedState         float64
		expectedHealth        float64
		expectedDetectorState string
	}{
		"detector reporting no leak": {
			fixture:               "leak_detector_ok.json",
			expectedState:         1,
			expectedHealth:        1,
			expectedDetectorState: "OK",
		},
		"detector reporting a critical leak": {
			fixture:               "leak_detector_leak.json",
			expectedState:         3,
			expectedHealth:        3,
			expectedDetectorState: "Critical",
		},
	}

	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			var detector schemas.LeakDetector
			loadTestDataInto(t, test.fixture, &detector)
			require.Equal(t, test.expectedDetectorState, string(detector.DetectorState),
				"fixture must use a DetectorState value from the Redfish enum")

			metricsCh := make(chan prometheus.Metric, 32)
			parseLeakDetector(metricsCh, "test_chassis", "LeakDetection", &detector)
			metrics := drainMetrics(t, metricsCh)

			state := requireMetric(t, metrics, "redfish_chassis_leak_detector_state")
			require.Equal(t, test.expectedState, state.value)
			require.Equal(t, "test_chassis", state.labels["chassis_id"])
			require.Equal(t, "LeakDetection", state.labels["leak_detection_id"])
			require.Equal(t, detector.ID, state.labels["leak_detector_id"])
			require.Equal(t, "leak_detector", state.labels["resource"])

			require.Equal(t, test.expectedHealth, requireMetric(t, metrics, "redfish_chassis_leak_detector_health").value)

			info := requireMetric(t, metrics, "redfish_chassis_leak_detector_info")
			require.Equal(t, float64(1), info.value)
			require.Equal(t, "Moisture", info.labels["leak_detector_type"])
			require.Equal(t, "CoolingSubsystem", info.labels["physical_context"])
		})
	}
}

// TestParseLeakDetectorEnabled covers firmware that omits the Enabled property. gofish
// types it as a plain bool, so reporting it unconditionally would label every healthy
// detector on a GB300 tray as disabled.
func TestParseLeakDetectorEnabled(t *testing.T) {
	t.Run("omitted by firmware is not reported", func(t *testing.T) {
		// The real ARS-121GL-NB3 detector body: no Enabled property.
		raw := []byte(`{
			"Id": "Chassis_0_LeakDetector_0_ColdPlate",
			"Name": "Chassis 0 LeakDetector 0 ColdPlate",
			"LeakDetectorType": "Moisture",
			"DetectorState": "OK",
			"Status": {"Health": "OK", "State": "Enabled"}
		}`)
		var detector schemas.LeakDetector
		require.NoError(t, json.Unmarshal(raw, &detector))
		require.Equal(t, raw, []byte(detector.RawData), "gofish should retain the raw payload")

		_, reported := leakDetectorEnabled(&detector)
		require.False(t, reported, "an absent Enabled property must not be reported")

		ch := make(chan prometheus.Metric, 32)
		parseLeakDetector(ch, "Chassis_0", "LeakDetection", &detector)
		metrics := drainMetrics(t, ch)
		require.NotContains(t, metrics, "redfish_chassis_leak_detector_enabled")
		// The leak signal itself must still be emitted.
		require.Equal(t, float64(1), requireMetric(t, metrics, "redfish_chassis_leak_detector_state").value)
	})

	t.Run("explicitly disabled is reported as zero", func(t *testing.T) {
		raw := []byte(`{
			"Id": "Chassis_0_LeakDetector_0_ColdPlate",
			"DetectorState": "Unavailable",
			"Enabled": false,
			"Status": {"Health": "OK", "State": "Disabled"}
		}`)
		var detector schemas.LeakDetector
		require.NoError(t, json.Unmarshal(raw, &detector))

		enabled, reported := leakDetectorEnabled(&detector)
		require.True(t, reported)
		require.False(t, enabled)

		ch := make(chan prometheus.Metric, 32)
		parseLeakDetector(ch, "Chassis_0", "LeakDetection", &detector)
		metrics := drainMetrics(t, ch)
		require.Equal(t, float64(0), requireMetric(t, metrics, "redfish_chassis_leak_detector_enabled").value)
		// A disabled detector reports Unavailable, which must not read as worse than Critical.
		require.Equal(t, float64(4), requireMetric(t, metrics, "redfish_chassis_leak_detector_state").value)
	})

	t.Run("explicitly enabled is reported as one", func(t *testing.T) {
		var detector schemas.LeakDetector
		loadTestDataInto(t, "leak_detector_ok.json", &detector)

		enabled, reported := leakDetectorEnabled(&detector)
		require.True(t, reported)
		require.True(t, enabled)

		ch := make(chan prometheus.Metric, 32)
		parseLeakDetector(ch, "Chassis_0", "LeakDetection", &detector)
		metrics := drainMetrics(t, ch)
		require.Equal(t, float64(1), requireMetric(t, metrics, "redfish_chassis_leak_detector_enabled").value)
	})
}

// TestParseLeakDetectorUnavailableIsNotWorseThanCritical documents why alert expressions
// on leak_detector_state must use equality: Unavailable and Absent sort above Critical.
func TestParseLeakDetectorUnavailableIsNotWorseThanCritical(t *testing.T) {
	for state, want := range map[schemas.DetectorState]float64{
		schemas.OKDetectorState:          1,
		schemas.WarningDetectorState:     2,
		schemas.CriticalDetectorState:    3,
		schemas.UnavailableDetectorState: 4,
		schemas.AbsentDetectorState:      5,
	} {
		got, ok := parseDetectorState(state)
		require.True(t, ok, "state %q should map to a value", state)
		require.Equal(t, want, got, "state %q", state)
	}

	_, ok := parseDetectorState(schemas.DetectorState("LeakDetected"))
	require.False(t, ok, "a value outside the Redfish enum must not be emitted")
}

// TestGetChassisSensorsUsesExpand asserts the collector asks the BMC to inline sensor
// bodies. Without $expand, gofish issues one request per sensor, which on a tray with a
// few hundred sensors and max_concurrent_requests=1 serialises into a very long scrape.
func TestGetChassisSensorsUsesExpand(t *testing.T) {
	server := newTestRedfishServer(t)
	server.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "chassis_main.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Sensors", "chassis_sensors_expanded.json")

	var expandedRequests int
	server.mux.HandleFunc("/redfish/v1/Chassis/Chassis_0/Sensors/", func(w http.ResponseWriter, r *http.Request) {
		expandedRequests++
		w.WriteHeader(http.StatusInternalServerError)
	})

	client := connectToTestServer(t, server.Server)
	t.Cleanup(func() {
		client.Logout()
		server.Close()
	})

	chassis, err := client.GetService().Chassis()
	require.NoError(t, err)

	collector, err := NewChassisCollector(t.Name(), client, NewTestLogger(t, slog.LevelDebug), config.DefaultChassisCollector)
	require.NoError(t, err)

	links := chassisAdvertisedLinks(chassis[0])
	sensors, err := collector.getChassisSensors(context.Background(), links.sensorsPath(chassis[0]), nil, NewTestLogger(t, slog.LevelDebug))
	require.NoError(t, err)
	require.Len(t, sensors, 6, "all sensors should arrive from the single expanded request")
	require.Zero(t, expandedRequests, "no per-sensor requests should be issued when $expand is honoured")
}

// TestGetChassisSensorsSkipsChassisWithoutSensors covers the ERoT/IRoT chassis that make up
// roughly a third of an NVL72 tray or HGX baseboard: they advertise no Sensors collection,
// and synthesising "<chassis>/Sensors" for them cost a 404 on every scrape.
func TestGetChassisSensorsSkipsChassisWithoutSensors(t *testing.T) {
	server := newTestRedfishServer(t)
	server.addRoute("/redfish/v1/Chassis", map[string]any{
		"@odata.id":   "/redfish/v1/Chassis",
		"@odata.type": "#ChassisCollection.ChassisCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Chassis/HGX_ERoT_NVSwitch_0"},
		},
	})
	// A real ERoT root: no Thermal, no Power, and no Sensors either.
	server.addRoute("/redfish/v1/Chassis/HGX_ERoT_NVSwitch_0", map[string]any{
		"@odata.id":   "/redfish/v1/Chassis/HGX_ERoT_NVSwitch_0",
		"@odata.type": "#Chassis.v1_22_0.Chassis",
		"Id":          "HGX_ERoT_NVSwitch_0",
		"ChassisType": "Component",
		"Status":      map[string]any{"Health": "OK", "State": "Enabled"},
	})

	var sensorRequests int
	server.mux.HandleFunc("/redfish/v1/Chassis/HGX_ERoT_NVSwitch_0/Sensors", func(w http.ResponseWriter, r *http.Request) {
		sensorRequests++
		w.WriteHeader(http.StatusNotFound)
	})

	client := connectToTestServer(t, server.Server)
	t.Cleanup(func() {
		client.Logout()
		server.Close()
	})

	collector, err := NewChassisCollector(t.Name(), client, NewTestLogger(t, slog.LevelDebug), config.DefaultChassisCollector)
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 64)
	collector.CollectWithContext(context.Background(), ch)
	drainMetrics(t, ch)

	require.Zero(t, sensorRequests, "a chassis advertising no Sensors collection must not be asked for one")
}

// TestGetChassisSensorsBoundedWhenExpandIgnored covers a BMC that answers the Sensors
// collection with link-only members. Fanning out to one request per sensor there would
// multiply request load against the BMCs least able to absorb it, so only the named
// sensors — the leak detectors — are fetched and the bulk telemetry is skipped.
func TestGetChassisSensorsBoundedWhenExpandIgnored(t *testing.T) {
	server := newTestRedfishServer(t)
	server.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "chassis_main.json")

	// Link-only members, as a BMC that ignores $expand would return.
	server.addRoute("/redfish/v1/Chassis/Chassis_0/Sensors", map[string]any{
		"@odata.id":           "/redfish/v1/Chassis/Chassis_0/Sensors",
		"@odata.type":         "#SensorCollection.SensorCollection",
		"Name":                "Sensor Collection",
		"Members@odata.count": 3,
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Chassis/Chassis_0/Sensors/Chassis_0_LeakDetector_0_ColdPlate"},
			{"@odata.id": "/redfish/v1/Chassis/Chassis_0/Sensors/Chassis_0_FAN_1_FRONT"},
			{"@odata.id": "/redfish/v1/Chassis/Chassis_0/Sensors/Chassis_0_Front_IO_Temp_0"},
		},
	})

	var perSensorRequests []string
	for _, id := range []string{"Chassis_0_LeakDetector_0_ColdPlate", "Chassis_0_FAN_1_FRONT", "Chassis_0_Front_IO_Temp_0"} {
		sensorID := id
		server.mux.HandleFunc("/redfish/v1/Chassis/Chassis_0/Sensors/"+sensorID, func(w http.ResponseWriter, r *http.Request) {
			perSensorRequests = append(perSensorRequests, sensorID)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"@odata.id":"/redfish/v1/Chassis/Chassis_0/Sensors/` + sensorID + `","Id":"` + sensorID + `","Name":"` + sensorID + `","ReadingType":"Voltage","Reading":1.7,"Thresholds":{"LowerCritical":{"Reading":1.65}}}`))
		})
	}

	client := connectToTestServer(t, server.Server)
	t.Cleanup(func() {
		client.Logout()
		server.Close()
	})

	chassis, err := client.GetService().Chassis()
	require.NoError(t, err)

	collector, err := NewChassisCollector(t.Name(), client, NewTestLogger(t, slog.LevelDebug), config.DefaultChassisCollector)
	require.NoError(t, err)

	required := map[string]string{"Chassis_0_LeakDetector_0_ColdPlate": "LeakDetection"}
	links := chassisAdvertisedLinks(chassis[0])
	sensors, err := collector.getChassisSensors(context.Background(), links.sensorsPath(chassis[0]), required, NewTestLogger(t, slog.LevelDebug))
	require.NoError(t, err)

	require.Len(t, sensors, 1, "only the required sensor should be fetched")
	require.Equal(t, "Chassis_0_LeakDetector_0_ColdPlate", sensors[0].ID)
	require.Equal(t, []string{"Chassis_0_LeakDetector_0_ColdPlate"}, perSensorRequests,
		"the fan and temperature sensors must not be fetched individually")
}

// TestGetNamedSensorsSkipsDetectorsWithoutSensors covers the MGX NVLink switch tray, which
// carries seven leak detectors and exactly one sensor. Synthesising "<Sensors>/<detector Id>"
// for each detector there is seven guaranteed 404s per chassis per scrape, so the fallback
// fetches only the detectors the collection actually lists.
func TestGetNamedSensorsSkipsDetectorsWithoutSensors(t *testing.T) {
	server := newTestRedfishServer(t)
	server.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "chassis_main.json")

	// A BMC that ignores $expand, listing one sensor against several detectors.
	server.addRoute("/redfish/v1/Chassis/Chassis_0/Sensors", map[string]any{
		"@odata.id":   "/redfish/v1/Chassis/Chassis_0/Sensors",
		"@odata.type": "#SensorCollection.SensorCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Chassis/Chassis_0/Sensors/leakage1"},
		},
	})

	var perSensorRequests []string
	server.mux.HandleFunc("/redfish/v1/Chassis/Chassis_0/Sensors/", func(w http.ResponseWriter, r *http.Request) {
		id := path.Base(r.URL.Path)
		perSensorRequests = append(perSensorRequests, id)
		if id != "leakage1" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"@odata.id":"/redfish/v1/Chassis/Chassis_0/Sensors/leakage1","Id":"leakage1","Name":"leakage1","ReadingType":"Voltage","Reading":1.7}`))
	})

	client := connectToTestServer(t, server.Server)
	t.Cleanup(func() {
		client.Logout()
		server.Close()
	})

	collector, err := NewChassisCollector(t.Name(), client, NewTestLogger(t, slog.LevelDebug), config.DefaultChassisCollector)
	require.NoError(t, err)

	required := map[string]string{
		"leakage1": "LeakDetection", "leakage2": "LeakDetection", "leakage3": "LeakDetection",
		"leakage4": "LeakDetection", "leakage5": "LeakDetection", "leakage_aggr": "LeakDetection",
	}
	sensors, err := collector.getChassisSensors(context.Background(), "/redfish/v1/Chassis/Chassis_0/Sensors", required, NewTestLogger(t, slog.LevelDebug))
	require.NoError(t, err)

	require.Len(t, sensors, 1)
	require.Equal(t, "leakage1", sensors[0].ID)
	require.Equal(t, []string{"leakage1"}, perSensorRequests,
		"detectors with no companion sensor must not be requested")
}

// TestListChassisFiltersBeforeFetching pins the saving that makes the leak_detection module
// cheap enough to poll on a short interval. skipChassis alone filters after every chassis
// body has already been paid for, which on a GB300 tray is forty-two fetches to look at one.
func TestListChassisFiltersBeforeFetching(t *testing.T) {
	members := []map[string]string{
		{"@odata.id": "/redfish/v1/Chassis/Chassis_0"},
		{"@odata.id": "/redfish/v1/Chassis/HGX_GPU_0"},
		{"@odata.id": "/redfish/v1/Chassis/HGX_ERoT_BMC_0"},
	}

	newServer := func(t *testing.T) (*testRedfishServer, *[]string) {
		t.Helper()
		server := newTestRedfishServer(t)
		server.addRoute("/redfish/v1/Chassis", map[string]any{
			"@odata.id":   "/redfish/v1/Chassis",
			"@odata.type": "#ChassisCollection.ChassisCollection",
			"Members":     members,
		})
		fetched := &[]string{}
		for _, member := range members {
			uri := member["@odata.id"]
			id := path.Base(uri)
			server.mux.HandleFunc(uri, func(w http.ResponseWriter, r *http.Request) {
				*fetched = append(*fetched, id)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"@odata.id":"` + uri + `","@odata.type":"#Chassis.v1_22_0.Chassis","Id":"` + id + `","Status":{"Health":"OK","State":"Enabled"}}`))
			})
		}
		return server, fetched
	}

	t.Run("an include pattern fetches only matching chassis", func(t *testing.T) {
		server, fetched := newServer(t)
		client := connectToTestServer(t, server.Server)
		t.Cleanup(func() { client.Logout(); server.Close() })

		collector, err := NewChassisCollector(t.Name(), client, NewTestLogger(t, slog.LevelDebug), config.ChassisCollectorConfig{
			ChassisInclude: "^Chassis_[0-9]+$",
		})
		require.NoError(t, err)

		chassises, err := collector.listChassis(context.Background(), NewTestLogger(t, slog.LevelDebug))
		require.NoError(t, err)
		require.Len(t, chassises, 1)
		require.Equal(t, "Chassis_0", chassises[0].ID)
		require.Equal(t, []string{"Chassis_0"}, *fetched,
			"the filtered-out chassis bodies must never be requested")
	})

	t.Run("an exclude pattern skips matching chassis", func(t *testing.T) {
		server, fetched := newServer(t)
		client := connectToTestServer(t, server.Server)
		t.Cleanup(func() { client.Logout(); server.Close() })

		collector, err := NewChassisCollector(t.Name(), client, NewTestLogger(t, slog.LevelDebug), config.ChassisCollectorConfig{
			ChassisExclude: "^HGX_",
		})
		require.NoError(t, err)

		chassises, err := collector.listChassis(context.Background(), NewTestLogger(t, slog.LevelDebug))
		require.NoError(t, err)
		require.Len(t, chassises, 1)
		require.Equal(t, []string{"Chassis_0"}, *fetched)
	})

	// No filter must keep the historical listing, which costs one request fewer: there is
	// nothing to decide, so there is no reason to read the service root for the link.
	t.Run("no filter fetches every chassis", func(t *testing.T) {
		server, fetched := newServer(t)
		client := connectToTestServer(t, server.Server)
		t.Cleanup(func() { client.Logout(); server.Close() })

		collector, err := NewChassisCollector(t.Name(), client, NewTestLogger(t, slog.LevelDebug), config.DefaultChassisCollector)
		require.NoError(t, err)

		chassises, err := collector.listChassis(context.Background(), NewTestLogger(t, slog.LevelDebug))
		require.NoError(t, err)
		require.Len(t, chassises, 3)
		require.Len(t, *fetched, 3)
	})
}

// TestChassisAdvertisedLinks pins the feature detection to the advertised links. Deriving
// it from whether a fetch returned data made every chassis look modern as soon as Thermal
// or Power was disabled by configuration, pulling in whole Sensors collections.
func TestChassisAdvertisedLinks(t *testing.T) {
	tT := map[string]struct {
		raw         string
		wantLegacy  bool
		wantSensors string
	}{
		"legacy chassis linking Thermal and Power": {
			raw:        `{"Id":"1","Thermal":{"@odata.id":"/redfish/v1/Chassis/1/Thermal"},"Power":{"@odata.id":"/redfish/v1/Chassis/1/Power"}}`,
			wantLegacy: true,
		},
		"Thermal only": {
			raw:        `{"Id":"1","Thermal":{"@odata.id":"/redfish/v1/Chassis/1/Thermal"}}`,
			wantLegacy: true,
		},
		// The real ARS-121GL-NB3 tray shelf: ThermalSubsystem and Sensors, no Thermal/Power.
		"NVL72 tray chassis": {
			raw:         `{"Id":"Chassis_0","ThermalSubsystem":{"@odata.id":"/redfish/v1/Chassis/Chassis_0/ThermalSubsystem"},"Sensors":{"@odata.id":"/redfish/v1/Chassis/Chassis_0/Sensors"},"PowerSubsystem":{"@odata.id":"/redfish/v1/Chassis/Chassis_0/PowerSubsystem"}}`,
			wantLegacy:  false,
			wantSensors: "/redfish/v1/Chassis/Chassis_0/Sensors",
		},
		// An ERoT root. No Sensors link means there is no collection to request.
		"root of trust chassis": {
			raw:         `{"Id":"HGX_ERoT_NVSwitch_0","@odata.id":"/redfish/v1/Chassis/HGX_ERoT_NVSwitch_0"}`,
			wantLegacy:  false,
			wantSensors: "",
		},
		"empty link objects do not count": {
			raw:        `{"Id":"1","Thermal":{},"Power":{},"Sensors":{}}`,
			wantLegacy: false,
		},
	}

	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			var chassis schemas.Chassis
			require.NoError(t, json.Unmarshal([]byte(test.raw), &chassis))
			links := chassisAdvertisedLinks(&chassis)
			require.Equal(t, test.wantLegacy, links.legacyThermalOrPower())
			require.Equal(t, test.wantSensors, links.sensorsPath(&chassis))
		})
	}

	// A chassis with no retained payload is treated conservatively as legacy, so the
	// Sensors fallback is suppressed rather than issuing a request per chassis. Its leak
	// detectors are still reachable at the conventional path.
	opaque := chassisAdvertisedLinks(&schemas.Chassis{})
	require.True(t, opaque.legacyThermalOrPower())
	require.Equal(t, "/Sensors", opaque.sensorsPath(&schemas.Chassis{}))
}

func TestParseChassisSensor(t *testing.T) {
	var collection sensorCollection
	loadTestDataInto(t, "chassis_sensors_expanded.json", &collection)
	require.Len(t, collection.Members, 6)

	byID := map[string]*schemas.Sensor{}
	for _, sensor := range collection.Members {
		byID[sensor.ID] = sensor
	}

	// Mirrors the collect loop: leak detectors are enumerated from ThermalSubsystem
	// before the Sensors pass, so their companion voltage sensors are recognised.
	leakDetectorIDs := map[string]string{"Chassis_0_LeakDetector_0_ColdPlate": "LeakDetection"}

	t.Run("leak detector voltage is correlated by detector id", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 32)
		parseChassisSensor(ch, "Chassis_0", byID["Chassis_0_LeakDetector_0_ColdPlate"], leakDetectorIDs)
		metrics := drainMetrics(t, ch)

		volts := requireMetric(t, metrics, "redfish_chassis_leak_detector_volts")
		require.InDelta(t, 1.724, volts.value, 1e-9)
		require.Equal(t, "leak_detector", volts.labels["resource"])
		require.Equal(t, "Chassis_0_LeakDetector_0_ColdPlate", volts.labels["leak_detector_id"])

		// The threshold is only available from the Sensor resource, never from the
		// telemetry metric report, which is why the alert can be written relative to it.
		// Firmware disagrees about the value — 1.3 V and 1.65 V both appear across
		// captured ARS-121GL-NB3 trays — so it has to be read, not assumed.
		threshold := requireMetric(t, metrics, "redfish_chassis_leak_detector_volts_lower_threshold_critical")
		require.InDelta(t, 1.65, threshold.value, 1e-9)

		// The upper bound is the open/short indicator: a disconnected sense line reads
		// high, which is a detector that can no longer see a leak.
		upper := requireMetric(t, metrics, "redfish_chassis_leak_detector_volts_upper_threshold_critical")
		require.InDelta(t, 2.0, upper.value, 1e-9)

		// A leak sensor must not also appear as a generic chassis voltage.
		require.NotContains(t, metrics, "redfish_chassis_power_voltage_volts")
	})

	// Firmware that reports only one bound must leave the other absent. Defaulting it to
	// zero would turn "reading <= lower_threshold" into a comparison that can never fire,
	// which reads as a healthy alert rather than as a missing one.
	t.Run("an unreported threshold is omitted rather than zeroed", func(t *testing.T) {
		var sensor schemas.Sensor
		require.NoError(t, json.Unmarshal([]byte(`{
			"Id":"leakage1","Name":"leakage1","ReadingType":"Voltage","Reading":1.7
		}`), &sensor))

		ch := make(chan prometheus.Metric, 32)
		parseChassisSensor(ch, "MGX_BMC_0", &sensor, map[string]string{"leakage1": "LeakDetection"})
		metrics := drainMetrics(t, ch)

		require.Contains(t, metrics, "redfish_chassis_leak_detector_volts")
		require.NotContains(t, metrics, "redfish_chassis_leak_detector_volts_lower_threshold_critical")
		require.NotContains(t, metrics, "redfish_chassis_leak_detector_volts_upper_threshold_critical")
	})

	t.Run("temperature folds into the existing chassis family", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 32)
		parseChassisSensor(ch, "Chassis_0", byID["Chassis_0_Front_IO_Temp_0"], leakDetectorIDs)
		metrics := drainMetrics(t, ch)

		celsius := requireMetric(t, metrics, "redfish_chassis_temperature_celsius")
		require.InDelta(t, 27.5, celsius.value, 1e-9)
		require.Equal(t, "temperature", celsius.labels["resource"])
		require.Equal(t, "Chassis 0 Front IO Temp 0", celsius.labels["sensor"])
		require.Equal(t, "Chassis_0_Front_IO_Temp_0", celsius.labels["sensor_id"])
		require.Equal(t, float64(1), requireMetric(t, metrics, "redfish_chassis_temperature_sensor_health").value)
	})

	t.Run("rotational folds into the existing fan family with thresholds", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 32)
		parseChassisSensor(ch, "Chassis_0", byID["Chassis_0_FAN_1_FRONT"], leakDetectorIDs)
		metrics := drainMetrics(t, ch)

		rpm := requireMetric(t, metrics, "redfish_chassis_fan_rpm")
		require.InDelta(t, 8623.0, rpm.value, 1e-9)
		require.Equal(t, "fan", rpm.labels["resource"])
		require.Equal(t, "rpm", rpm.labels["fan_unit"])

		// The existing ChassisFanCritical alert compares these two series directly, so a
		// Sensors-only platform must populate both for that alert to work there.
		require.InDelta(t, 2204.0, requireMetric(t, metrics, "redfish_chassis_fan_rpm_lower_threshold_critical").value, 1e-9)
		require.InDelta(t, 23.056, requireMetric(t, metrics, "redfish_chassis_fan_rpm_percentage").value, 0.001)
	})

	// Fan PWM duty cycle and CPU core utilisation are both ReadingType "Percent" and
	// neither carries a distinguishing PhysicalContext, so Percent must not be assumed
	// to be a fan speed.
	t.Run("percent is not assumed to be a fan speed", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 32)
		parseChassisSensor(ch, "Chassis_0", byID["Chassis_0_FAN_1_PWM"], leakDetectorIDs)
		metrics := drainMetrics(t, ch)

		require.NotContains(t, metrics, "redfish_chassis_fan_rpm_percentage")
		percent := requireMetric(t, metrics, "redfish_chassis_sensor_percent")
		require.InDelta(t, 29.803921, percent.value, 1e-5)
		require.Equal(t, "%", percent.labels["sensor_units"])
	})

	t.Run("power and unrecognised reading types reach a catch-all", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 32)
		parseChassisSensor(ch, "Chassis_0", byID["Chassis_0_TotalHSC_Power_0"], leakDetectorIDs)
		metrics := drainMetrics(t, ch)
		require.InDelta(t, 1171.349692, requireMetric(t, metrics, "redfish_chassis_sensor_watts").value, 1e-6)

		// Altitude has no dedicated metric; it must still be emitted rather than dropped.
		ch = make(chan prometheus.Metric, 32)
		parseChassisSensor(ch, "Chassis_0", byID["Chassis_0_Altitude_0"], leakDetectorIDs)
		metrics = drainMetrics(t, ch)
		reading := requireMetric(t, metrics, "redfish_chassis_sensor_reading")
		require.InDelta(t, 132.0, reading.value, 1e-9)
		require.Equal(t, "m", reading.labels["sensor_units"])
	})
}

// newLeakChassisServer wires a chassis that carries both leak detectors and a full Sensors
// collection, which is the shape of an ARS-121GL-NB3 tray shelf. withThermal additionally
// gives it the deprecated Thermal schema, which no captured tray has but a CDU or rack
// manager plausibly would.
func newLeakChassisServer(t *testing.T, withThermal bool) *testRedfishServer {
	t.Helper()
	server := newTestRedfishServer(t)
	server.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")

	chassis := server.loadFixture("chassis_main.json")
	if withThermal {
		chassis["Thermal"] = map[string]string{"@odata.id": "/redfish/v1/Chassis/Chassis_0/Thermal"}
		server.addRoute("/redfish/v1/Chassis/Chassis_0/Thermal", map[string]any{
			"@odata.id":   "/redfish/v1/Chassis/Chassis_0/Thermal",
			"@odata.type": "#Thermal.v1_7_1.Thermal",
			"Id":          "Thermal",
			"Temperatures": []map[string]any{{
				"MemberId":       "0",
				"Name":           "Chassis 0 Front IO Temp 0",
				"ReadingCelsius": 27.5,
				"Status":         map[string]any{"Health": "OK", "State": "Enabled"},
			}},
		})
	}
	server.addRoute("/redfish/v1/Chassis/Chassis_0", chassis)

	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem", "thermal_subsystem.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection", "leak_detection.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors", "leak_detectors_collection.json")
	for _, id := range []string{"Chassis_0_LeakDetector_0_ColdPlate", "Chassis_0_LeakDetector_0_Manifold", "Chassis_0_LeakDetector_1_ColdPlate", "Chassis_0_LeakDetector_1_Manifold"} {
		server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors/"+id, "leak_detector_ok.json")
	}
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Sensors", "chassis_sensors_expanded.json")
	return server
}

// collectChassis runs a full chassis collection against a wired-up test server.
func collectChassis(t *testing.T, server *testRedfishServer, cfg config.ChassisCollectorConfig) map[string][]collectedMetric {
	t.Helper()
	client := connectToTestServer(t, server.Server)
	t.Cleanup(func() {
		client.Logout()
		server.Close()
	})

	collector, err := NewChassisCollector(t.Name(), client, NewTestLogger(t, slog.LevelDebug), cfg)
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 512)
	collector.CollectWithContext(context.Background(), ch)
	return drainMetrics(t, ch)
}

func TestChassisCollectorSensorScope(t *testing.T) {
	// A chassis with no Thermal or Power is the reason the Sensors pass exists: it stands
	// in for both, so everything in the collection is emitted.
	t.Run("a sensors-only chassis emits the whole collection", func(t *testing.T) {
		metrics := collectChassis(t, newLeakChassisServer(t, false), config.DefaultChassisCollector)

		require.Contains(t, metrics, "redfish_chassis_temperature_celsius")
		require.Contains(t, metrics, "redfish_chassis_fan_rpm")
		require.Contains(t, metrics, "redfish_chassis_sensor_watts")
		require.Len(t, metrics["redfish_chassis_leak_detector_volts"], 1)
		require.Len(t, metrics["redfish_chassis_leak_detector_state"], 4)
	})

	// Disabling both deprecated schemas is what makes the leak_detection module cheap. On
	// these platforms no chassis implements either, so without this the Sensors pass would
	// stand in for what was just turned off and the disables would save almost nothing.
	t.Run("disabling thermal and power narrows the pass to leak detectors", func(t *testing.T) {
		metrics := collectChassis(t, newLeakChassisServer(t, false), config.DefaultModuleConfig["leak_detection"].ChassisCollector)

		require.Len(t, metrics["redfish_chassis_leak_detector_volts"], 1,
			"the leak detector voltage is the whole point of the module and must survive")
		require.Len(t, metrics["redfish_chassis_leak_detector_state"], 4)

		for _, bulk := range []string{
			"redfish_chassis_temperature_celsius",
			"redfish_chassis_fan_rpm",
			"redfish_chassis_sensor_watts",
			"redfish_chassis_sensor_percent",
		} {
			require.NotContains(t, metrics, bulk, "bulk sensor telemetry must not come back under a different schema")
		}
	})

	// A chassis implementing Thermal *and* leak detection consults Sensors only for the
	// detectors. Emitting the rest would publish its temperatures twice under the same
	// series name, which fails the whole scrape at registration.
	t.Run("a legacy chassis with leak detectors emits no duplicate thermal series", func(t *testing.T) {
		metrics := collectChassis(t, newLeakChassisServer(t, true), config.DefaultChassisCollector)

		require.Len(t, metrics["redfish_chassis_temperature_celsius"], 1,
			"the Thermal reading must not be republished from Sensors")
		require.Len(t, metrics["redfish_chassis_leak_detector_volts"], 1,
			"the leak detector voltage is only available from Sensors and must still arrive")
		require.NotContains(t, metrics, "redfish_chassis_sensor_watts")
	})
}

func TestChassisCollectorSensorExclude(t *testing.T) {
	t.Run("excluded sensors emit nothing", func(t *testing.T) {
		metrics := collectChassis(t, newLeakChassisServer(t, false), config.ChassisCollectorConfig{
			SensorExclude: ptr("_PWM$"),
		})
		require.NotContains(t, metrics, "redfish_chassis_sensor_percent")
		require.Contains(t, metrics, "redfish_chassis_sensor_watts")
	})

	// An explicitly empty pattern is how a deployment opts back in to everything,
	// including the per-core utilisation the shipped default declines.
	t.Run("an empty pattern excludes nothing", func(t *testing.T) {
		collector, err := NewChassisCollector(t.Name(), nil, NewTestLogger(t, slog.LevelWarn), config.ChassisCollectorConfig{
			SensorExclude: ptr(""),
		})
		require.NoError(t, err)
		require.False(t, collector.skipSensor("ProcessorModule_0_CPU_0_CoreUtil_0"))
	})

	// A pattern broad enough to catch a leak detector must not silence it. The Sensors
	// collection is consulted for those detectors in the first place, and losing a safety
	// signal to a series-trimming pattern is not a tradeoff worth offering.
	t.Run("leak detectors are never excluded", func(t *testing.T) {
		metrics := collectChassis(t, newLeakChassisServer(t, false), config.ChassisCollectorConfig{
			SensorExclude: ptr("."),
		})
		require.Len(t, metrics["redfish_chassis_leak_detector_volts"], 1)
		require.NotContains(t, metrics, "redfish_chassis_temperature_celsius")
	})

	// The default declines per-core CPU utilisation, which a GB300 tray publishes 144 of.
	// The telemetry collector declines the identical sensors; the two collectors must not
	// disagree about the same hardware.
	//
	// Every construction path must agree on that, or a hand-written chassis_collector
	// block would quietly collect more than the module shipped beside it. The zero value
	// is the case that matters: it is what `chassis_collector: {}` unmarshals to.
	t.Run("every unconfigured path declines per-core utilisation", func(t *testing.T) {
		for tName, cfg := range map[string]config.ChassisCollectorConfig{
			"shipped default":        config.DefaultChassisCollector,
			"zero value":             {},
			"configured but not set": {DisableNetworkAdapters: true},
			"built-in module":        config.DefaultModuleConfig["chassis_collector"].ChassisCollector,
			"leak_detection module":  config.DefaultModuleConfig["leak_detection"].ChassisCollector,
		} {
			t.Run(tName, func(t *testing.T) {
				collector, err := NewChassisCollector(t.Name(), nil, NewTestLogger(t, slog.LevelWarn), cfg)
				require.NoError(t, err)

				require.True(t, collector.skipSensor("ProcessorModule_0_CPU_0_CoreUtil_0"))
				require.True(t, collector.skipSensor("ProcessorModule_1_CPU_0_CoreUtil_71"))
				require.False(t, collector.skipSensor("ProcessorModule_0_CPU_0_CpuFreq_0"))
				require.False(t, collector.skipSensor("Chassis_0_FAN_1_PWM"))
			})
		}
	})
}

func TestChassisCollectorFiltering(t *testing.T) {
	t.Run("include and exclude patterns", func(t *testing.T) {
		tT := map[string]struct {
			include, exclude string
			chassisID        string
			wantSkip         bool
		}{
			"no filters collects everything":   {chassisID: "HGX_GPU_0", wantSkip: false},
			"include matches":                  {include: "^Chassis_[0-9]+$", chassisID: "Chassis_0", wantSkip: false},
			"include does not match":           {include: "^Chassis_[0-9]+$", chassisID: "HGX_GPU_0", wantSkip: true},
			"exclude matches":                  {exclude: "^HGX_", chassisID: "HGX_GPU_0", wantSkip: true},
			"exclude does not match":           {exclude: "^HGX_", chassisID: "Chassis_0", wantSkip: false},
			"exclude takes effect via include": {include: "^Chassis_", exclude: "_1$", chassisID: "Chassis_1", wantSkip: true},
		}
		for tName, test := range tT {
			t.Run(tName, func(t *testing.T) {
				collector, err := NewChassisCollector(t.Name(), nil, NewTestLogger(t, slog.LevelWarn), config.ChassisCollectorConfig{
					ChassisInclude: test.include,
					ChassisExclude: test.exclude,
				})
				require.NoError(t, err)
				require.Equal(t, test.wantSkip, collector.skipChassis(test.chassisID))
			})
		}
	})

	t.Run("an invalid pattern is reported at construction", func(t *testing.T) {
		_, err := NewChassisCollector(t.Name(), nil, NewTestLogger(t, slog.LevelWarn), config.ChassisCollectorConfig{
			ChassisInclude: "([unclosed",
		})
		require.ErrorContains(t, err, "invalid chassis_include pattern")

		_, err = NewChassisCollector(t.Name(), nil, NewTestLogger(t, slog.LevelWarn), config.ChassisCollectorConfig{
			ChassisExclude: "([unclosed",
		})
		require.ErrorContains(t, err, "invalid chassis_exclude pattern")

		_, err = NewChassisCollector(t.Name(), nil, NewTestLogger(t, slog.LevelWarn), config.ChassisCollectorConfig{
			SensorExclude: ptr("([unclosed"),
		})
		require.ErrorContains(t, err, "invalid sensor_exclude pattern")
	})

	// The zero-value config must skip no chassis and no subsystem, so that existing
	// deployments passing chassis_collector: {} keep collecting what they always have.
	t.Run("zero value config disables nothing", func(t *testing.T) {
		var cfg config.ChassisCollectorConfig
		require.False(t, cfg.DisableThermal)
		require.False(t, cfg.DisableThermalSubsystem)
		require.False(t, cfg.DisablePower)
		require.False(t, cfg.DisableNetworkAdapters)
		require.False(t, cfg.DisableSensors)
		require.Empty(t, cfg.ChassisInclude)
		require.Empty(t, cfg.ChassisExclude)
		require.Equal(t, config.DefaultChassisCollector, cfg,
			"the shipped default must be the zero value, so no construction path is special")
	})
}

// ptr returns a pointer to v, for the config fields where an absent key and an empty value
// mean different things.
func ptr[T any](v T) *T { return &v }

func TestParseLeakDetection(t *testing.T) {
	var leakDetection schemas.LeakDetection
	loadTestDataInto(t, "leak_detection.json", &leakDetection)

	metricsCh := make(chan prometheus.Metric, 8)
	parseLeakDetection(metricsCh, "test_chassis", "LeakDetection", &leakDetection)
	metrics := drainMetrics(t, metricsCh)

	health := requireMetric(t, metrics, "redfish_chassis_leak_detection_health")
	require.Equal(t, float64(1), health.value)
	require.Equal(t, "leak_detection", health.labels["resource"])
	require.Equal(t, "test_chassis", health.labels["chassis_id"])
	require.Equal(t, "LeakDetection", health.labels["leak_detection_id"])

	require.Equal(t, float64(1), requireMetric(t, metrics, "redfish_chassis_leak_detection_state").value)
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
