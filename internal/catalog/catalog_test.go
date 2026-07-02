package catalog_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/internal/catalog"
)

var (
	validMetricName = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)
	validLabelName  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// TestCatalogInternalConsistency validates the structural invariants of every
// catalog entry independent of the collector. It catches:
//   - duplicate metric names across modules
//   - missing or unset MetricType / ValueType
//   - MetricInfo / ValueConstant mismatch (must be paired; neither alone is valid)
//   - ValueIntEnum entries missing their EnumDef, or non-enum entries carrying one
//   - duplicate codes within an EnumDef
//   - empty Value.Path or Value.Field for concrete (non-constant) entries
//   - invalid Prometheus metric or label names
func TestCatalogInternalConsistency(t *testing.T) {
	t.Helper()

	seen := map[string]string{} // metric name → first-seen module

	for _, mod := range catalog.All() {
		for _, e := range mod.Entries {
			e, mod := e, mod
			t.Run(fmt.Sprintf("%s/%s", mod.Name, e.Name), func(t *testing.T) {
				// 1. Name must be non-empty and a valid Prometheus metric name.
				if e.Name == "" {
					t.Fatal("Name is empty")
				}
				if !validMetricName.MatchString(e.Name) {
					t.Errorf("Name %q is not a valid Prometheus metric name", e.Name)
				}

				// 2. No duplicate names across modules.
				if prev, ok := seen[e.Name]; ok {
					t.Errorf("Name %q already defined in module %q", e.Name, prev)
				} else {
					seen[e.Name] = mod.Name
				}

				// 3. Help must be non-empty.
				if strings.TrimSpace(e.Help) == "" {
					t.Error("Help is empty")
				}

				// 4. MetricType must be set.
				if e.MetricType == catalog.MetricUnset {
					t.Error("MetricType is Unset")
				}

				// 5. ValueType must be set.
				if e.ValueType == catalog.ValueUnset {
					t.Error("ValueType is Unset")
				}

				// 6. MetricInfo ↔ ValueConstant must always appear together.
				if e.MetricType == catalog.MetricInfo && e.ValueType != catalog.ValueConstant {
					t.Errorf("MetricInfo entry has ValueType %s, want ValueConstant", e.ValueType)
				}
				if e.MetricType != catalog.MetricInfo && e.ValueType == catalog.ValueConstant {
					t.Errorf("%s entry has ValueConstant; only MetricInfo may use ValueConstant", e.MetricType)
				}

				// 7. ValueIntEnum requires a non-empty EnumDef with unique codes.
				if e.ValueType == catalog.ValueIntEnum {
					if e.Enum == nil {
						t.Error("ValueIntEnum entry has nil Enum")
					} else {
						if len(e.Enum.Values) == 0 {
							t.Errorf("Enum %q has no values", e.Enum.Name)
						}
						codes := map[int]string{}
						for _, ev := range e.Enum.Values {
							if prev, dup := codes[ev.Code]; dup {
								t.Errorf("Enum %q: duplicate code %d (labels %q and %q)",
									e.Enum.Name, ev.Code, prev, ev.Label)
							} else {
								codes[ev.Code] = ev.Label
							}
						}
					}
				}

				// 8. Non-ValueIntEnum entries must not carry an EnumDef.
				if e.ValueType != catalog.ValueIntEnum && e.Enum != nil {
					t.Errorf("ValueType %s entry has Enum %q; only ValueIntEnum entries may have one",
						e.ValueType, e.Enum.Name)
				}

				// 9–10. Source path and field — skip the JSON module (runtime-defined, no fixed entries).
				if mod.Name != "json" {
					if e.Value.Path == "" {
						t.Error("Value.Path is empty")
					}
					// ValueConstant (info) metrics carry descriptive text in Field, not a JSON path.
					// All other value types must point at a real field.
					if e.ValueType != catalog.ValueConstant && e.Value.Field == "" {
						t.Error("Value.Field is empty for non-constant metric")
					}
				}

				// 11. Label names must be valid and not use the reserved __ prefix.
				for _, l := range e.Labels {
					if l.Name == "" {
						t.Error("Label has empty Name")
						continue
					}
					if !validLabelName.MatchString(l.Name) {
						t.Errorf("label name %q is not a valid Prometheus label name", l.Name)
					}
					if strings.HasPrefix(l.Name, "__") {
						t.Errorf("label name %q uses reserved __ prefix", l.Name)
					}
				}
			})
		}
	}
}
