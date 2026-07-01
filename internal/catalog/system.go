package catalog

// System returns the catalog for the system collector.
//
// See internal/collector/system_collector.go for the emission code. The
// collector iterates every Redfish System and drills into Memory,
// Processors (including ProcessorMetrics for PCIe/cache errors), Storage
// (Volumes, Drives), PCIeDevices, PCIeFunctions, NetworkInterfaces, and
// EthernetInterfaces.
func System() Module {
	return Module{
		Name: "system",
		Entries: concat(
			systemRootEntries(),
			systemBIOSEntries(),
			systemSummaryEntries(),
			systemMemoryEntries(),
			systemProcessorEntries(),
			systemStorageEntries(),
			systemPCIeEntries(),
			systemNetworkEntries(),
			systemEthernetEntries(),
			systemLogServiceEntries(),
		),
	}
}

// systemRootEntries — /redfish/v1/Systems/{system_id}.
func systemRootEntries() []Entry {
	labels := []Label{
		labelResource("system"),
		systemIDLabel,
	}
	path := "/redfish/v1/Systems/{system_id}"
	return []Entry{
		{
			Name:       "redfish_system_state",
			Help:       "system state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: path, Field: "Status.State"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_health_state",
			Help:       "system health,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: path, Field: "Status.Health"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_power_state",
			Help:       "system power state",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumPowerState,
			Value:      Source{Path: path, Field: "PowerState"},
			Labels:     labels,
		},
	}
}

// systemBIOSEntries — bios_info uses labels {bios_version, model} only.
func systemBIOSEntries() []Entry {
	return []Entry{
		{
			Name:       "redfish_system_bios_info",
			Help:       "host BIOS version (info metric, always 1)",
			MetricType: MetricInfo,
			ValueType:  ValueConstant,
			Value: Source{
				Path:  "/redfish/v1/Systems/{system_id}",
				Field: "(constant 1; identifying data carried on labels)",
			},
			Labels: []Label{
				{Name: "bios_version", Field: "BiosVersion",
					Description: "Host BIOS version string; empty BiosVersion values (e.g. HGX baseboards) are skipped."},
				{Name: "model", Field: "Model",
					Description: "System model string as reported by the manufacturer."},
			},
		},
	}
}

// systemSummaryEntries — ProcessorSummary and MemorySummary on the system resource.
func systemSummaryEntries() []Entry {
	labels := []Label{
		labelResource("system"),
		systemIDLabel,
	}
	path := "/redfish/v1/Systems/{system_id}"
	return []Entry{
		{
			Name:       "redfish_system_total_memory_state",
			Help:       "system overall memory state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: path, Field: "MemorySummary.Status.State"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_total_memory_health_state",
			Help:       "system overall memory health,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: path, Field: "MemorySummary.Status.Health"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_total_memory_size",
			Help:       "system total memory size, GiB",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: path, Field: "MemorySummary.TotalSystemMemoryGiB"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_total_processor_state",
			Help:       "system overall processor state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: path, Field: "ProcessorSummary.Status.State"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_total_processor_health_state",
			Help:       "system overall processor health,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: path, Field: "ProcessorSummary.Status.Health"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_total_processor_count",
			Help:       "system total processor count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: path, Field: "ProcessorSummary.Count"},
			Labels:     labels,
		},
	}
}

// systemMemoryEntries — /redfish/v1/Systems/{system_id}/Memory/{memory_id}.
// Note: the exporter emits these with labels [resource, memory, memory_id]
// (no system_id), following the collector's parseMemory function.
func systemMemoryEntries() []Entry {
	labels := []Label{
		labelResource("memory"),
		{Name: "memory", Field: "Name",
			Description: "Human-readable memory module name."},
		{Name: "memory_id", Field: "Id",
			Description: "Memory module Redfish identifier; matches the {memory_id} URL segment."},
	}
	path := "/redfish/v1/Systems/{system_id}/Memory/{memory_id}"
	return []Entry{
		{
			Name:       "redfish_system_memory_state",
			Help:       "system memory state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: path, Field: "Status.State"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_memory_health_state",
			Help:       "system memory health state,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: path, Field: "Status.Health"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_memory_capacity",
			Help:       "system memory capacity, MiB",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: path, Field: "CapacityMiB"},
			Labels:     labels,
		},
	}
}

// systemProcessorEntries — /redfish/v1/Systems/{system_id}/Processors/{processor_id}
// plus /redfish/v1/Systems/{system_id}/Processors/{processor_id}/ProcessorMetrics.
func systemProcessorEntries() []Entry {
	labels := []Label{
		labelResource("processor"),
		{Name: "processor", Field: "Name",
			Description: "Human-readable processor name."},
		{Name: "processor_id", Field: "Id",
			Description: "Processor Redfish identifier; matches the {processor_id} URL segment."},
	}
	procPath := "/redfish/v1/Systems/{system_id}/Processors/{processor_id}"
	metricsPath := procPath + "/ProcessorMetrics"

	return []Entry{
		{
			Name:       "redfish_system_processor_state",
			Help:       "system processor state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: procPath, Field: "Status.State"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_processor_health_state",
			Help:       "system processor health state,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: procPath, Field: "Status.Health"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_processor_health_rollup",
			Help:       "system processor health rollup,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: procPath, Field: "Status.HealthRollup"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_processor_total_threads",
			Help:       "system processor total threads",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: procPath, Field: "TotalThreads"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_processor_total_cores",
			Help:       "system processor total cores",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: procPath, Field: "TotalCores"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_processor_pcie_errors_l0_to_recovery_count",
			Help:       "system processor PCIe L0 to recovery state transition count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "PCIeErrors.L0ToRecoveryCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_processor_pcie_errors_correctable_count",
			Help:       "system processor PCIe correctable error count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "PCIeErrors.CorrectableErrorCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_processor_pcie_errors_fatal_count",
			Help:       "system processor PCIe fatal error count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "PCIeErrors.FatalErrorCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_processor_pcie_errors_non_fatal_count",
			Help:       "system processor PCIe non-fatal error count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "PCIeErrors.NonFatalErrorCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_processor_pcie_errors_nak_received_count",
			Help:       "system processor PCIe NAK received count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "PCIeErrors.NAKReceivedCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_processor_pcie_errors_nak_sent_count",
			Help:       "system processor PCIe NAK sent count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "PCIeErrors.NAKSentCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_processor_pcie_errors_replay_count",
			Help:       "system processor PCIe replay count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "PCIeErrors.ReplayCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_processor_pcie_errors_replay_rollover_count",
			Help:       "system processor PCIe replay rollover count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "PCIeErrors.ReplayRolloverCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_processor_cache_lifetime_uncorrectable_ecc_error_count",
			Help:       "system processor cache lifetime uncorrectable ECC error count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "CacheMetricsTotal.LifeTime.UncorrectableECCErrorCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_processor_cache_lifetime_correctable_ecc_error_count",
			Help:       "system processor cache lifetime correctable ECC error count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "CacheMetricsTotal.LifeTime.CorrectableECCErrorCount"},
			Labels:     labels,
		},
	}
}

// systemStorageEntries — /redfish/v1/Systems/{system_id}/Storage/{storage_id}/{Volumes,Drives}.
func systemStorageEntries() []Entry {
	volumeLabels := []Label{
		labelResource("volume"),
		{Name: "volume", Field: "Name",
			Description: "Human-readable volume name."},
		{Name: "volume_id", Field: "Id",
			Description: "Volume Redfish identifier."},
	}
	driveLabels := []Label{
		labelResource("drive"),
		{Name: "drive", Field: "Name",
			Description: "Human-readable drive name."},
		{Name: "drive_id", Field: "Id",
			Description: "Drive Redfish identifier."},
		{
			Name:        "storage_controller_id",
			Path:        "/redfish/v1/Systems/{system_id}/Storage/{storage_controller_id}",
			Field:       "Id",
			Description: "Parent Storage resource Id (the collector labels this as storage_controller_id).",
		},
	}
	ctrlLabels := []Label{
		labelResource("storage_controller"),
		{Name: "storage_controller", Field: "Name",
			Description: "Human-readable storage controller name."},
		{Name: "storage_controller_id", Field: "Id",
			Description: "Storage controller Redfish identifier."},
	}

	volumePath := "/redfish/v1/Systems/{system_id}/Storage/{storage_controller_id}/Volumes/{volume_id}"
	drivePath := "/redfish/v1/Systems/{system_id}/Storage/{storage_controller_id}/Drives/{drive_id}"
	ctrlPath := "/redfish/v1/Systems/{system_id}/Storage/{storage_controller_id}"

	return []Entry{
		{
			Name:       "redfish_system_storage_volume_state",
			Help:       "system storage volume state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: volumePath, Field: "Status.State"},
			Labels:     volumeLabels,
		},
		{
			Name:       "redfish_system_storage_volume_health_state",
			Help:       "system storage volume health state,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: volumePath, Field: "Status.Health"},
			Labels:     volumeLabels,
		},
		{
			Name:       "redfish_system_storage_volume_capacity",
			Help:       "system storage volume capacity, Bytes",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: volumePath, Field: "CapacityBytes"},
			Labels:     volumeLabels,
		},
		{
			Name:       "redfish_system_storage_drive_state",
			Help:       "system storage drive state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: drivePath, Field: "Status.State"},
			Labels:     driveLabels,
		},
		{
			Name:       "redfish_system_storage_drive_health_state",
			Help:       "system storage drive health state,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: drivePath, Field: "Status.Health"},
			Labels:     driveLabels,
		},
		{
			Name:       "redfish_system_storage_drive_capacity",
			Help:       "system storage drive capacity, Bytes",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: drivePath, Field: "CapacityBytes"},
			Labels:     driveLabels,
		},
		{
			Name:       "redfish_system_storage_controller_state",
			Help:       "system storage controller state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: ctrlPath, Field: "Status.State"},
			Labels:     ctrlLabels,
		},
		{
			Name:       "redfish_system_storage_controller_health_state",
			Help:       "system storage controller health state,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: ctrlPath, Field: "Status.Health"},
			Labels:     ctrlLabels,
		},
	}
}

// systemPCIeEntries — /redfish/v1/Systems/{system_id}/PCIeDevices/{pcie_device_id}
// and /redfish/v1/Systems/{system_id}/PCIeFunctions/{pcie_function_id}.
func systemPCIeEntries() []Entry {
	deviceLabels := []Label{
		labelResource("pcie_device"),
		{Name: "pcie_device", Field: "Name",
			Description: "PCIe device human-readable name."},
		{Name: "pcie_device_id", Field: "Id",
			Description: "PCIe device Redfish identifier."},
		{Name: "pcie_device_partnumber", Field: "PartNumber",
			Description: "Manufacturer part number of the PCIe device."},
		{Name: "pcie_device_type", Field: "DeviceType",
			Description: "PCIe device type enumeration."},
		{Name: "pcie_serial_number", Field: "SerialNumber",
			Description: "Serial number of the PCIe device."},
	}
	functionLabels := []Label{
		labelResource("pcie_function"),
		{Name: "pcie_function_name", Field: "Name",
			Description: "PCIe function human-readable name."},
		{Name: "pcie_function_id", Field: "Id",
			Description: "PCIe function Redfish identifier (stringified integer)."},
		{Name: "pci_function_deviceclass", Field: "DeviceClass",
			Description: "PCIe function device class enumeration."},
		{Name: "pci_function_type", Field: "FunctionType",
			Description: "PCIe function type (e.g. Physical, Virtual)."},
	}

	devicePath := "/redfish/v1/Systems/{system_id}/PCIeDevices/{pcie_device_id}"
	functionPath := "/redfish/v1/Systems/{system_id}/PCIeFunctions/{pcie_function_id}"

	return []Entry{
		{
			Name:       "redfish_system_pcie_device_state",
			Help:       "system pcie device state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: devicePath, Field: "Status.State"},
			Labels:     deviceLabels,
		},
		{
			Name:       "redfish_system_pcie_device_health_state",
			Help:       "system pcie device health state,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: devicePath, Field: "Status.Health"},
			Labels:     deviceLabels,
		},
		{
			Name:       "redfish_system_pcie_function_state",
			Help:       "system pcie function state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: functionPath, Field: "Status.State"},
			Labels:     functionLabels,
		},
		{
			Name:       "redfish_system_pcie_function_health_state",
			Help:       "system pcie device function state,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: functionPath, Field: "Status.Health"},
			Labels:     functionLabels,
		},
	}
}

// systemNetworkEntries — /redfish/v1/Systems/{system_id}/NetworkInterfaces/{network_interface_id}.
func systemNetworkEntries() []Entry {
	labels := []Label{
		labelResource("network_interface"),
		{Name: "network_interface", Field: "Name",
			Description: "Network interface human-readable name."},
		{Name: "network_interface_id", Field: "Id",
			Description: "Network interface Redfish identifier."},
	}
	path := "/redfish/v1/Systems/{system_id}/NetworkInterfaces/{network_interface_id}"
	return []Entry{
		{
			Name:       "redfish_system_network_interface_state",
			Help:       "system network interface state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: path, Field: "Status.State"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_network_interface_health_state",
			Help:       "system network interface health state,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: path, Field: "Status.Health"},
			Labels:     labels,
		},
	}
}

// systemEthernetEntries — /redfish/v1/Systems/{system_id}/EthernetInterfaces/{ethernet_interface_id}.
func systemEthernetEntries() []Entry {
	labels := []Label{
		labelResource("ethernet_interface"),
		{Name: "ethernet_interface", Field: "Name",
			Description: "Ethernet interface human-readable name."},
		{Name: "ethernet_interface_id", Field: "Id",
			Description: "Ethernet interface Redfish identifier."},
		{Name: "ethernet_interface_speed", Field: "SpeedMbps",
			Description: "Interface speed formatted as \"<N> Mbps\"."},
	}
	path := "/redfish/v1/Systems/{system_id}/EthernetInterfaces/{ethernet_interface_id}"
	return []Entry{
		{
			Name:       "redfish_system_ethernet_interface_state",
			Help:       "system ethernet interface state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: path, Field: "Status.State"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_ethernet_interface_health_state",
			Help:       "system ethernet interface health state,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: path, Field: "Status.Health"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_ethernet_interface_link_status",
			Help:       "system ethernet interface link status,1(LinkUp),2(NoLink),3(LinkDown)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumEthernetLinkStatus,
			Value:      Source{Path: path, Field: "LinkStatus"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_ethernet_interface_link_enabled",
			Help:       "system ethernet interface if the link is enabled",
			MetricType: MetricGauge,
			ValueType:  ValueBool,
			Value:      Source{Path: path, Field: "InterfaceEnabled"},
			Labels:     labels,
		},
	}
}

// systemLogServiceEntries — declared metrics; current collector code does not iterate them.
func systemLogServiceEntries() []Entry {
	labels := []Label{
		{Name: "system_id", Path: "/redfish/v1/Systems/{system_id}", Field: "Id",
			Description: "Parent system identifier this log service belongs to."},
		{Name: "log_service", Field: "Name",
			Description: "Human-readable log service name."},
		{Name: "log_service_id", Field: "Id",
			Description: "Log service Redfish identifier."},
		{Name: "log_service_enabled", Field: "ServiceEnabled",
			Description: "Whether the log service is enabled."},
		{Name: "log_service_overwrite_policy", Field: "OverWritePolicy",
			Description: "Ring-buffer overwrite policy."},
	}
	path := "/redfish/v1/Systems/{system_id}/LogServices/{log_service_id}"
	return []Entry{
		{
			Name:       "redfish_system_log_service_state",
			Help:       "system log service state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: path, Field: "Status.State"},
			Labels:     labels,
		},
		{
			Name:       "redfish_system_log_service_health_state",
			Help:       "system log service health state,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: path, Field: "Status.Health"},
			Labels:     labels,
		},
	}
}

var systemIDLabel = Label{
	Name:        "system_id",
	Field:       "Id",
	Description: "System Redfish identifier; matches the {system_id} URL segment.",
}
