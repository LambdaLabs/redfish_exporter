// Package catalog is a declarative description of every Prometheus metric
// emitted by the redfish_exporter, together with the Redfish endpoints and
// fields those metrics (and their labels) are sourced from.
//
// It exists so downstream tooling (see cmd/catalog) can:
//   - render an inventory of what the exporter emits
//   - walk a live Redfish target and diff it against the catalog to surface
//     endpoints or fields that the exporter is not (yet) scraping
//
// The catalog is authored by hand alongside each collector's implementation.
// It is intentionally decoupled from the runtime collector package so tooling
// does not drag in gofish or the HTTP surface.
package catalog

// MetricType is the Prometheus metric type an Entry is registered as.
type MetricType uint8

const (
	MetricUnset   MetricType = iota // must not appear in catalog entries
	MetricGauge                     // time-varying measurement
	MetricCounter                   // monotonically increasing total
	MetricInfo                      // constant-1 gauge carrying identifying data in labels
)

func (t MetricType) String() string {
	switch t {
	case MetricGauge:
		return "Gauge"
	case MetricCounter:
		return "Counter"
	case MetricInfo:
		return "Info"
	default:
		return "Unset"
	}
}

// ValueType describes the semantic type of a metric's numeric value.
type ValueType uint8

const (
	ValueUnset    ValueType = iota // must not appear in catalog entries
	ValueFloat                     // continuous float (watts, celsius, Gbps, %, …)
	ValueInt                       // integer count or size
	ValueBool                      // binary 0=false / 1=true flag
	ValueDuration                  // float seconds parsed from an ISO 8601 duration string
	ValueIntEnum                   // integer encoding of a named Redfish string enum; see Enum field
	ValueConstant                  // always literal 1 (info metrics; identifying data in labels)
)

func (t ValueType) String() string {
	switch t {
	case ValueFloat:
		return "float"
	case ValueInt:
		return "int"
	case ValueBool:
		return "bool"
	case ValueDuration:
		return "duration"
	case ValueIntEnum:
		return "int_enum"
	case ValueConstant:
		return "constant"
	default:
		return "unset"
	}
}

// EnumValue is one member of a Redfish string enum as mapped to a Prometheus float64.
type EnumValue struct {
	Code  int
	Label string
}

// EnumDef names a Redfish string enum and lists every code/label pair.
// Populated only on entries where ValueType == ValueIntEnum.
type EnumDef struct {
	Name   string
	Values []EnumValue
}

// Package-level EnumDef vars referenced by catalog entries whose ValueType is
// ValueIntEnum. The integer codes match the parse functions in
// internal/collector/redfish_collector.go.
var (
	EnumCommonHealth = &EnumDef{
		Name: "CommonHealth",
		Values: []EnumValue{
			{1, "OK"}, {2, "Warning"}, {3, "Critical"},
		},
	}
	EnumCommonState = &EnumDef{
		Name: "CommonState",
		Values: []EnumValue{
			{1, "Enabled"}, {2, "Disabled"}, {3, "StandbyOffline"},
			{4, "StandbySpare"}, {5, "InTest"}, {6, "Starting"},
			{7, "Absent"}, {8, "UnavailableOffline"}, {9, "Deferring"},
			{10, "Quiesced"}, {11, "Updating"}, {12, "Standby"},
		},
	}
	EnumPowerState = &EnumDef{
		Name: "PowerState",
		Values: []EnumValue{
			{1, "On"}, {2, "Off"}, {3, "PoweringOn"}, {4, "PoweringOff"},
		},
	}
	// NVLink port link status — parseNVLinkPortLinkStatus in redfish_collector.go.
	EnumNVLinkPortLinkStatus = &EnumDef{
		Name: "NVLinkPortLinkStatus",
		Values: []EnumValue{
			{1, "LinkUp"}, {2, "Starting"}, {3, "Training"}, {4, "LinkDown"}, {5, "NoLink"},
		},
	}
	// Ethernet interface link status — parseLinkStatus in redfish_collector.go.
	EnumEthernetLinkStatus = &EnumDef{
		Name: "EthernetLinkStatus",
		Values: []EnumValue{
			{1, "LinkUp"}, {2, "NoLink"}, {3, "LinkDown"},
		},
	}
	// Network port link state — parsePortLinkStatus in redfish_collector.go.
	EnumPortLinkState = &EnumDef{
		Name: "PortLinkState",
		Values: []EnumValue{
			{0, "Down"}, {1, "Up"},
		},
	}
	EnumIntrusionSensor = &EnumDef{
		Name: "IntrusionSensor",
		Values: []EnumValue{
			{1, "Normal"}, {2, "TamperingDetected"}, {3, "HardwareIntrusion"},
		},
	}
	// LastResetType — switch in telemetry_collector.go collectResetMetrics.
	EnumLastResetType = &EnumDef{
		Name: "LastResetType",
		Values: []EnumValue{
			{1, "Conventional"}, {2, "Fundamental"}, {3, "IRoT"}, {4, "PF_FLR"},
		},
	}
)

// Entry describes a single Prometheus metric emitted by the exporter.
type Entry struct {
	Name       string
	Help       string
	MetricType MetricType
	ValueType  ValueType
	Enum       *EnumDef // non-nil only when ValueType == ValueIntEnum
	Value      Source
	Labels     []Label
}

// Source identifies a location within the Redfish API. Path may contain
// {placeholder} segments that correspond to Label.Name values on the same
// Entry (typical: {chassis_id}, {system_id}). Field uses a dotted JSON
// pointer-ish notation with [*] for list traversal.
//
// OriginalPath is the Redfish resource path the metric is ultimately sourced
// from, populated only for TelemetryService entries where Path points at a
// MetricReport and OriginalPath carries the underlying resource URL template
// (e.g. /redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics).
// Empty for all non-telemetry collectors.
type Source struct {
	Path         string
	Field        string
	OriginalPath string
}

// Label describes a single Prometheus label on a metric and where its value
// originates in the Redfish API. Path defaults to the enclosing Entry's
// Value.Path when empty — set it explicitly only when the label's value is
// read from a different resource than the metric value.
//
// Constant, when non-empty, means the label always carries that literal string
// value regardless of Redfish data (Field must be ""). Use labelResource for
// the common "resource" label pattern.
type Label struct {
	Name        string
	Path        string
	Field       string
	Constant    string // non-empty: label value is this literal string, not from Redfish
	Description string
}

// EffectivePath returns the endpoint the label's value is sourced from. If
// the label did not declare its own Path, the enclosing Entry's Value.Path
// applies.
func (l Label) EffectivePath(entry Entry) string {
	if l.Path != "" {
		return l.Path
	}
	return entry.Value.Path
}

// Module groups the entries emitted by a single collector module. Name
// matches the collector's subsystem string (e.g. "chassis", "system").
// Description is optional free-text shown in markdown output; useful for
// modules like "json" whose metrics are defined at runtime rather than here.
type Module struct {
	Name        string
	Description string
	Entries     []Entry
}

// All returns every Module known to the catalog, in a stable order suitable
// for deterministic output.
func All() []Module {
	return []Module{
		Chassis(),
		System(),
		Manager(),
		GPU(),
		Telemetry(),
		Powershelf(),
		JSON(),
	}
}
