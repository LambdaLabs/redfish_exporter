package collector

import (
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stmcginnis/gofish"
	"github.com/stretchr/testify/require"
)

// psMetric is a flattened emitted metric: fully-qualified name, its labels, and value.
type psMetric struct {
	name   string
	labels map[string]string
	value  float64
}

var fqNameRe = regexp.MustCompile(`fqName: "([^"]+)"`)

// collectPowershelf runs the powershelf collector against client and flattens every
// emitted metric into a []psMetric for assertion.
func collectPowershelf(t *testing.T, client *gofish.APIClient) []psMetric {
	t.Helper()
	c, err := NewPowershelfCollector(t.Name(), client, NewTestLogger(t, slog.LevelWarn), config.PowershelfCollectorConfig{})
	require.NoError(t, err)

	ch := make(chan prometheus.Metric, 512)
	c.Collect(ch)
	close(ch)

	var out []psMetric
	for m := range ch {
		d := &dto.Metric{}
		require.NoError(t, m.Write(d))

		match := fqNameRe.FindStringSubmatch(m.Desc().String())
		if match == nil {
			continue
		}
		labels := map[string]string{}
		for _, lp := range d.GetLabel() {
			labels[lp.GetName()] = lp.GetValue()
		}
		var v float64
		switch {
		case d.GetGauge() != nil:
			v = d.GetGauge().GetValue()
		case d.GetCounter() != nil:
			v = d.GetCounter().GetValue()
		}
		out = append(out, psMetric{name: match[1], labels: labels, value: v})
	}
	return out
}

// countPowershelf returns how many emitted series are redfish_powershelf_* (excludes
// the redfish_collector_scrape_status meta-metric).
func countPowershelf(ms []psMetric) int {
	n := 0
	for _, m := range ms {
		if strings.HasPrefix(m.name, "redfish_powershelf_") {
			n++
		}
	}
	return n
}

// findPS returns the value of the series matching name and all given labels.
func findPS(t *testing.T, ms []psMetric, name string, labels map[string]string) float64 {
	t.Helper()
	for _, m := range ms {
		if m.name != name {
			continue
		}
		ok := true
		for k, v := range labels {
			if m.labels[k] != v {
				ok = false
				break
			}
		}
		if ok {
			return m.value
		}
	}
	t.Fatalf("metric %s%v not found", name, labels)
	return 0
}

func TestPowershelfLiteon(t *testing.T) {
	_, client := setupTestServerClient(t, "testdata/powershelf_liteon")
	ms := collectPowershelf(t, client)

	require.Equal(t, 84, countPowershelf(ms), "expected 84 redfish_powershelf_* series for Lite-On")

	// scrape succeeded
	require.Equal(t, float64(1), findPS(t, ms, "redfish_collector_scrape_status", map[string]string{"collector": "powershelf"}))

	// per-PSU readings (ps1 == p0 in the capture)
	require.Equal(t, 240.25, findPS(t, ms, "redfish_powershelf_input_voltage", map[string]string{"power_supply_id": "ps1"}))
	require.Equal(t, 49.85, findPS(t, ms, "redfish_powershelf_output_voltage", map[string]string{"power_supply_id": "ps1"}))

	// PSU status: health OK -> 1, state "Standby" -> 12 (the non-standard value we added)
	require.Equal(t, float64(1), findPS(t, ms, "redfish_powershelf_psu_health", map[string]string{"power_supply_id": "ps1"}))
	require.Equal(t, float64(12), findPS(t, ms, "redfish_powershelf_psu_state", map[string]string{"power_supply_id": "ps1"}))

	// shelf-level (mapped from chassis_* sensors) — findPS fails the test if absent
	findPS(t, ms, "redfish_powershelf_total_power_in", nil)

	// catch-all: chassis_input_voltage has no curated home, lands under sensor_volts
	findPS(t, ms, "redfish_powershelf_sensor_volts", map[string]string{"sensor_id": "chassis_input_voltage"})
}

func TestPowershelfDelta(t *testing.T) {
	_, client := setupTestServerClient(t, "testdata/powershelf_delta")
	ms := collectPowershelf(t, client)

	require.Equal(t, 154, countPowershelf(ms), "expected 154 redfish_powershelf_* series for Delta")
	require.Equal(t, float64(1), findPS(t, ms, "redfish_collector_scrape_status", map[string]string{"collector": "powershelf"}))

	// per-PSU readings; Delta sensor ids are already 1-indexed (ps1)
	require.Equal(t, 241.5, findPS(t, ms, "redfish_powershelf_input_voltage", map[string]string{"power_supply_id": "ps1"}))
	require.Equal(t, 85.28, findPS(t, ms, "redfish_powershelf_powerfactor", map[string]string{"power_supply_id": "ps1"}))

	// PSU status: Delta reports Enabled -> 1 (vs Lite-On's Standby -> 12)
	require.Equal(t, float64(1), findPS(t, ms, "redfish_powershelf_psu_health", map[string]string{"power_supply_id": "ps1"}))
	require.Equal(t, float64(1), findPS(t, ms, "redfish_powershelf_psu_state", map[string]string{"power_supply_id": "ps1"}))

	// shelf-level: identity-mapped (sensor id == metric name), incl. 3-phase input
	require.Equal(t, float64(1749), findPS(t, ms, "redfish_powershelf_total_power_in", nil))
	require.Equal(t, 241.75, findPS(t, ms, "redfish_powershelf_voltage_in_A_A", nil))
}

// TestPowershelfNoMatch points the collector at a non-powershelf chassis (NVIDIA):
// both adapters decline, so no powershelf metrics are emitted and scrape_status is 0.
func TestPowershelfNoMatch(t *testing.T) {
	_, client := setupTestServerClient(t, "testdata/powershelf_none")
	ms := collectPowershelf(t, client)

	require.Equal(t, 0, countPowershelf(ms), "non-powershelf target should emit no redfish_powershelf_* series")
	require.Equal(t, float64(0), findPS(t, ms, "redfish_collector_scrape_status", map[string]string{"collector": "powershelf"}))
}
