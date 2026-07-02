---
description: Review the redfish_exporter metric catalog and bring it up to date with the current collector implementations. Use this when new metrics, labels, enums, or Redfish paths have been added to collectors but not yet reflected in internal/catalog/.
---

Review the redfish_exporter metric catalog and bring it up to date with the current collector implementations.

## Steps

1. **Read the catalog types and conventions** from `internal/catalog/catalog.go` — understand `Entry`, `Source`, `Label`, `MetricType`, `ValueType`, `EnumDef`, and all package-level `EnumDef` vars.

2. **For each collector↔catalog pair**, read both files and identify gaps:

   | Collector | Catalog file |
   |---|---|
   | `internal/collector/chassis_collector.go` | `internal/catalog/chassis.go` |
   | `internal/collector/system_collector.go` | `internal/catalog/system.go` |
   | `internal/collector/manager_collector.go` | `internal/catalog/manager.go` |
   | `internal/collector/gpu_collector.go` | `internal/catalog/gpu.go` |
   | `internal/collector/telemetry_collector.go` | `internal/catalog/telemetry.go` |
   | `internal/collector/powershelf_collector.go` + `powershelf_liteon.go` + `powershelf_delta.go` | `internal/catalog/powershelf.go` |

3. **Look for these specific gaps**:
   - A metric registered with `prometheus.MustNewConstMetric` or `prometheus.NewDesc` in the collector but absent from the catalog
   - A label emitted by the collector but not declared in the catalog entry's `Labels` slice
   - A catalog entry whose `Value.Path` or `Value.Field` no longer matches what the collector actually reads
   - An enum parse function in `internal/collector/redfish_collector.go` whose integer codes differ from the corresponding `EnumDef` in the catalog
   - A catalog entry with `MetricType` or `ValueType` inconsistent with how the collector emits it (`prometheus.CounterValue` → `MetricCounter`, `prometheus.GaugeValue` → `MetricGauge`)
   - For telemetry: a new MetricReport the collector handles (via `strings.Contains` on the report ID) that has no entries in `internal/catalog/telemetry.go`

4. **Update the catalog files** for every gap found:
   - Use the closure helpers already in each file (e.g. `sensorFloat`, `shelf`, `cnt`, `dur`) rather than verbose struct literals
   - Set `MetricType`, `ValueType`, and `Enum` on every entry — the invariant test in `cmd/catalog/main_test.go` enforces this
   - For `ValueIntEnum` entries, reuse an existing `EnumDef` var or add a new one in `catalog.go`; derive integer codes from the actual parse function in `redfish_collector.go`, not from Help strings
   - For telemetry `OriginalPath`: check `telemetry_collector.go` for the `// Format:` or `// MetricProperty format:` comment above the relevant `parse*MetricProperty` function
   - Match `Source.Field` to the actual JSON field path the collector reads (dotted notation, `[*]` for slices)
   - Set `OriginalPath` only on telemetry entries, empty for all other collectors

5. **Run the tests** after each file change and fix any failures:
   ```
   go test ./cmd/catalog/... ./internal/catalog/...
   ```

6. **Report** a summary of every change: which catalog file, which entries were added or updated, and what was wrong or missing.

## Rules

- Do not add entries for metrics the collector explicitly skips or marks as TODO
- Do not guess enum integer codes — always read the parse function source
- Do not change `MetricType` from `MetricCounter` to `MetricGauge` just because the exporter is stateless; the distinction reflects the semantic type of the BMC-side value
- Do not add `OriginalPath` to non-telemetry entries
