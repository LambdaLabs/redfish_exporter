package catalog

// Powershelf returns the catalog for the power shelf collector.
//
// See internal/collector/powershelf_collector.go and its per-vendor adapters
// (powershelf_liteon.go, powershelf_delta.go). The collector selects a
// Chassis by Manufacturer (LITEON/DELTA), reads its Sensors collection at
// /redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}, and its
// PowerSubsystem/PowerSupplies for status. Each vendor adapter maps its
// raw sensor IDs to the canonical metric names below.
//
// Value.Field on Sensor-sourced entries is always the sensor Reading; the
// mapping from Redfish sensor id → canonical metric is embedded in the
// adapter (see docs/powershelf-metrics-contract.md for the vendor tables).
func Powershelf() Module {
	return Module{
		Name: "powershelf",
		Entries: concat(
			powershelfPSUEntries(),
			powershelfSensorCatchallEntries(),
			powershelfShelfEntries(),
		),
	}
}

func powershelfPSUEntries() []Entry {
	psuLabels := []Label{
		{Name: "power_supply_id", Field: "",
			Description: "Canonical PSU id (\"ps1\"..\"ps6\") produced by the vendor adapter from PowerSupplies[*].Id."},
	}
	sensorPath := "/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}"
	psuPath := "/redfish/v1/Chassis/{chassis_id}/PowerSubsystem/PowerSupplies/{psu_id}"

	sensorFloat := func(name, help string) Entry {
		return Entry{
			Name: name, Help: help,
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value:  Source{Path: sensorPath, Field: "Reading (sensor id → PSU by vendor adapter)"},
			Labels: psuLabels,
		}
	}

	return []Entry{
		sensorFloat("redfish_powershelf_input_voltage", "PSU AC input voltage, volts"),
		{
			Name: "redfish_powershelf_psu_health", Help: "PSU health,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge, ValueType: ValueIntEnum, Enum: EnumCommonHealth,
			Value:  Source{Path: psuPath, Field: "Status.Health"},
			Labels: psuLabels,
		},
		{
			Name: "redfish_powershelf_psu_state", Help: "PSU state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge, ValueType: ValueIntEnum, Enum: EnumCommonState,
			Value:  Source{Path: psuPath, Field: "Status.State"},
			Labels: psuLabels,
		},
		sensorFloat("redfish_powershelf_output_voltage", "PSU DC output (rail) voltage, volts"),
		sensorFloat("redfish_powershelf_input_current", "PSU input current, amperes"),
		sensorFloat("redfish_powershelf_output_current", "PSU output current, amperes"),
		sensorFloat("redfish_powershelf_input_power", "PSU input power, watts"),
		sensorFloat("redfish_powershelf_output_power", "PSU output power, watts"),
		sensorFloat("redfish_powershelf_standby_output_voltage", "PSU standby rail voltage, volts"),
		sensorFloat("redfish_powershelf_standby_output_current", "PSU standby rail current, amperes"),
		sensorFloat("redfish_powershelf_standby_output_power", "PSU standby rail power, watts"),
		sensorFloat("redfish_powershelf_powerfactor", "PSU power factor, percent (0-100)"),
		sensorFloat("redfish_powershelf_energy_in", "PSU input energy (per-interval gauge)"),
		sensorFloat("redfish_powershelf_energy_out", "PSU output energy (per-interval gauge)"),
		{
			Name: "redfish_powershelf_total_energy_in", Help: "PSU accumulated input energy (counter)",
			MetricType: MetricCounter, ValueType: ValueFloat,
			Value:  Source{Path: sensorPath, Field: "Reading (sensor id → PSU by vendor adapter)"},
			Labels: psuLabels,
		},
		sensorFloat("redfish_powershelf_fan1", "PSU fan 1 speed, RPM"),
		sensorFloat("redfish_powershelf_fan2", "PSU fan 2 speed, RPM"),
		sensorFloat("redfish_powershelf_temp_input", "PSU intake (ambient) temperature, celsius"),
		sensorFloat("redfish_powershelf_temp_output", "PSU exhaust temperature, celsius"),
		sensorFloat("redfish_powershelf_temp_hotspot", "PSU hotspot temperature, celsius"),
		sensorFloat("redfish_powershelf_temp_clip_plus", "PSU DC clip+ temperature, celsius"),
		sensorFloat("redfish_powershelf_temp_clip_minus", "PSU DC clip- temperature, celsius"),
		sensorFloat("redfish_powershelf_input_frequency", "PSU AC input frequency, hertz"),
	}
}

func powershelfSensorCatchallEntries() []Entry {
	labels := []Label{
		{Name: "sensor_id", Field: "Id",
			Description: "Raw Redfish sensor identifier (unmodified by the adapter)."},
	}
	sensorPath := "/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}"
	entry := func(name, help string) Entry {
		return Entry{
			Name: name, Help: help,
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value:  Source{Path: sensorPath, Field: "Reading (mapped by ReadingType, uncurated)"},
			Labels: labels,
		}
	}
	return []Entry{
		entry("redfish_powershelf_sensor_volts", "uncurated sensor reading, volts"),
		entry("redfish_powershelf_sensor_amperes", "uncurated sensor reading, amperes"),
		entry("redfish_powershelf_sensor_watts", "uncurated sensor reading, watts"),
		entry("redfish_powershelf_sensor_celsius", "uncurated sensor reading, celsius"),
		entry("redfish_powershelf_sensor_percent", "uncurated sensor reading, percent"),
		entry("redfish_powershelf_sensor_rpm", "uncurated sensor reading, rpm"),
	}
}

func powershelfShelfEntries() []Entry {
	sensorPath := "/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}"
	shelf := func(name, help string) Entry {
		return Entry{
			Name: name, Help: help,
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value:  Source{Path: sensorPath, Field: "Reading (shelf-level sensor id mapped by adapter)"},
			Labels: nil,
		}
	}
	return []Entry{
		shelf("redfish_powershelf_total_power_in", "shelf total input power, watts"),
		shelf("redfish_powershelf_total_power_in_A", "shelf total input power on phase A, watts"),
		shelf("redfish_powershelf_total_power_out", "shelf total output power, watts"),
		shelf("redfish_powershelf_total_current_out", "shelf total output current, amperes"),
		shelf("redfish_powershelf_average_current_out", "shelf average per-PSU output current, amperes"),
		shelf("redfish_powershelf_max_voltage_out", "shelf maximum output (rail) voltage, volts"),
		shelf("redfish_powershelf_power_load", "shelf load, percent (0-100)"),
		shelf("redfish_powershelf_total_efficiency", "shelf conversion efficiency, percent (0-100)"),
		shelf("redfish_powershelf_current_share", "shelf load-share current, amperes"),
		shelf("redfish_powershelf_voltage_in_A_A", "shelf AC input voltage, phase A, volts"),
		shelf("redfish_powershelf_voltage_in_A_B", "shelf AC input voltage, phase B, volts"),
		shelf("redfish_powershelf_voltage_in_A_C", "shelf AC input voltage, phase C, volts"),
		shelf("redfish_powershelf_current_in_A_A", "shelf AC input current, phase A, amperes"),
		shelf("redfish_powershelf_current_in_A_B", "shelf AC input current, phase B, amperes"),
		shelf("redfish_powershelf_current_in_A_C", "shelf AC input current, phase C, amperes"),
		shelf("redfish_powershelf_bmc_12v", "shelf BMC 12V rail, volts"),
		shelf("redfish_powershelf_bmc_3v3", "shelf BMC 3.3V rail, volts"),
		shelf("redfish_powershelf_bmc_temp", "shelf BMC temperature, celsius"),
		shelf("redfish_powershelf_hotswap_input_current", "shelf hotswap input current, amperes"),
		shelf("redfish_powershelf_hotswap_input_power", "shelf hotswap input power, watts"),
		shelf("redfish_powershelf_hotswap_input_voltage", "shelf hotswap input voltage, volts"),
		shelf("redfish_powershelf_hotswap_output_voltage", "shelf hotswap output voltage, volts"),
		shelf("redfish_powershelf_hotswap_temp", "shelf hotswap temperature, celsius"),
		shelf("redfish_powershelf_total_current_in", "shelf total input current, amperes"),
		shelf("redfish_powershelf_temp_shelf", "shelf ambient temperature, celsius"),
		shelf("redfish_powershelf_dc_temp_plus", "shelf DC bus (+) temperature, celsius"),
		shelf("redfish_powershelf_dc_temp_minus", "shelf DC bus (-) temperature, celsius"),
	}
}
