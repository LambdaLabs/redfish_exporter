package catalog

// JSON returns the catalog for the JSON collector.
//
// The JSON collector is fundamentally dynamic: metric names, labels, and
// Redfish paths are all supplied at runtime via user JQ filters, so it has
// no fixed catalog. The Entries slice is intentionally empty; the diff
// subcommand should skip endpoints that the running configuration covers
// via json_collector by consulting the exporter config, not this package.
func JSON() Module {
	return Module{
		Name: "json",
		Description: "The JSON collector is configuration-driven: metric names, labels, Redfish " +
			"paths, and JQ extraction filters are all supplied at runtime via the exporter " +
			"configuration file. There are no predefined metrics — what gets emitted depends " +
			"entirely on the `json_collector` stanzas in your config. " +
			"See the exporter documentation for configuration syntax.",
		Entries: nil,
	}
}
