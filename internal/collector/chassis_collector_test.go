package collector

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stmcginnis/gofish"
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

// TestCollectSurvivesOneUnreachableChassis pins the behaviour that a single failing member
// of the chassis collection must not discard the members that did answer.
func TestCollectSurvivesOneUnreachableChassis(t *testing.T) {
	server := newTestRedfishServer(t)
	server.addRoute("/redfish/v1/Chassis", map[string]any{
		"@odata.id":   "/redfish/v1/Chassis",
		"@odata.type": "#ChassisCollection.ChassisCollection",
		"Members": []map[string]string{
			{"@odata.id": "/redfish/v1/Chassis/Chassis_0"},
			{"@odata.id": "/redfish/v1/Chassis/Chassis_1"},
		},
	})
	server.addRoute("/redfish/v1/Chassis/Chassis_0", map[string]any{
		"@odata.id":   "/redfish/v1/Chassis/Chassis_0",
		"@odata.type": "#Chassis.v1_22_0.Chassis",
		"Id":          "Chassis_0",
		"Status":      map[string]any{"Health": "OK", "State": "Enabled"},
	})
	server.addErrorRoute("/redfish/v1/Chassis/Chassis_1", http.StatusInternalServerError)

	client := connectToTestServer(t, server.Server)
	t.Cleanup(func() {
		client.Logout()
		server.Close()
	})

	collector, err := NewChassisCollector(t.Name(), client, NewTestLogger(t, slog.LevelDebug), config.DefaultChassisCollector)
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 128)
	collector.CollectWithContext(context.Background(), ch)
	metrics := drainMetrics(t, ch)

	health := metrics["redfish_chassis_health"]
	require.Len(t, health, 1, "the healthy chassis must still be collected")
	require.Equal(t, "Chassis_0", health[0].labels["chassis_id"])
}

// TestGetChassisSensorsUsesExpand pins that the request actually carries $expand. The BMC
// only inlines the member bodies when asked, so a dropped $expand costs every sensor on
// every platform — the server here answers with link-only members when it is missing, which
// is what a real BMC does.
func TestGetChassisSensorsUsesExpand(t *testing.T) {
	server := newTestRedfishServer(t)
	server.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "chassis_main.json")

	expanded := loadTestData(t, "chassis_sensors_expanded.json")
	var expandQueries []string
	server.mux.HandleFunc("/redfish/v1/Chassis/Chassis_0/Sensors", func(w http.ResponseWriter, r *http.Request) {
		expandQueries = append(expandQueries, r.URL.Query().Get("$expand"))
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("$expand") == "" {
			_, _ = w.Write([]byte(`{"@odata.id":"/redfish/v1/Chassis/Chassis_0/Sensors","Members":[{"@odata.id":"/redfish/v1/Chassis/Chassis_0/Sensors/Chassis_0_FAN_1_FRONT"}]}`))
			return
		}
		_ = json.NewEncoder(w).Encode(expanded)
	})

	var perSensorRequests int
	server.mux.HandleFunc("/redfish/v1/Chassis/Chassis_0/Sensors/", func(w http.ResponseWriter, r *http.Request) {
		perSensorRequests++
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
	sensors := collector.getChassisSensors(context.Background(), links.sensorsPath(chassis[0]), NewTestLogger(t, slog.LevelDebug))
	require.Len(t, sensors, 6, "all sensors should arrive from the single expanded request")
	require.Equal(t, []string{".($levels=1)"}, expandQueries, "the collection must be requested with $expand, exactly once")
	require.Zero(t, perSensorRequests, "no per-sensor requests should be issued when $expand is honoured")
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
// multiply request load against the BMCs least able to absorb it, so the bulk telemetry is
// skipped with a warning instead.
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
			_, _ = w.Write([]byte(`{"@odata.id":"/redfish/v1/Chassis/Chassis_0/Sensors/` + sensorID + `","Id":"` + sensorID + `","Name":"` + sensorID + `","ReadingType":"Voltage","Reading":1.7}`))
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

	links := chassisAdvertisedLinks(chassis[0])
	sensors := collector.getChassisSensors(context.Background(), links.sensorsPath(chassis[0]), NewTestLogger(t, slog.LevelDebug))

	require.Empty(t, sensors, "an unexpanded collection yields no bulk sensors")
	require.Empty(t, perSensorRequests, "no sensor may be fetched individually")
}

// TestSensorsAreNotCollectedAlongsideThermal pins the invariant the whole fallback rests on:
// a chassis that implements Thermal must not also be read through Sensors. Both express the
// same temperature, so collecting both publishes redfish_chassis_temperature_celsius twice
// under one label set, which fails the entire scrape at registration rather than merely
// thinning it — on the legacy platforms, not the new ones this PR is for.
func TestSensorsAreNotCollectedAlongsideThermal(t *testing.T) {
	server := newTestRedfishServer(t)
	server.addRoute("/redfish/v1/Chassis", map[string]any{
		"@odata.id":   "/redfish/v1/Chassis",
		"@odata.type": "#ChassisCollection.ChassisCollection",
		"Members":     []map[string]string{{"@odata.id": "/redfish/v1/Chassis/Chassis_0"}},
	})
	// A legacy chassis: Thermal as well as a Sensors collection. Both carry the same
	// reading, which is exactly the collision to avoid.
	server.addRoute("/redfish/v1/Chassis/Chassis_0", map[string]any{
		"@odata.id":   "/redfish/v1/Chassis/Chassis_0",
		"@odata.type": "#Chassis.v1_22_0.Chassis",
		"Id":          "Chassis_0",
		"Status":      map[string]any{"Health": "OK", "State": "Enabled"},
		"Thermal":     map[string]string{"@odata.id": "/redfish/v1/Chassis/Chassis_0/Thermal"},
		"Sensors":     map[string]string{"@odata.id": "/redfish/v1/Chassis/Chassis_0/Sensors"},
	})
	server.addRoute("/redfish/v1/Chassis/Chassis_0/Thermal", map[string]any{
		"@odata.id": "/redfish/v1/Chassis/Chassis_0/Thermal",
		"Id":        "Thermal",
		"Temperatures": []map[string]any{{
			"MemberId":       "Chassis_0_Front_IO_Temp_0",
			"Name":           "Chassis 0 Front IO Temp 0",
			"ReadingCelsius": 27.5,
			"Status":         map[string]any{"Health": "OK", "State": "Enabled"},
		}},
	})

	var sensorRequests int
	server.mux.HandleFunc("/redfish/v1/Chassis/Chassis_0/Sensors", func(w http.ResponseWriter, r *http.Request) {
		sensorRequests++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(loadTestData(t, "chassis_sensors_expanded.json"))
	})

	client := connectToTestServer(t, server.Server)
	t.Cleanup(func() {
		client.Logout()
		server.Close()
	})

	collector, err := NewChassisCollector(t.Name(), client, NewTestLogger(t, slog.LevelDebug), config.DefaultChassisCollector)
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 256)
	collector.CollectWithContext(context.Background(), ch)
	metrics := drainMetrics(t, ch)

	require.Zero(t, sensorRequests, "a chassis advertising Thermal must not be asked for Sensors")
	celsius := requireMetric(t, metrics, "redfish_chassis_temperature_celsius")
	require.InDelta(t, 27.5, celsius.value, 1e-9, "the reading must come from Thermal, once")
	require.NotContains(t, metrics, "redfish_chassis_sensor_reading", "no Sensors series may appear")
}

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
		// A reference object may spell its URI "href", which gofish accepts. Reading only
		// "@odata.id" would have this conclude the chassis has no Thermal while gofish
		// collects one, and the chassis would publish its temperatures from both paths at
		// once — a duplicate series, which fails the whole scrape at registration.
		"href spelling counts as a link": {
			raw:        `{"Id":"1","Thermal":{"href":"/redfish/v1/Chassis/1/Thermal"}}`,
			wantLegacy: true,
		},
		"href spelling is followed for Sensors too": {
			raw:         `{"Id":"Chassis_0","Sensors":{"href":"/redfish/v1/Chassis/Chassis_0/Sensors"}}`,
			wantLegacy:  false,
			wantSensors: "/redfish/v1/Chassis/Chassis_0/Sensors",
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
	// Sensors fallback is suppressed rather than issuing a request per chassis.
	opaque := chassisAdvertisedLinks(&schemas.Chassis{})
	require.True(t, opaque.legacyThermalOrPower())
	require.Equal(t, "/Sensors", opaque.sensorsPath(&schemas.Chassis{}))
}

// TestLeakDetectorSensorIsNeverAChassisVoltage pins a safety property that has to hold no
// matter how leak detection itself evolves: a detector's companion sensor is a Voltage
// reading like any other, so anything that folds it into the power_voltage family reports a
// leak signal as a rail measurement.
//
// The assertion is deliberately negative rather than "this sensor produces nothing", so it
// keeps its meaning once the detectors gain a metric family of their own.
func TestLeakDetectorSensorIsNeverAChassisVoltage(t *testing.T) {
	server := newTestRedfishServer(t)
	server.addRouteFromFixture("/redfish/v1/Chassis", "chassis_collection.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0", "chassis_main.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/Sensors", "chassis_sensors_expanded.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem", "thermal_subsystem.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection", "leak_detection.json")
	server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors", "leak_detectors_collection.json")
	for _, id := range []string{
		"Chassis_0_LeakDetector_0_ColdPlate",
		"Chassis_0_LeakDetector_0_Manifold",
		"Chassis_0_LeakDetector_1_ColdPlate",
		"Chassis_0_LeakDetector_1_Manifold",
	} {
		server.addRouteFromFixture("/redfish/v1/Chassis/Chassis_0/ThermalSubsystem/LeakDetection/LeakDetectors/"+id, "leak_detector_ok.json")
	}

	client := connectToTestServer(t, server.Server)
	t.Cleanup(func() {
		client.Logout()
		server.Close()
	})

	collector, err := NewChassisCollector(t.Name(), client, NewTestLogger(t, slog.LevelDebug), config.DefaultChassisCollector)
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 512)
	collector.CollectWithContext(context.Background(), ch)
	metrics := drainMetrics(t, ch)

	// The rest of the collection still arrives.
	require.Contains(t, metrics, "redfish_chassis_temperature_celsius")
	require.Contains(t, metrics, "redfish_chassis_sensor_watts")

	// The detector's voltage does not, under any series name — power_voltage included.
	for name, samples := range metrics {
		for _, sample := range samples {
			for _, label := range []string{"sensor_id", "power_voltage_id"} {
				require.NotContains(t, sample.labels[label], "LeakDetector",
					"leak detector %q leaked into %s", sample.labels[label], name)
			}
		}
	}
}

func TestParseChassisSensor(t *testing.T) {
	var collection sensorCollection
	loadTestDataInto(t, "chassis_sensors_expanded.json", &collection)
	require.Len(t, collection.Members, 6)

	byID := map[string]*schemas.Sensor{}
	for _, sensor := range collection.Members {
		byID[sensor.ID] = sensor
	}

	t.Run("temperature folds into the existing chassis family", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 32)
		parseChassisSensor(ch, "Chassis_0", byID["Chassis_0_Front_IO_Temp_0"])
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
		parseChassisSensor(ch, "Chassis_0", byID["Chassis_0_FAN_1_FRONT"])
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

	// parseChassisFan has always handled two cases beyond a plain RPM range, and the
	// Sensors path has to reach the same numbers or a platform moving from Thermal to
	// Sensors would shift the series under the existing alerts.
	t.Run("rotational percentage matches the thermal path in its awkward cases", func(t *testing.T) {
		reading := 50.0
		max := 20000.0

		t.Run("a reading already in percent is used as-is", func(t *testing.T) {
			ch := make(chan prometheus.Metric, 32)
			parseChassisSensor(ch, "Chassis_0", &schemas.Sensor{
				Entity:       schemas.Entity{ID: "FAN_PCT", Name: "FAN_PCT"},
				ReadingType:  schemas.RotationalReadingType,
				ReadingUnits: "%",
				Reading:      &reading,
			})
			metrics := drainMetrics(t, ch)
			require.InDelta(t, 50.0, requireMetric(t, metrics, "redfish_chassis_fan_rpm_percentage").value, 1e-9)
		})

		// Some vendors report null Min/Max but do supply the upper thresholds, so those
		// stand in for the max rather than leaving the percentage stuck at zero.
		t.Run("an absent max falls back to the upper thresholds", func(t *testing.T) {
			ch := make(chan prometheus.Metric, 32)
			parseChassisSensor(ch, "Chassis_0", &schemas.Sensor{
				Entity:      schemas.Entity{ID: "FAN_NORANGE", Name: "FAN_NORANGE"},
				ReadingType: schemas.RotationalReadingType,
				Reading:     &reading,
				Thresholds:  schemas.Thresholds{UpperCritical: schemas.Threshold{Reading: &max}},
			})
			metrics := drainMetrics(t, ch)
			require.InDelta(t, 0.25, requireMetric(t, metrics, "redfish_chassis_fan_rpm_percentage").value, 1e-9)
		})
	})

	// Fan PWM duty cycle and CPU core utilisation are both ReadingType "Percent" and
	// neither carries a distinguishing PhysicalContext, so Percent must not be assumed
	// to be a fan speed.
	t.Run("percent is not assumed to be a fan speed", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 32)
		parseChassisSensor(ch, "Chassis_0", byID["Chassis_0_FAN_1_PWM"])
		metrics := drainMetrics(t, ch)

		require.NotContains(t, metrics, "redfish_chassis_fan_rpm_percentage")
		percent := requireMetric(t, metrics, "redfish_chassis_sensor_percent")
		require.InDelta(t, 29.803921, percent.value, 1e-5)
		require.Equal(t, "%", percent.labels["sensor_units"])
	})

	t.Run("power and unrecognised reading types reach a catch-all", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 32)
		parseChassisSensor(ch, "Chassis_0", byID["Chassis_0_TotalHSC_Power_0"])
		metrics := drainMetrics(t, ch)
		require.InDelta(t, 1171.349692, requireMetric(t, metrics, "redfish_chassis_sensor_watts").value, 1e-6)

		// Altitude has no dedicated metric; it must still be emitted rather than dropped.
		ch = make(chan prometheus.Metric, 32)
		parseChassisSensor(ch, "Chassis_0", byID["Chassis_0_Altitude_0"])
		metrics = drainMetrics(t, ch)
		reading := requireMetric(t, metrics, "redfish_chassis_sensor_reading")
		require.InDelta(t, 132.0, reading.value, 1e-9)
		require.Equal(t, "m", reading.labels["sensor_units"])
	})
}

// TestFanPercentageSharedByBothPaths drives the same firmware numbers through the Thermal
// and the Sensors path and requires one answer from both, series identity included. An alert
// on redfish_chassis_fan_rpm_percentage has to mean the same thing on a platform reporting
// fans through Sensors as on one reporting them through Thermal, which holds only for as long
// as the two go on sharing fanPercentage and fanUnitLabel.
//
// The two schemas spell the units differently — Thermal's enum has "RPM" and "Percent" where
// a Sensor has "RPM" and "%" — so each case states both spellings of one unit.
func TestFanPercentageSharedByBothPaths(t *testing.T) {
	tT := map[string]struct {
		reading       float64
		min           float64
		max           float64
		upperCritical float64
		upperFatal    float64
		thermalUnits  schemas.ReadingUnits
		sensorUnits   string
		want          float64
		wantUnitLabel string
	}{
		"a plain reading range": {reading: 8623, min: 2000, max: 30000, want: 35.41},
		// Some vendors (e.g. PowerEdge C6420) report no range at all but do supply the
		// upper thresholds, which then stand in for the max.
		"no range, upper critical only":   {reading: 50, upperCritical: 20000, want: 0.25},
		"no range, both upper thresholds": {reading: 50, upperCritical: 20000, upperFatal: 24000, want: 0.2083333},
		// Nothing to divide by: the percentage is zero rather than an infinity.
		"no range and no thresholds": {reading: 8623, want: 0},
		// A fan already reporting a duty cycle needs no derivation, and the two spellings
		// of that unit must not split the series in two. The reading is whole because the
		// Thermal schema carries it as an int.
		"a reading already in percent": {
			reading: 30, max: 30000, thermalUnits: schemas.PercentReadingUnits, sensorUnits: "%",
			want: 30, wantUnitLabel: "percent",
		},
	}

	for name, test := range tT {
		t.Run(name, func(t *testing.T) {
			intPtr := func(v float64) *int { i := int(v); return &i }
			floatPtr := func(v float64) *float64 { return &v }

			thermalUnits, sensorUnits := test.thermalUnits, test.sensorUnits
			if thermalUnits == "" {
				thermalUnits, sensorUnits = schemas.RPMReadingUnits, "RPM"
			}
			wantUnitLabel := test.wantUnitLabel
			if wantUnitLabel == "" {
				wantUnitLabel = "rpm"
			}

			thermalCh := make(chan prometheus.Metric, 32)
			parseChassisFan(thermalCh, "Chassis_0", schemas.ThermalFan{
				Entity:                 schemas.Entity{ID: "FAN_0", Name: "FAN_0"},
				MemberID:               "FAN_0",
				ReadingUnits:           thermalUnits,
				Reading:                intPtr(test.reading),
				MinReadingRange:        intPtr(test.min),
				MaxReadingRange:        intPtr(test.max),
				UpperThresholdCritical: intPtr(test.upperCritical),
				UpperThresholdFatal:    intPtr(test.upperFatal),
			})
			thermal := requireMetric(t, drainMetrics(t, thermalCh), "redfish_chassis_fan_rpm_percentage")

			sensorCh := make(chan prometheus.Metric, 32)
			parseChassisSensor(sensorCh, "Chassis_0", &schemas.Sensor{
				Entity:          schemas.Entity{ID: "FAN_0", Name: "FAN_0"},
				ReadingType:     schemas.RotationalReadingType,
				ReadingUnits:    sensorUnits,
				Reading:         floatPtr(test.reading),
				ReadingRangeMin: floatPtr(test.min),
				ReadingRangeMax: floatPtr(test.max),
				Thresholds: schemas.Thresholds{
					UpperCritical: schemas.Threshold{Reading: floatPtr(test.upperCritical)},
					UpperFatal:    schemas.Threshold{Reading: floatPtr(test.upperFatal)},
				},
			})
			sensor := requireMetric(t, drainMetrics(t, sensorCh), "redfish_chassis_fan_rpm_percentage")

			require.Equal(t, wantUnitLabel, thermal.labels["fan_unit"])
			require.Equal(t, thermal.labels["fan_unit"], sensor.labels["fan_unit"],
				"one fan must not land under two fan_unit spellings")
			require.InDelta(t, test.want, thermal.value, 1e-6)
			require.InDelta(t, thermal.value, sensor.value, 1e-9)
		})
	}
}

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

	parseLeakDetector(metricsCh, "test_chassis", detectors[0])
	close(metricsCh)

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
