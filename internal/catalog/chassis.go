package catalog

// Chassis returns the catalog for the chassis collector.
//
// See internal/collector/chassis_collector.go for the emission code and
// gofish traversal that backs these entries.
func Chassis() Module {
	return Module{
		Name: "chassis",
		Entries: concat(
			chassisRootEntries(),
			chassisThermalEntries(),
			chassisLeakEntries(),
			chassisPowerEntries(),
			chassisNetworkEntries(),
			chassisPhysicalSecurityEntries(),
			chassisLogServiceEntries(),
		),
	}
}

// chassisRootEntries — /redfish/v1/Chassis/{chassis_id}.
func chassisRootEntries() []Entry {
	rootLabels := []Label{
		labelResource("chassis"),
		chassisIDLabel,
	}
	modelLabels := append([]Label{}, rootLabels...)
	modelLabels = append(modelLabels,
		Label{Name: "manufacturer", Field: "Manufacturer",
			Description: "Chassis manufacturer name as reported by Redfish."},
		Label{Name: "model", Field: "Model",
			Description: "Chassis model string as reported by the manufacturer."},
		Label{Name: "part_number", Field: "PartNumber",
			Description: "Manufacturer-assigned chassis part number."},
		Label{Name: "sku", Field: "SKU",
			Description: "Manufacturer-assigned chassis SKU."},
	)

	return []Entry{
		{
			Name:       "redfish_chassis_health",
			Help:       "health of chassis,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: "/redfish/v1/Chassis/{chassis_id}", Field: "Status.Health"},
			Labels:     rootLabels,
		},
		{
			Name:       "redfish_chassis_health_rollup",
			Help:       "health rollup of chassis,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: "/redfish/v1/Chassis/{chassis_id}", Field: "Status.HealthRollup"},
			Labels:     rootLabels,
		},
		{
			Name:       "redfish_chassis_state",
			Help:       "state of chassis,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: "/redfish/v1/Chassis/{chassis_id}", Field: "Status.State"},
			Labels:     rootLabels,
		},
		{
			Name:       "redfish_chassis_model_info",
			Help:       "organization responsible for producing the chassis, the name by which the manufacturer generally refers to the chassis, and a part number and sku assigned by the organization that is responsible for producing or manufacturing the chassis",
			MetricType: MetricInfo,
			ValueType:  ValueConstant,
			Value:      Source{Path: "/redfish/v1/Chassis/{chassis_id}", Field: "(constant 1; identifying data carried on labels)"},
			Labels:     modelLabels,
		},
	}
}

// chassisThermalEntries — /redfish/v1/Chassis/{chassis_id}/Thermal (Temperatures[*], Fans[*]).
func chassisThermalEntries() []Entry {
	tempLabels := []Label{
		labelResource("temperature"),
		chassisIDLabelParent,
		{Name: "sensor", Field: "Temperatures[*].Name",
			Description: "Human-readable temperature sensor name (e.g. \"CPU1 Temp\")."},
		{Name: "sensor_id", Field: "Temperatures[*].MemberId",
			Description: "Stable per-chassis temperature sensor identifier."},
	}
	fanLabels := []Label{
		labelResource("fan"),
		chassisIDLabelParent,
		{Name: "fan", Field: "Fans[*].Name",
			Description: "Human-readable fan name."},
		{Name: "fan_id", Field: "Fans[*].MemberId",
			Description: "Stable per-chassis fan identifier."},
		{Name: "fan_unit", Field: "Fans[*].ReadingUnits",
			Description: "Fan reading unit lowercased (e.g. \"rpm\", \"percent\")."},
	}
	thermalPath := "/redfish/v1/Chassis/{chassis_id}/Thermal"

	return []Entry{
		{
			Name:       "redfish_chassis_temperature_sensor_state",
			Help:       "status state of temperature on this chassis component,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: thermalPath, Field: "Temperatures[*].Status.State"},
			Labels:     tempLabels,
		},
		{
			Name:       "redfish_chassis_temperature_sensor_health",
			Help:       "status health of temperature on this chassis component,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: thermalPath, Field: "Temperatures[*].Status.Health"},
			Labels:     tempLabels,
		},
		{
			Name:       "redfish_chassis_temperature_celsius",
			Help:       "celsius of temperature on this chassis component",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: thermalPath, Field: "Temperatures[*].ReadingCelsius"},
			Labels:     tempLabels,
		},
		{
			Name:       "redfish_chassis_fan_health",
			Help:       "fan health on this chassis component,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: thermalPath, Field: "Fans[*].Status.Health"},
			Labels:     fanLabels,
		},
		{
			Name:       "redfish_chassis_fan_state",
			Help:       "fan state on this chassis component,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: thermalPath, Field: "Fans[*].Status.State"},
			Labels:     fanLabels,
		},
		{
			Name:       "redfish_chassis_fan_rpm",
			Help:       "fan RPM or percentage on this chassis component",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: thermalPath, Field: "Fans[*].Reading"},
			Labels:     fanLabels,
		},
		{
			Name:       "redfish_chassis_fan_rpm_percentage",
			Help:       "fan RPM, as a percentage of the min-max RPMs possible, on this chassis component",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: thermalPath, Field: "(derived from Fans[*].Reading, MinReadingRange, MaxReadingRange, thresholds)"},
			Labels:     fanLabels,
		},
		{
			Name:       "redfish_chassis_fan_rpm_min",
			Help:       "lowest possible fan RPM or percentage, on this chassis component",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: thermalPath, Field: "Fans[*].MinReadingRange"},
			Labels:     fanLabels,
		},
		{
			Name:       "redfish_chassis_fan_rpm_max",
			Help:       "highest possible fan RPM or percentage, on this chassis component",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: thermalPath, Field: "Fans[*].MaxReadingRange"},
			Labels:     fanLabels,
		},
		{
			Name:       "redfish_chassis_fan_rpm_lower_threshold_critical",
			Help:       "threshold below the normal range fan RPM or percentage, but not fatal, on this chassis component",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: thermalPath, Field: "Fans[*].LowerThresholdCritical"},
			Labels:     fanLabels,
		},
		{
			Name:       "redfish_chassis_fan_rpm_lower_threshold_non_critical",
			Help:       "threshold below the normal range fan RPM or percentage, but not critical, on this chassis component",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: thermalPath, Field: "Fans[*].LowerThresholdNonCritical"},
			Labels:     fanLabels,
		},
		{
			Name:       "redfish_chassis_fan_rpm_lower_threshold_fatal",
			Help:       "threshold below the normal range fan RPM or percentage, and is fatal, on this chassis component",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: thermalPath, Field: "Fans[*].LowerThresholdFatal"},
			Labels:     fanLabels,
		},
		{
			Name:       "redfish_chassis_fan_rpm_upper_threshold_critical",
			Help:       "threshold above the normal range fan RPM or percentage, but not fatal, on this chassis component",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: thermalPath, Field: "Fans[*].UpperThresholdCritical"},
			Labels:     fanLabels,
		},
		{
			Name:       "redfish_chassis_fan_rpm_upper_threshold_non_critical",
			Help:       "threshold above the normal range fan RPM or percentage, but not critical, on this chassis component",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: thermalPath, Field: "Fans[*].UpperThresholdNonCritical"},
			Labels:     fanLabels,
		},
		{
			Name:       "redfish_chassis_fan_rpm_upper_threshold_fatal",
			Help:       "threshold above the normal range fan RPM or percentage, and is fatal, on this chassis component",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: thermalPath, Field: "Fans[*].UpperThresholdFatal"},
			Labels:     fanLabels,
		},
	}
}

// chassisLeakEntries — /redfish/v1/Chassis/{chassis_id}/ThermalSubsystem/LeakDetection/LeakDetectors/{leak_detector_id}.
func chassisLeakEntries() []Entry {
	labels := []Label{
		labelResource("leak_detector"),
		chassisIDLabelParent,
		{Name: "leak_detection_id", Field: "", Constant: "LeakDetection",
			Description: "Constant string \"LeakDetection\" identifying the parent collection."},
		{Name: "leak_detector_id", Field: "Id",
			Description: "Leak detector Redfish identifier; matches the {leak_detector_id} URL segment."},
	}
	return []Entry{
		{
			Name:       "redfish_chassis_leak_detector_health",
			Help:       "chassis leak detector health state,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value: Source{
				Path:  "/redfish/v1/Chassis/{chassis_id}/ThermalSubsystem/LeakDetection/LeakDetectors/{leak_detector_id}",
				Field: "Status.Health",
			},
			Labels: labels,
		},
	}
}

// chassisPowerEntries — /redfish/v1/Chassis/{chassis_id}/Power (Voltages[*], PowerControl[*], PowerSupplies[*]).
func chassisPowerEntries() []Entry {
	powerPath := "/redfish/v1/Chassis/{chassis_id}/Power"

	voltageLabels := []Label{
		labelResource("power_voltage"),
		chassisIDLabelParent,
		{Name: "power_voltage", Field: "Voltages[*].Name",
			Description: "Human-readable voltage sensor name."},
		{Name: "power_voltage_id", Field: "Voltages[*].MemberId",
			Description: "Stable per-chassis voltage sensor identifier."},
	}
	powerControlLabels := []Label{
		labelResource("power_wattage"),
		chassisIDLabelParent,
		{Name: "power_voltage", Field: "PowerControl[*].Name",
			Description: "PowerControl entry name. Reuses the voltage label slot (exporter quirk)."},
		{Name: "power_voltage_id", Field: "PowerControl[*].MemberId",
			Description: "PowerControl entry MemberId. Reuses the voltage label slot (exporter quirk)."},
	}
	psuLabels := []Label{
		labelResource("power_supply"),
		chassisIDLabelParent,
		{Name: "power_supply", Field: "PowerSupplies[*].Name",
			Description: "Human-readable power supply name."},
		{Name: "power_supply_id", Field: "PowerSupplies[*].MemberId",
			Description: "PSU MemberId; falls back to PowerSupplies[*].SerialNumber when MemberId is empty."},
	}

	return []Entry{
		{
			Name:       "redfish_chassis_power_voltage_state",
			Help:       "power voltage state of chassis component,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: powerPath, Field: "Voltages[*].Status.State"},
			Labels:     voltageLabels,
		},
		{
			Name:       "redfish_chassis_power_voltage_volts",
			Help:       "power voltage volts number of chassis component",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: powerPath, Field: "Voltages[*].ReadingVolts"},
			Labels:     voltageLabels,
		},
		{
			Name:       "redfish_chassis_power_average_consumed_watts",
			Help:       "power wattage watts number of chassis component",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: powerPath, Field: "PowerControl[*].PowerMetrics.AverageConsumedWatts"},
			Labels:     powerControlLabels,
		},
		{
			Name:       "redfish_chassis_power_powersupply_state",
			Help:       "powersupply state of chassis component,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: powerPath, Field: "PowerSupplies[*].Status.State"},
			Labels:     psuLabels,
		},
		{
			Name:       "redfish_chassis_power_powersupply_health",
			Help:       "powersupply health of chassis component,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: powerPath, Field: "PowerSupplies[*].Status.Health"},
			Labels:     psuLabels,
		},
		{
			Name:       "redfish_chassis_power_powersupply_power_efficiency_percentage",
			Help:       "rated efficiency, as a percentage, of the associated power supply on this chassis",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: powerPath, Field: "PowerSupplies[*].EfficiencyPercent"},
			Labels:     psuLabels,
		},
		{
			Name:       "redfish_chassis_power_powersupply_last_power_output_watts",
			Help:       "average power output, measured in Watts, of the associated power supply on this chassis",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: powerPath, Field: "PowerSupplies[*].LastPowerOutputWatts"},
			Labels:     psuLabels,
		},
		{
			Name:       "redfish_chassis_power_powersupply_power_input_watts",
			Help:       "measured input power, in Watts, of powersupply on this chassis",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: powerPath, Field: "PowerSupplies[*].PowerInputWatts"},
			Labels:     psuLabels,
		},
		{
			Name:       "redfish_chassis_power_powersupply_power_output_watts",
			Help:       "measured output power, in Watts, of powersupply on this chassis",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: powerPath, Field: "PowerSupplies[*].PowerOutputWatts"},
			Labels:     psuLabels,
		},
		{
			Name:       "redfish_chassis_power_powersupply_power_capacity_watts",
			Help:       "power_capacity_watts of powersupply on this chassis",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: powerPath, Field: "PowerSupplies[*].PowerCapacityWatts"},
			Labels:     psuLabels,
		},
	}
}

// chassisNetworkEntries — /redfish/v1/Chassis/{chassis_id}/NetworkAdapters/... .
func chassisNetworkEntries() []Entry {
	adapterPath := "/redfish/v1/Chassis/{chassis_id}/NetworkAdapters/{network_adapter_id}"
	portPath := adapterPath + "/NetworkPorts/{network_port_id}"

	adapterLabels := []Label{
		labelResource("network_adapter"),
		chassisIDLabelParent,
		{Name: "network_adapter", Field: "Name",
			Description: "Human-readable network adapter name."},
		{Name: "network_adapter_id", Field: "Id",
			Description: "Network adapter Redfish identifier; matches {network_adapter_id}."},
	}
	portLabels := []Label{
		labelResource("network_port"),
		chassisIDLabelParent,
		{Name: "network_adapter", Path: adapterPath, Field: "Name",
			Description: "Parent network adapter name."},
		{Name: "network_adapter_id", Path: adapterPath, Field: "Id",
			Description: "Parent network adapter Redfish identifier."},
		{Name: "network_port", Field: "Name",
			Description: "Human-readable network port name."},
		{Name: "network_port_id", Field: "Id",
			Description: "Network port Redfish identifier; matches {network_port_id}."},
		{Name: "network_port_type", Field: "ActiveLinkTechnology",
			Description: "Active link technology (e.g. \"Ethernet\", \"FibreChannel\")."},
		{Name: "network_port_speed", Field: "CurrentLinkSpeedMbps",
			Description: "Current link speed formatted as \"<N> Mbps\"."},
		{Name: "network_port_connectiont_type", Field: "FCPortConnectionType",
			Description: "FC port connection type (label name preserves historical typo)."},
		{Name: "network_physical_port_number", Field: "PhysicalPortNumber",
			Description: "Physical port number on the adapter."},
	}

	return []Entry{
		{
			Name:       "redfish_chassis_network_adapter_state",
			Help:       "chassis network adapter state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: adapterPath, Field: "Status.State"},
			Labels:     adapterLabels,
		},
		{
			Name:       "redfish_chassis_network_adapter_health_state",
			Help:       "chassis network adapter health state,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: adapterPath, Field: "Status.Health"},
			Labels:     adapterLabels,
		},
		{
			Name:       "redfish_chassis_network_port_state",
			Help:       "chassis network port state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: portPath, Field: "Status.State"},
			Labels:     portLabels,
		},
		{
			Name:       "redfish_chassis_network_port_health_state",
			Help:       "chassis network port health state,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: portPath, Field: "Status.Health"},
			Labels:     portLabels,
		},
		{
			Name:       "redfish_chassis_network_port_link_state",
			Help:       "chassis network port link state state,1(Up),0(Down)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumPortLinkState,
			Value:      Source{Path: portPath, Field: "LinkStatus"},
			Labels:     portLabels,
		},
	}
}

// chassisPhysicalSecurityEntries — /redfish/v1/Chassis/{chassis_id}#PhysicalSecurity.
func chassisPhysicalSecurityEntries() []Entry {
	labels := []Label{
		labelResource("physical_security"),
		chassisIDLabel,
		{Name: "intrusion_sensor_number", Field: "PhysicalSecurity.IntrusionSensorNumber",
			Description: "Intrusion sensor slot number (stringified)."},
		{Name: "intrusion_sensor_rearm", Field: "PhysicalSecurity.IntrusionSensorReArm",
			Description: "How the sensor is re-armed after tripping (e.g. \"Manual\", \"Automatic\")."},
	}
	return []Entry{
		{
			Name:       "redfish_chassis_physical_security_sensor_state",
			Help:       "indicates the known state of the physical security sensor, such as if it is hardware intrusion detected,1(Normal),2(TamperingDetected),3(HardwareIntrusion)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumIntrusionSensor,
			Value: Source{
				Path:  "/redfish/v1/Chassis/{chassis_id}",
				Field: "PhysicalSecurity.IntrusionSensor",
			},
			Labels: labels,
		},
	}
}

// chassisLogServiceEntries — /redfish/v1/Chassis/{chassis_id}/LogServices/{log_service_id}.
// These metrics are declared by the collector but current chassis emission code
// does not iterate LogServices; entries are cataloged for future scrape work.
func chassisLogServiceEntries() []Entry {
	labels := []Label{
		{Name: "chassis_id", Path: "/redfish/v1/Chassis/{chassis_id}", Field: "Id",
			Description: "Redfish chassis identifier this log service belongs to."},
		{Name: "log_service", Field: "Name",
			Description: "Human-readable log service name."},
		{Name: "log_service_id", Field: "Id",
			Description: "Log service Redfish identifier; matches {log_service_id}."},
		{Name: "log_service_enabled", Field: "ServiceEnabled",
			Description: "Whether the log service is enabled (\"true\"/\"false\" as string)."},
		{Name: "log_service_overwrite_policy", Field: "OverWritePolicy",
			Description: "Ring-buffer overwrite policy (e.g. \"WrapsWhenFull\", \"NeverOverwrites\")."},
	}
	path := "/redfish/v1/Chassis/{chassis_id}/LogServices/{log_service_id}"
	return []Entry{
		{
			Name:       "redfish_chassis_log_service_state",
			Help:       "chassis log service state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: path, Field: "Status.State"},
			Labels:     labels,
		},
		{
			Name:       "redfish_chassis_log_service_health_state",
			Help:       "chassis log service health state,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: path, Field: "Status.Health"},
			Labels:     labels,
		},
	}
}

// Shared label helpers.

var chassisIDLabel = Label{
	Name:        "chassis_id",
	Field:       "Id",
	Description: "Chassis Redfish identifier; matches the {chassis_id} URL segment.",
}

var chassisIDLabelParent = Label{
	Name:        "chassis_id",
	Path:        "/redfish/v1/Chassis/{chassis_id}",
	Field:       "Id",
	Description: "Parent chassis identifier propagated from the enclosing Chassis resource.",
}

func labelResource(value string) Label {
	return Label{
		Name:        "resource",
		Field:       "",
		Constant:    value,
		Description: "Constant string \"" + value + "\" identifying the metric group; not sourced from Redfish.",
	}
}

func concat(groups ...[]Entry) []Entry {
	var total int
	for _, g := range groups {
		total += len(g)
	}
	out := make([]Entry, 0, total)
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}
