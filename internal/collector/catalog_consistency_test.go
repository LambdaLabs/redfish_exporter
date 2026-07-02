package collector

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/internal/catalog"
)

// descRe extracts fields from prometheus.Desc.String() which has the format:
// Desc{fqName: "...", help: "...", constLabels: {}, variableLabels: {a,b,c}}
var (
	descFQNameRe = regexp.MustCompile(`fqName: "([^"]+)"`)
	descHelpRe   = regexp.MustCompile(`help: "([^"]*)"`)
	descLabelsRe = regexp.MustCompile(`variableLabels: \{([^}]*)\}`)
)

func descFQName(d fmt.Stringer) string {
	m := descFQNameRe.FindStringSubmatch(d.String())
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func descHelp(d fmt.Stringer) string {
	m := descHelpRe.FindStringSubmatch(d.String())
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func descVarLabels(d fmt.Stringer) []string {
	m := descLabelsRe.FindStringSubmatch(d.String())
	if len(m) < 2 || strings.TrimSpace(m[1]) == "" {
		return nil
	}
	parts := strings.Split(m[1], ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

// TestCatalogConsistency cross-checks every prometheus.Desc registered in a
// collector metric map against the corresponding catalog.Entry. It catches:
//   - metrics present in a collector map but absent from the catalog
//   - catalog entries with no collector metric (documented but never emitted)
//   - help string drift between the collector and the catalog
//   - label-name drift between the collector and the catalog
//
// MetricType (Gauge/Counter) is not stored in prometheus.Desc and therefore
// cannot be verified here; that gap is closed by the planned refactor that
// makes the collector import prometheus.Desc directly from the catalog.
func TestCatalogConsistency(t *testing.T) {
	t.Helper()

	// Merge every collector metric map into one name→desc table.
	type collectorEntry struct {
		help   string
		labels []string
	}
	collectorMetrics := map[string]collectorEntry{}

	for _, mm := range []map[string]Metric{
		createChassisMetricMap(),
		createSystemMetricMap(),
		createManagerMetricMap(),
		createGPUMetricMap(),
		createTelemetryMetricMap(),
		createPowershelfMetricsMap(),
	} {
		for _, m := range mm {
			name := descFQName(m.desc)
			collectorMetrics[name] = collectorEntry{
				help:   descHelp(m.desc),
				labels: descVarLabels(m.desc),
			}
		}
	}

	// Build catalog lookup.
	catalogEntries := map[string]catalog.Entry{}
	for _, mod := range catalog.All() {
		for _, e := range mod.Entries {
			catalogEntries[e.Name] = e
		}
	}

	// 1. Every collector metric must appear in the catalog.
	for name, cm := range collectorMetrics {
		e, ok := catalogEntries[name]
		if !ok {
			t.Errorf("collector metric %q has no catalog entry", name)
			continue
		}

		// 2. Help strings must match exactly.
		if cm.help != e.Help {
			t.Errorf("metric %q help mismatch:\n  collector: %q\n  catalog:   %q",
				name, cm.help, e.Help)
		}

		// 3. Label names must match in order (order matters: values are
		// positional in prometheus.MustNewConstMetric calls).
		catLabels := make([]string, 0, len(e.Labels))
		for _, l := range e.Labels {
			catLabels = append(catLabels, l.Name)
		}
		if !slices.Equal(cm.labels, catLabels) {
			t.Errorf("metric %q label mismatch:\n  collector: %v\n  catalog:   %v",
				name, cm.labels, catLabels)
		}
	}

	// 4. Every catalog entry must have a collector metric.
	//    Entries that appear in a metric map but are never emitted still pass
	//    this check; catching dead metric registrations requires integration
	//    tests against a live or mock BMC.
	for name := range catalogEntries {
		if _, ok := collectorMetrics[name]; !ok {
			t.Errorf("catalog entry %q has no collector metric map entry", name)
		}
	}
}
