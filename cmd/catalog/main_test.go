package main

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/internal/catalog"
)

// -- catalog invariants ------------------------------------------------------

func TestCatalogInvariants(t *testing.T) {
	names := map[string]bool{}
	for _, m := range catalog.All() {
		for _, e := range m.Entries {
			if e.Name == "" {
				t.Errorf("%s: entry with empty Name", m.Name)
			}
			if !strings.HasPrefix(e.Name, "redfish_") {
				t.Errorf("%s: entry %q must start with redfish_", m.Name, e.Name)
			}
			if names[e.Name] {
				t.Errorf("duplicate metric name %q", e.Name)
			}
			names[e.Name] = true
			if e.Value.Path == "" {
				t.Errorf("%s: entry %q missing Value.Path", m.Name, e.Name)
			}
			if e.Help == "" {
				t.Errorf("%s: entry %q missing Help", m.Name, e.Name)
			}
			if e.MetricType == catalog.MetricUnset {
				t.Errorf("%s: entry %q missing MetricType", m.Name, e.Name)
			}
			if e.ValueType == catalog.ValueUnset {
				t.Errorf("%s: entry %q missing ValueType", m.Name, e.Name)
			}
			if e.ValueType == catalog.ValueIntEnum && e.Enum == nil {
				t.Errorf("%s: entry %q has ValueType=ValueIntEnum but nil Enum", m.Name, e.Name)
			}
			labelNames := map[string]bool{}
			for _, l := range e.Labels {
				if l.Name == "" {
					t.Errorf("%s: entry %q has label with empty Name", m.Name, e.Name)
				}
				if l.Description == "" {
					t.Errorf("%s: entry %q label %q missing Description", m.Name, e.Name, l.Name)
				}
				if labelNames[l.Name] {
					t.Errorf("%s: entry %q has duplicate label %q", m.Name, e.Name, l.Name)
				}
				labelNames[l.Name] = true
			}
		}
	}
}

// -- writeMarkdown -----------------------------------------------------------

func TestWriteMarkdown(t *testing.T) {
	var buf bytes.Buffer
	if err := writeMarkdown(&buf, catalog.All()); err != nil {
		t.Fatalf("writeMarkdown: %v", err)
	}
	out := buf.String()

	// Header + TOC anchor for a known module.
	if !strings.Contains(out, "# redfish_exporter metric catalog") {
		t.Error("markdown missing top-level heading")
	}
	if !strings.Contains(out, "[chassis](#module-chassis)") {
		t.Error("markdown missing chassis TOC entry")
	}
	if !strings.Contains(out, "## Module: chassis") {
		t.Error("markdown missing chassis module heading")
	}

	// Every catalogued metric should appear as an H3 code span.
	for _, m := range catalog.All() {
		for _, e := range m.Entries {
			needle := "### `" + e.Name + "`"
			if !strings.Contains(out, needle) {
				t.Errorf("markdown missing entry for %q", e.Name)
			}
		}
	}

	// Table header for a metric with labels must be present.
	if !strings.Contains(out, "| Name | Endpoint | Field | Description |") {
		t.Error("markdown missing labels table header")
	}
}

func TestWriteMarkdown_SummaryTable(t *testing.T) {
	mods := []catalog.Module{{
		Name: "alpha",
		Entries: []catalog.Entry{{
			Name:       "redfish_alpha_metric",
			Help:       "test",
			MetricType: catalog.MetricGauge,
			ValueType:  catalog.ValueFloat,
			Value:      catalog.Source{Path: "/redfish/v1/Alpha", Field: "SomeField"},
			Labels: []catalog.Label{
				{Name: "a", Field: "A", Description: "d1"},
				{Name: "b", Field: "B", Description: "d2"},
			},
		}},
	}, {
		Name: "beta",
		Entries: []catalog.Entry{{
			Name:       "redfish_beta_shelf_metric",
			Help:       "shelf",
			MetricType: catalog.MetricGauge,
			ValueType:  catalog.ValueFloat,
			Value:      catalog.Source{Path: "/redfish/v1/Beta", Field: "OtherField"},
			Labels:     nil,
		}},
	}}
	var buf bytes.Buffer
	if err := writeMarkdown(&buf, mods); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	// Single global summary section, not per-module.
	if !strings.Contains(out, "## Summary") {
		t.Error("missing global summary heading")
	}
	if !strings.Contains(out, "All 2 metrics across 2 modules.") {
		t.Error("missing summary preamble with totals")
	}
	if !strings.Contains(out, "| Module | Metric | Metric Type | Value Type | Endpoint | Labels | Value |") {
		t.Error("missing global summary table header (must include Module, Metric Type, Value Type columns)")
	}
	// Rows carry the module name in the first column and the new type columns.
	if !strings.Contains(out, "| alpha | `redfish_alpha_metric` | Gauge | float | `/redfish/v1/Alpha` | a, b | `SomeField` |") {
		t.Errorf("multi-label global row not found in output:\n%s", out)
	}
	if !strings.Contains(out, "| beta | `redfish_beta_shelf_metric` | Gauge | float | `/redfish/v1/Beta` | *(none)* | `OtherField` |") {
		t.Errorf("no-label global row not found in output:\n%s", out)
	}
}

func TestWriteCSV(t *testing.T) {
	mods := []catalog.Module{{
		Name: "alpha",
		Entries: []catalog.Entry{{
			Name:       "redfish_alpha_metric",
			Help:       "alpha description, with a comma",
			MetricType: catalog.MetricGauge,
			ValueType:  catalog.ValueFloat,
			Value:      catalog.Source{Path: "/redfish/v1/Alpha", Field: "SomeField"},
			Labels: []catalog.Label{
				{Name: "a", Field: "A", Description: "d1"},
				{Name: "b", Field: "B", Description: "d2"},
			},
		}},
	}, {
		Name: "beta",
		Entries: []catalog.Entry{{
			Name:       "redfish_beta_shelf_metric",
			Help:       `shelf metric with "quotes"`,
			MetricType: catalog.MetricGauge,
			ValueType:  catalog.ValueIntEnum,
			Enum:       catalog.EnumCommonHealth,
			Value:      catalog.Source{Path: "/redfish/v1/Beta,With,Commas", Field: `Weird "Field"`},
			Labels:     nil,
		}},
	}}
	var buf bytes.Buffer
	if err := writeCSV(&buf, mods); err != nil {
		t.Fatalf("writeCSV: %v", err)
	}

	// Round-trip parse to verify quoting/escaping is valid CSV.
	r := csv.NewReader(&buf)
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("csv.ReadAll: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3 (header + 2 entries)", len(rows))
	}
	header := []string{"module", "metric", "description", "metric_type", "labels", "label_sources", "value_type", "value", "possible_values", "endpoint", "original_endpoint"}
	for i, col := range header {
		if rows[0][i] != col {
			t.Errorf("header col %d = %q, want %q", i, rows[0][i], col)
		}
	}
	// alpha row: float gauge, no enum, no original_endpoint.
	want := []string{"alpha", "redfish_alpha_metric", "alpha description, with a comma", "Gauge", "a; b", "a=.A; b=.B", "float", "SomeField", "", "/redfish/v1/Alpha", ""}
	for i, v := range want {
		if rows[1][i] != v {
			t.Errorf("alpha row col %d = %q, want %q", i, rows[1][i], v)
		}
	}
	// beta row: int_enum gauge, possible_values populated, no original_endpoint.
	want = []string{"beta", "redfish_beta_shelf_metric", `shelf metric with "quotes"`, "Gauge", "", "", "int_enum(CommonHealth)", `Weird "Field"`, "1=OK, 2=Warning, 3=Critical", "/redfish/v1/Beta,With,Commas", ""}
	for i, v := range want {
		if rows[2][i] != v {
			t.Errorf("beta row col %d = %q, want %q", i, rows[2][i], v)
		}
	}
}

func TestWriteMarkdownDir(t *testing.T) {
	dir := t.TempDir()
	if err := writeMarkdownDir(dir, catalog.All()); err != nil {
		t.Fatalf("writeMarkdownDir: %v", err)
	}

	// README.md must exist and contain overview + global summary.
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("reading README.md: %v", err)
	}
	rs := string(readme)
	if !strings.Contains(rs, "# redfish_exporter metric catalog") {
		t.Error("README.md missing top-level heading")
	}
	if !strings.Contains(rs, "[chassis](chassis.md)") {
		t.Error("README.md missing relative link to chassis.md")
	}
	if !strings.Contains(rs, "| Module | Metric | Metric Type | Value Type | Endpoint | Labels | Value | Description |") {
		t.Error("README.md missing global summary table header with Metric Type and Value Type columns")
	}

	// One file per module.
	for _, m := range catalog.All() {
		p := filepath.Join(dir, m.Name+".md")
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("reading %s: %v", p, err)
		}
		s := string(data)
		if !strings.Contains(s, "# Module: "+m.Name) {
			t.Errorf("%s missing H1", p)
		}
		if !strings.Contains(s, "Back to [overview](README.md).") {
			t.Errorf("%s missing back link", p)
		}
		// Every metric belonging to this module must have an H3 in the file.
		for _, e := range m.Entries {
			needle := "### `" + e.Name + "`"
			if !strings.Contains(s, needle) {
				t.Errorf("%s missing H3 entry for %q", p, e.Name)
			}
		}
	}
}

func TestWriteMarkdownDir_CreatesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new", "nested")
	if err := writeMarkdownDir(dir, catalog.All()); err != nil {
		t.Fatalf("writeMarkdownDir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err != nil {
		t.Fatalf("README.md not created in nested dir: %v", err)
	}
}

func TestWriteMarkdown_EscapesPipes(t *testing.T) {
	mods := []catalog.Module{{
		Name: "test",
		Entries: []catalog.Entry{{
			Name:  "redfish_test_metric",
			Help:  "example",
			Value: catalog.Source{Path: "/x", Field: "a|b"},
			Labels: []catalog.Label{{
				Name: "k", Field: "v", Description: "pipe|inside",
			}},
		}},
	}}
	var buf bytes.Buffer
	if err := writeMarkdown(&buf, mods); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `a\|b`) {
		t.Error("expected pipe in Value.Field to be escaped")
	}
	if !strings.Contains(out, `pipe\|inside`) {
		t.Error("expected pipe in label Description to be escaped")
	}
}
