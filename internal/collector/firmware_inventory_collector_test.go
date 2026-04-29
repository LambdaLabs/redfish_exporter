package collector

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

type collectedFirmwareMetric struct {
	metricName string
	labels     map[string]string
	value      float64
}

func collectFirmwareInventoryMetrics(t *testing.T) []collectedFirmwareMetric {
	t.Helper()
	return collectFirmwareInventoryMetricsFromFixture(t, "testdata/firmware_inventory_happypath")
}

func collectFirmwareInventoryMetricsFromFixture(t *testing.T, fixture string) []collectedFirmwareMetric {
	t.Helper()
	_, client := setupTestServerClient(t, fixture)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector, err := NewFirmwareInventoryCollector(t.Name(), client, logger, config.DefaultFirmwareInventoryCollector)
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 100)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	var collected []collectedFirmwareMetric
	for metric := range ch {
		descString := metric.Desc().String()
		// Skip the meta `redfish_collector_scrape_status` gauge — only want firmware_inventory_* metrics.
		if !strings.Contains(descString, "redfish_firmware_inventory_") {
			continue
		}
		d := &dto.Metric{}
		require.NoError(t, metric.Write(d))
		labels := make(map[string]string, len(d.Label))
		for _, l := range d.Label {
			labels[l.GetName()] = l.GetValue()
		}
		// Pull metric name from FQName in the desc string: 'fqName: "redfish_firmware_inventory_state"'.
		var name string
		if i := strings.Index(descString, `fqName: "`); i >= 0 {
			rest := descString[i+len(`fqName: "`):]
			if j := strings.Index(rest, `"`); j >= 0 {
				name = rest[:j]
			}
		}
		collected = append(collected, collectedFirmwareMetric{
			metricName: name,
			labels:     labels,
			value:      d.GetGauge().GetValue(),
		})
	}
	return collected
}

func TestFirmwareInventoryCollectorEmitsExpectedMembers(t *testing.T) {
	collected := collectFirmwareInventoryMetrics(t)

	componentIDs := map[string]struct{}{}
	for _, m := range collected {
		componentIDs[m.labels["component_id"]] = struct{}{}
	}
	for _, expected := range []string{"HGX_FW_BMC_0", "BMC", "BIOS", "GPU1", "HGX_FW_ERoT_BMC_0"} {
		require.Contains(t, componentIDs, expected, "expected firmware inventory member %s to be present", expected)
	}
}

func TestFirmwareInventoryCollectorExcludesBackupGoldenCapsule(t *testing.T) {
	collected := collectFirmwareInventoryMetrics(t)

	for _, m := range collected {
		id := m.labels["component_id"]
		require.False(t, strings.HasPrefix(id, "Backup_"), "Backup_-prefixed firmware should be excluded; got %s", id)
		require.False(t, strings.HasPrefix(id, "Golden_"), "Golden_-prefixed firmware should be excluded; got %s", id)
		require.False(t, strings.HasPrefix(id, "Capsule_"), "Capsule_-prefixed firmware should be excluded; got %s", id)
	}
}

func TestFirmwareInventoryCollectorInfoMetricLabels(t *testing.T) {
	collected := collectFirmwareInventoryMetrics(t)

	expectedInfo := map[string]struct{ name, version, manufacturer string }{
		"HGX_FW_BMC_0":      {"Software Inventory", "HGX-22.10-1-rc67", "NVIDIA"},
		"BMC":               {"BMC", "01.04.11", "Supermicro"},
		"BIOS":              {"BIOS", "BIOS Date: 10/28/2025 Ver 2.7a", "Supermicro"},
		"GPU1":              {"GPU1 System Slot1", "96.00.A5.00.01", "Supermicro"},
		"HGX_FW_ERoT_BMC_0": {"Software Inventory", "00.02.0182.0000_n00", "NVIDIA"},
	}

	seen := map[string]bool{}
	for _, m := range collected {
		if m.metricName != "redfish_firmware_inventory_info" {
			continue
		}
		want, ok := expectedInfo[m.labels["component_id"]]
		if !ok {
			continue
		}
		require.Equal(t, float64(1), m.value, "info metric value must always be 1")
		require.Equal(t, want.name, m.labels["name"], "name label for %s", m.labels["component_id"])
		require.Equal(t, want.version, m.labels["version"], "version label for %s", m.labels["component_id"])
		require.Equal(t, want.manufacturer, m.labels["manufacturer"], "manufacturer label for %s", m.labels["component_id"])
		seen[m.labels["component_id"]] = true
	}
	require.Len(t, seen, len(expectedInfo), "did not see _info metrics for all expected members")
}

func TestFirmwareInventoryCollectorVersionOnlyOnInfoMetric(t *testing.T) {
	collected := collectFirmwareInventoryMetrics(t)

	// Drift detection requires version to live exclusively on _info; if it leaks onto
	// _state/_health/_write_protected, every firmware update spawns new time series
	// for those metrics and breaks historical health queries.
	for _, m := range collected {
		switch m.metricName {
		case "redfish_firmware_inventory_state",
			"redfish_firmware_inventory_health",
			"redfish_firmware_inventory_write_protected":
			require.NotContains(t, m.labels, "version",
				"version label must not appear on %s (component_id=%s)", m.metricName, m.labels["component_id"])
			require.NotContains(t, m.labels, "manufacturer",
				"manufacturer label must not appear on %s (component_id=%s)", m.metricName, m.labels["component_id"])
		}
	}
}

func TestFirmwareInventoryCollectorStateAndHealthValues(t *testing.T) {
	collected := collectFirmwareInventoryMetrics(t)

	// All fixture members report State=Enabled (1) and Health=OK (1).
	for _, m := range collected {
		switch m.metricName {
		case "redfish_firmware_inventory_state":
			require.Equal(t, float64(1), m.value, "expected Enabled (1) for %s", m.labels["component_id"])
		case "redfish_firmware_inventory_health":
			require.Equal(t, float64(1), m.value, "expected OK (1) for %s", m.labels["component_id"])
		case "redfish_firmware_inventory_write_protected":
			require.Equal(t, float64(0), m.value, "expected WriteProtected=false (0) for %s", m.labels["component_id"])
		}
	}
}

func TestFirmwareInventoryCollectorMapsWarningCriticalAndWriteProtected(t *testing.T) {
	collected := collectFirmwareInventoryMetricsFromFixture(t, "testdata/firmware_inventory_unhealthy")

	want := map[string]struct {
		state, health, writeProtected float64
	}{
		"WARNING_FW":   {state: 1, health: 2, writeProtected: 0},
		"CRITICAL_FW":  {state: 2, health: 3, writeProtected: 0},
		"PROTECTED_FW": {state: 1, health: 1, writeProtected: 1},
	}

	got := map[string]struct {
		state, health, writeProtected float64
	}{}

	for _, m := range collected {
		entry := got[m.labels["component_id"]]
		switch m.metricName {
		case "redfish_firmware_inventory_state":
			entry.state = m.value
		case "redfish_firmware_inventory_health":
			entry.health = m.value
		case "redfish_firmware_inventory_write_protected":
			entry.writeProtected = m.value
		}
		got[m.labels["component_id"]] = entry
	}

	for id, expected := range want {
		require.Equal(t, expected, got[id], "metric values for %s", id)
	}
}

func TestFirmwareInventoryCollectorEmptyInventoryEmitsNoMemberMetrics(t *testing.T) {
	collected := collectFirmwareInventoryMetricsFromFixture(t, "testdata/firmware_inventory_empty")
	require.Empty(t, collected, "no firmware_inventory_* metrics should be emitted when inventory is empty")
}

func TestFirmwareInventoryCollectorAllExcludedEmitsNoMemberMetrics(t *testing.T) {
	collected := collectFirmwareInventoryMetricsFromFixture(t, "testdata/firmware_inventory_all_excluded")
	require.Empty(t, collected, "no firmware_inventory_* metrics should be emitted when every member matches an excluded prefix")
}
