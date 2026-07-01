package catalog

// GPU returns the catalog for the GPU collector.
//
// See internal/collector/gpu_collector.go. The collector iterates Systems
// whose Name contains "HGX_", filters Processors to ProcessorType == "GPU",
// and pulls Memory subresources, ProcessorMetrics OEM data, and NVLink
// Ports for each GPU. Note the emitted `system_id` label carries the
// system's *Name* (not Id) — this is a quirk of the collector.
func GPU() Module {
	return Module{
		Name: "gpu",
		Entries: concat(
			gpuHealthEntries(),
			gpuMemoryEntries(),
			gpuOEMEntries(),
			gpuNVLinkEntries(),
		),
	}
}

// gpuHealthEntries — GPU processor itself.
func gpuHealthEntries() []Entry {
	baseLabels := []Label{
		gpuSystemNameLabel,
		gpuIDLabel,
	}
	infoLabels := append([]Label{}, baseLabels...)
	infoLabels = append(infoLabels,
		Label{Name: "firmware_version", Field: "FirmwareVersion",
			Description: "GPU firmware version string; \"unknown\" when absent."},
		Label{Name: "serial_number", Field: "SerialNumber",
			Description: "GPU serial number; \"unknown\" when absent."},
		Label{Name: "uuid", Field: "UUID",
			Description: "GPU UUID; \"unknown\" when absent."},
	)
	path := "/redfish/v1/Systems/{system_id}/Processors/{gpu_id}"
	return []Entry{
		{
			Name:       "redfish_gpu_state",
			Help:       "GPU processor state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: path, Field: "Status.State"},
			Labels:     baseLabels,
		},
		{
			Name:       "redfish_gpu_health",
			Help:       "GPU processor health,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: path, Field: "Status.Health"},
			Labels:     baseLabels,
		},
		{
			Name:       "redfish_gpu_info",
			Help:       "GPU information with serial number and UUID",
			MetricType: MetricInfo,
			ValueType:  ValueConstant,
			Value: Source{
				Path:  path,
				Field: "(constant 1; identifying data carried on labels)",
			},
			Labels: infoLabels,
		},
	}
}

// gpuMemoryEntries — /redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}
// plus per-memory MemoryMetrics.
func gpuMemoryEntries() []Entry {
	labels := []Label{
		gpuSystemIDLabel, // note: memory metrics use SystemID (not SystemName)
		gpuIDLabel,
		{Name: "memory_id", Field: "Id",
			Description: "GPU memory module identifier; matches the {memory_id} URL segment."},
	}
	// The health metric uses SystemName in commonLabels — but the code creates
	// two distinct label sets. Both use identical label *names*, so the same
	// []Label describes them.
	memPath := "/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}"
	metricsPath := memPath + "/MemoryMetrics"
	return []Entry{
		{
			Name:       "redfish_gpu_memory_ecc_correctable_total",
			Help:       "current correctable memory ecc errors reported on the gpu",
			MetricType: MetricCounter,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "LifeTime.CorrectableECCErrorCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_memory_ecc_uncorrectable_total",
			Help:       "current uncorrectable memory ecc errors reported on the gpu",
			MetricType: MetricCounter,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "LifeTime.UncorrectableECCErrorCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_memory_capacity_mib",
			Help:       "GPU memory capacity in MiB",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: memPath, Field: "CapacityMiB"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_memory_state",
			Help:       "GPU memory state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: memPath, Field: "Status.State"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_memory_health",
			Help:       "GPU memory health,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: memPath, Field: "Status.Health"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_memory_row_remapping_failed",
			Help:       "GPU memory row remapping failed status (1 if failed)",
			MetricType: MetricGauge,
			ValueType:  ValueBool,
			Value:      Source{Path: memPath, Field: "Oem.Nvidia.RowRemappingFailed"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_memory_row_remapping_pending",
			Help:       "GPU memory row remapping pending status (1 if pending)",
			MetricType: MetricGauge,
			ValueType:  ValueBool,
			Value:      Source{Path: memPath, Field: "Oem.Nvidia.RowRemappingPending"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_memory_correctable_row_remapping_count",
			Help:       "GPU memory correctable row remapping count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "Oem.Nvidia.RowRemapping.CorrectableRowRemappingCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_memory_uncorrectable_row_remapping_count",
			Help:       "GPU memory uncorrectable row remapping count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "Oem.Nvidia.RowRemapping.UncorrectableRowRemappingCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_memory_high_availability_bank_count",
			Help:       "GPU memory high availability bank count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "Oem.Nvidia.RowRemapping.HighAvailabilityBankCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_memory_low_availability_bank_count",
			Help:       "GPU memory low availability bank count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "Oem.Nvidia.RowRemapping.LowAvailabilityBankCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_memory_no_availability_bank_count",
			Help:       "GPU memory no availability bank count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "Oem.Nvidia.RowRemapping.NoAvailabilityBankCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_memory_partial_availability_bank_count",
			Help:       "GPU memory partial availability bank count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "Oem.Nvidia.RowRemapping.PartialAvailabilityBankCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_memory_max_availability_bank_count",
			Help:       "GPU memory max availability bank count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "Oem.Nvidia.RowRemapping.MaxAvailabilityBankCount"},
			Labels:     labels,
		},
	}
}

// gpuOEMEntries — ProcessorMetrics OEM data + accumulated context duration.
func gpuOEMEntries() []Entry {
	baseLabels := []Label{gpuSystemNameLabel, gpuIDLabel}
	metricsPath := "/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics"
	return []Entry{
		{
			Name:       "redfish_gpu_sram_ecc_error_threshold_exceeded",
			Help:       "GPU SRAM ECC error threshold exceeded (1 if exceeded)",
			MetricType: MetricGauge,
			ValueType:  ValueBool,
			Value:      Source{Path: metricsPath, Field: "Oem.Nvidia.SRAMECCErrorThresholdExceeded"},
			Labels:     baseLabels,
		},
		{
			Name:       "redfish_gpu_context_utilization_seconds_total",
			Help:       "Accumulated GPU context utilization duration in seconds",
			MetricType: MetricCounter,
			ValueType:  ValueDuration,
			Value:      Source{Path: metricsPath, Field: "Oem.Nvidia.AccumulatedGPUContextUtilizationDuration"},
			Labels:     baseLabels,
		},
	}
}

// gpuNVLinkEntries — /redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}
// filtered to PortProtocol == NVLink and Id containing "NVLink_".
func gpuNVLinkEntries() []Entry {
	labels := []Label{
		gpuSystemNameLabel,
		gpuIDLabel,
		{Name: "port_id", Field: "Id",
			Description: "NVLink port identifier (e.g. \"NVLink_0\")."},
		{Name: "port_type", Field: "PortType",
			Description: "Redfish port type enumeration."},
		{Name: "port_protocol", Field: "PortProtocol",
			Description: "Port protocol (constant \"NVLink\" after filtering)."},
	}
	portPath := "/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}"
	metricsPath := portPath + "/Metrics"
	return []Entry{
		{
			Name:       "redfish_gpu_nvlink_state",
			Help:       "NVLink port state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonState,
			Value:      Source{Path: portPath, Field: "Status.State"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_nvlink_health",
			Help:       "NVLink port health,1(OK),2(Warning),3(Critical)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumCommonHealth,
			Value:      Source{Path: portPath, Field: "Status.Health"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_nvlink_link_status",
			Help:       "NVLink port link status,1(LinkUp),2(Starting),3(Training),4(LinkDown),5(NoLink)",
			MetricType: MetricGauge,
			ValueType:  ValueIntEnum,
			Enum:       EnumNVLinkPortLinkStatus,
			Value:      Source{Path: portPath, Field: "LinkStatus"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_nvlink_runtime_error",
			Help:       "NVLink runtime error status (1 if error)",
			MetricType: MetricGauge,
			ValueType:  ValueBool,
			Value:      Source{Path: metricsPath, Field: "Oem.Nvidia.NVLinkErrors.RuntimeError"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_nvlink_training_error",
			Help:       "NVLink training error status (1 if error)",
			MetricType: MetricGauge,
			ValueType:  ValueBool,
			Value:      Source{Path: metricsPath, Field: "Oem.Nvidia.NVLinkErrors.TrainingError"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_nvlink_link_error_recovery_count",
			Help:       "NVLink error recovery count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "Oem.Nvidia.LinkErrorRecoveryCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_nvlink_link_downed_count",
			Help:       "NVLink link downed count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "Oem.Nvidia.LinkDownedCount"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_nvlink_symbol_errors",
			Help:       "NVLink symbol error count",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value:      Source{Path: metricsPath, Field: "Oem.Nvidia.SymbolErrors"},
			Labels:     labels,
		},
		{
			Name:       "redfish_gpu_nvlink_bit_error_rate",
			Help:       "NVLink bit error rate",
			MetricType: MetricGauge,
			ValueType:  ValueFloat,
			Value:      Source{Path: metricsPath, Field: "Oem.Nvidia.BitErrorRate"},
			Labels:     labels,
		},
	}
}

var gpuSystemIDLabel = Label{
	Name:        "system_id",
	Path:        "/redfish/v1/Systems/{system_id}",
	Field:       "Id",
	Description: "Parent system Redfish identifier.",
}

var gpuSystemNameLabel = Label{
	Name:        "system_id",
	Path:        "/redfish/v1/Systems/{system_id}",
	Field:       "Name",
	Description: "System *Name* (not Id) — the collector emits Name in the system_id label for most GPU metrics (except memory ECC counters which use Id).",
}

var gpuIDLabel = Label{
	Name:        "gpu_id",
	Field:       "Id",
	Description: "GPU processor Redfish identifier; matches the {gpu_id} URL segment.",
}
