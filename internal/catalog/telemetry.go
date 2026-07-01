package catalog

// Telemetry returns the catalog for the TelemetryService collector.
//
// See internal/collector/telemetry_collector.go. All metrics originate
// from MetricReports under /redfish/v1/TelemetryService/MetricReports/.
// The collector dispatches on the report ID (via strings.Contains) and
// then walks each report's MetricValues[*], parsing MetricProperty paths
// to extract {system_id, gpu_id, memory_id, port_id, cpu_id, sensor_id}
// as appropriate.
//
// Value.Path always points at the MetricReport that carries the value.
// Value.Field describes the MetricValues[*].MetricValue selected by
// matching MetricValues[*].MetricProperty against the substring shown.
// Report IDs use {n} because HGX firmware may expose *_0, *_1, etc.
func Telemetry() Module {
	return Module{
		Name: "telemetry",
		Entries: concat(
			telemetryProcessorEntries(),
			telemetryMemoryEntries(),
			telemetryResetEntries(),
			telemetryPortEntries(),
			telemetryGPMEntries(),
			telemetryPlatformEnvEntries(),
			telemetryMetaEntries(),
		),
	}
}

func telemetryProcessorEntries() []Entry {
	labels := []Label{telSystemIDLabel, telGPUIDLabel}
	path := "/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}"
	origPath := "/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics"
	mkField := func(prop string) string {
		return "MetricValues[?MetricProperty ~ '" + prop + "'].MetricValue"
	}
	cnt := func(name, help, prop string) Entry {
		return Entry{Name: name, Help: help, MetricType: MetricCounter, ValueType: ValueInt,
			Value: Source{Path: path, Field: mkField(prop), OriginalPath: origPath}, Labels: labels}
	}
	dur := func(name, help, prop string) Entry {
		return Entry{Name: name, Help: help, MetricType: MetricCounter, ValueType: ValueDuration,
			Value: Source{Path: path, Field: mkField(prop), OriginalPath: origPath}, Labels: labels}
	}
	return []Entry{
		cnt("redfish_telemetry_cache_ecc_correctable_total", "Total correctable ECC errors in GPU cache (L2/SRAM)", "CacheMetricsTotal/LifeTime/CorrectableECCErrorCount"),
		cnt("redfish_telemetry_cache_ecc_uncorrectable_total", "Total uncorrectable ECC errors in GPU cache (L2/SRAM)", "CacheMetricsTotal/LifeTime/UncorrectableECCErrorCount"),
		cnt("redfish_telemetry_pcie_correctable_errors_total", "Total PCIe correctable errors", "PCIeErrors/CorrectableErrorCount"),
		cnt("redfish_telemetry_pcie_nonfatal_errors_total", "Total PCIe non-fatal errors", "PCIeErrors/NonFatalErrorCount"),
		cnt("redfish_telemetry_pcie_fatal_errors_total", "Total PCIe fatal errors", "PCIeErrors/FatalErrorCount"),
		cnt("redfish_telemetry_pcie_l0_to_recovery_total", "Total PCIe L0 to recovery transitions", "PCIeErrors/L0ToRecoveryCount"),
		cnt("redfish_telemetry_pcie_replay_total", "Total PCIe replay events", "PCIeErrors/ReplayCount"),
		cnt("redfish_telemetry_pcie_replay_rollover_total", "Total PCIe replay rollover events", "PCIeErrors/ReplayRolloverCount"),
		cnt("redfish_telemetry_pcie_nak_sent_total", "Total PCIe NAK sent", "PCIeErrors/NAKSentCount"),
		cnt("redfish_telemetry_pcie_nak_received_total", "Total PCIe NAK received", "PCIeErrors/NAKReceivedCount"),
		cnt("redfish_telemetry_pcie_unsupported_request_total", "Total PCIe unsupported requests", "PCIeErrors/UnsupportedRequestCount"),
		dur("redfish_telemetry_power_throttle_duration_seconds_total", "Total time GPU was throttled due to power limits", "PowerLimitThrottleDuration"),
		dur("redfish_telemetry_thermal_throttle_duration_seconds_total", "Total time GPU was throttled due to thermal limits", "ThermalLimitThrottleDuration"),
		dur("redfish_telemetry_hardware_violation_throttle_duration_seconds_total", "Total time GPU was throttled due to hardware violations", "Oem/Nvidia/HardwareViolationThrottleDuration"),
		dur("redfish_telemetry_software_violation_throttle_duration_seconds_total", "Total time GPU was throttled due to software violations", "Oem/Nvidia/GlobalSoftwareViolationThrottleDuration"),
	}
}

func telemetryMemoryEntries() []Entry {
	labels := []Label{telSystemIDLabel, telGPUIDLabel, telMemoryIDLabel}
	path := "/redfish/v1/TelemetryService/MetricReports/HGX_MemoryMetrics_{n}"
	origPath := "/redfish/v1/Systems/{system_id}/Memory/{memory_id}/MemoryMetrics"
	f := func(prop string) string {
		return "MetricValues[?MetricProperty ~ '" + prop + "'].MetricValue"
	}
	return []Entry{
		{Name: "redfish_telemetry_memory_ecc_correctable_lifetime_total", Help: "Lifetime correctable DRAM ECC errors",
			MetricType: MetricCounter, ValueType: ValueInt,
			Value: Source{Path: path, Field: f("LifeTime/CorrectableECCErrorCount"), OriginalPath: origPath}, Labels: labels},
		{Name: "redfish_telemetry_memory_ecc_uncorrectable_lifetime_total", Help: "Lifetime uncorrectable DRAM ECC errors",
			MetricType: MetricCounter, ValueType: ValueInt,
			Value: Source{Path: path, Field: f("LifeTime/UncorrectableECCErrorCount"), OriginalPath: origPath}, Labels: labels},
		{Name: "redfish_telemetry_memory_bandwidth_percent", Help: "Memory bandwidth utilization percentage",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("BandwidthPercent"), OriginalPath: origPath}, Labels: labels},
		{Name: "redfish_telemetry_memory_capacity_utilization_percent", Help: "Memory capacity utilization percentage",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("CapacityUtilizationPercent"), OriginalPath: origPath}, Labels: labels},
		{Name: "redfish_telemetry_memory_operating_speed_mhz", Help: "Memory operating speed in MHz",
			MetricType: MetricGauge, ValueType: ValueInt,
			Value: Source{Path: path, Field: f("OperatingSpeedMHz"), OriginalPath: origPath}, Labels: labels},
	}
}

func telemetryResetEntries() []Entry {
	labels := []Label{telSystemIDLabel, telGPUIDLabel}
	path := "/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorResetMetrics_{n}"
	origPath := "/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Oem/Nvidia/ProcessorResetMetrics"
	f := func(prop string) string {
		return "MetricValues[?MetricProperty ~ '" + prop + "'].MetricValue"
	}
	return []Entry{
		{Name: "redfish_telemetry_conventional_reset_entry_total", Help: "Total conventional reset entry events",
			MetricType: MetricCounter, ValueType: ValueInt,
			Value: Source{Path: path, Field: f("ConventionalResetEntryCount"), OriginalPath: origPath}, Labels: labels},
		{Name: "redfish_telemetry_conventional_reset_exit_total", Help: "Total conventional reset exit events",
			MetricType: MetricCounter, ValueType: ValueInt,
			Value: Source{Path: path, Field: f("ConventionalResetExitCount"), OriginalPath: origPath}, Labels: labels},
		{Name: "redfish_telemetry_fundamental_reset_entry_total", Help: "Total fundamental reset entry events",
			MetricType: MetricCounter, ValueType: ValueInt,
			Value: Source{Path: path, Field: f("FundamentalResetEntryCount"), OriginalPath: origPath}, Labels: labels},
		{Name: "redfish_telemetry_fundamental_reset_exit_total", Help: "Total fundamental reset exit events",
			MetricType: MetricCounter, ValueType: ValueInt,
			Value: Source{Path: path, Field: f("FundamentalResetExitCount"), OriginalPath: origPath}, Labels: labels},
		{Name: "redfish_telemetry_irot_reset_exit_total", Help: "Total IRoT (Internal Root of Trust) reset exit events",
			MetricType: MetricCounter, ValueType: ValueInt,
			Value: Source{Path: path, Field: f("IRoTResetExitCount"), OriginalPath: origPath}, Labels: labels},
		{Name: "redfish_telemetry_pf_flr_reset_entry_total", Help: "Total PF FLR (Physical Function Function-Level Reset) entry events",
			MetricType: MetricCounter, ValueType: ValueInt,
			Value: Source{Path: path, Field: f("PF_FLR_ResetEntryCount"), OriginalPath: origPath}, Labels: labels},
		{Name: "redfish_telemetry_pf_flr_reset_exit_total", Help: "Total PF FLR (Physical Function Function-Level Reset) exit events",
			MetricType: MetricCounter, ValueType: ValueInt,
			Value: Source{Path: path, Field: f("PF_FLR_ResetExitCount"), OriginalPath: origPath}, Labels: labels},
		// Named with _info suffix but is a Gauge encoding a LastResetType enum, NOT an Info metric.
		{Name: "redfish_telemetry_last_reset_type_info", Help: "Last reset type (1=Conventional, 2=Fundamental, 3=IRoT, 4=PF_FLR)",
			MetricType: MetricGauge, ValueType: ValueIntEnum, Enum: EnumLastResetType,
			Value: Source{Path: path, Field: f("LastResetType"), OriginalPath: origPath}, Labels: labels},
	}
}

func telemetryPortEntries() []Entry {
	labels := []Label{telSystemIDLabel, telGPUIDLabel, telPortIDLabel}
	path := "/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}"
	origPath := "/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics"
	f := func(prop string) string {
		return "MetricValues[?MetricProperty ~ '" + prop + "'].MetricValue"
	}
	cnt := func(name, help, prop string) Entry {
		return Entry{Name: name, Help: help, MetricType: MetricCounter, ValueType: ValueInt,
			Value: Source{Path: path, Field: f(prop), OriginalPath: origPath}, Labels: labels}
	}
	return []Entry{
		{Name: "redfish_telemetry_port_current_speed_gbps", Help: "Current port link speed in Gbps",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("CurrentSpeedGbps"), OriginalPath: "/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}"}, Labels: labels},
		cnt("redfish_telemetry_port_rx_bytes_total", "Total bytes received on port", "RXBytes"),
		cnt("redfish_telemetry_port_tx_bytes_total", "Total bytes transmitted on port", "TXBytes"),
		cnt("redfish_telemetry_port_rx_errors_total", "Total receive errors on port", "RXErrors"),
		cnt("redfish_telemetry_port_rx_frames_total", "Total frames received on port", "Networking/RXFrames"),
		cnt("redfish_telemetry_port_tx_frames_total", "Total frames transmitted on port", "Networking/TXFrames"),
		cnt("redfish_telemetry_port_tx_discards_total", "Total transmit discards on port", "Networking/TXDiscards"),
		cnt("redfish_telemetry_port_nvidia_intentional_link_down_count_total", "Total intentional link down events (NVIDIA OEM)", "Oem/Nvidia/IntentionalLinkDownCount"),
		cnt("redfish_telemetry_port_nvidia_unintentional_link_down_count_total", "Total unintentional link down events (NVIDIA OEM)", "Oem/Nvidia/UnintentionalLinkDownCount"),
		{Name: "redfish_telemetry_port_nvidia_link_down_reason_code", Help: "Last link down reason code (NVIDIA OEM)",
			MetricType: MetricGauge, ValueType: ValueInt,
			Value: Source{Path: path, Field: f("Oem/Nvidia/LinkDownReasonCode"), OriginalPath: origPath}, Labels: labels},
		cnt("redfish_telemetry_port_nvidia_neighbor_mtu_discards_total", "Total neighbor MTU discards (NVIDIA OEM)", "Oem/Nvidia/NeighborMTUDiscards"),
		cnt("redfish_telemetry_port_nvidia_qp1_dropped_total", "Total QP1 packets dropped (NVIDIA OEM)", "Oem/Nvidia/QP1Dropped"),
		cnt("redfish_telemetry_port_nvidia_rx_remote_physical_errors_total", "Total RX remote physical errors (NVIDIA OEM)", "Oem/Nvidia/RXRemotePhysicalErrors"),
		cnt("redfish_telemetry_port_nvidia_rx_switch_relay_errors_total", "Total RX switch relay errors (NVIDIA OEM)", "Oem/Nvidia/RXSwitchRelayErrors"),
		cnt("redfish_telemetry_port_nvidia_vl15_dropped_total", "Total VL15 packets dropped (NVIDIA OEM)", "Oem/Nvidia/VL15Dropped"),
		cnt("redfish_telemetry_port_nvidia_rx_no_protocol_bytes_total", "Total RX bytes without protocol (NVIDIA OEM)", "Oem/Nvidia/RXNoProtocolBytes"),
		cnt("redfish_telemetry_port_nvidia_tx_no_protocol_bytes_total", "Total TX bytes without protocol (NVIDIA OEM)", "Oem/Nvidia/TXNoProtocolBytes"),
		cnt("redfish_telemetry_port_nvidia_vl15_tx_bytes_total", "Total VL15 bytes transmitted (NVIDIA OEM)", "Oem/Nvidia/VL15TXBytes"),
		cnt("redfish_telemetry_port_nvidia_vl15_tx_packets_total", "Total VL15 packets transmitted (NVIDIA OEM)", "Oem/Nvidia/VL15TXPackets"),
		{Name: "redfish_telemetry_port_nvidia_rx_width", Help: "Current receive link width (NVIDIA OEM)",
			MetricType: MetricGauge, ValueType: ValueInt,
			Value: Source{Path: path, Field: f("Oem/Nvidia/RXWidth"), OriginalPath: origPath}, Labels: labels},
		{Name: "redfish_telemetry_port_nvidia_tx_width", Help: "Current transmit link width (NVIDIA OEM)",
			MetricType: MetricGauge, ValueType: ValueInt,
			Value: Source{Path: path, Field: f("Oem/Nvidia/TXWidth"), OriginalPath: origPath}, Labels: labels},
		cnt("redfish_telemetry_port_nvidia_tx_wait_total", "Total TX wait time (NVIDIA OEM)", "Oem/Nvidia/TXWait"),
		{Name: "redfish_telemetry_port_nvidia_total_raw_ber", Help: "Total raw bit error rate (NVIDIA OEM)",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("Oem/Nvidia/TotalRawBER"), OriginalPath: origPath}, Labels: labels},
	}
}

func telemetryGPMEntries() []Entry {
	labels := []Label{telSystemIDLabel, telGPUIDLabel}
	instanceLabels := []Label{telSystemIDLabel, telGPUIDLabel, telInstanceIDLabel}
	path := "/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}"
	origPath := "/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics"
	f := func(prop string) string {
		return "MetricValues[?MetricProperty ~ '" + prop + "'].MetricValue"
	}
	pct := func(name, help, prop string) Entry {
		return Entry{Name: name, Help: help, MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f(prop), OriginalPath: origPath}, Labels: labels}
	}
	gbps := func(name, help, prop string) Entry {
		return Entry{Name: name, Help: help, MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f(prop), OriginalPath: origPath}, Labels: labels}
	}
	return []Entry{
		pct("redfish_telemetry_nvidia_tensor_core_activity_percent", "Tensor core activity percentage (NVIDIA GPM)", "TensorCoreActivityPercent"),
		pct("redfish_telemetry_nvidia_sm_activity_percent", "Streaming Multiprocessor activity percentage (NVIDIA GPM)", "SMActivityPercent"),
		pct("redfish_telemetry_nvidia_sm_occupancy_percent", "Streaming Multiprocessor occupancy percentage (NVIDIA GPM)", "SMOccupancyPercent"),
		pct("redfish_telemetry_nvidia_fp16_activity_percent", "FP16 floating point activity percentage (NVIDIA GPM)", "FP16ActivityPercent"),
		pct("redfish_telemetry_nvidia_fp32_activity_percent", "FP32 floating point activity percentage (NVIDIA GPM)", "FP32ActivityPercent"),
		pct("redfish_telemetry_nvidia_fp64_activity_percent", "FP64 floating point activity percentage (NVIDIA GPM)", "FP64ActivityPercent"),
		pct("redfish_telemetry_nvidia_integer_activity_percent", "Integer operation activity percentage (NVIDIA GPM)", "IntegerActivityUtilizationPercent"),
		pct("redfish_telemetry_nvidia_dmma_utilization_percent", "Double precision Matrix Multiply-Accumulate utilization (NVIDIA GPM)", "DMMAUtilizationPercent"),
		pct("redfish_telemetry_nvidia_hmma_utilization_percent", "Half precision Matrix Multiply-Accumulate utilization (NVIDIA GPM)", "HMMAUtilizationPercent"),
		pct("redfish_telemetry_nvidia_imma_utilization_percent", "Integer Matrix Multiply-Accumulate utilization (NVIDIA GPM)", "IMMAUtilizationPercent"),
		pct("redfish_telemetry_nvidia_graphics_engine_activity_percent", "Graphics engine activity percentage (NVIDIA GPM)", "GraphicsEngineActivityPercent"),
		pct("redfish_telemetry_nvidia_nvdec_utilization_percent", "Video decoder overall utilization (NVIDIA GPM)", "NVDecUtilizationPercent"),
		pct("redfish_telemetry_nvidia_nvjpg_utilization_percent", "JPEG decoder overall utilization (NVIDIA GPM)", "NVJpgUtilizationPercent"),
		{Name: "redfish_telemetry_nvidia_nvdec_instance_utilization_percent", Help: "Video decoder instance utilization (NVIDIA GPM)",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("NVDecInstanceUtilizationPercent"), OriginalPath: origPath}, Labels: instanceLabels},
		{Name: "redfish_telemetry_nvidia_nvjpg_instance_utilization_percent", Help: "JPEG decoder instance utilization (NVIDIA GPM)",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("NVJpgInstanceUtilizationPercent"), OriginalPath: origPath}, Labels: instanceLabels},
		gbps("redfish_telemetry_nvidia_nvlink_data_rx_bandwidth_gbps", "NVLink data receive bandwidth in Gbps (NVIDIA GPM)", "NVLinkDataRxBandwidthGbps"),
		gbps("redfish_telemetry_nvidia_nvlink_data_tx_bandwidth_gbps", "NVLink data transmit bandwidth in Gbps (NVIDIA GPM)", "NVLinkDataTxBandwidthGbps"),
		gbps("redfish_telemetry_nvidia_nvlink_raw_rx_bandwidth_gbps", "NVLink raw receive bandwidth in Gbps including overhead (NVIDIA GPM)", "NVLinkRawRxBandwidthGbps"),
		gbps("redfish_telemetry_nvidia_nvlink_raw_tx_bandwidth_gbps", "NVLink raw transmit bandwidth in Gbps including overhead (NVIDIA GPM)", "NVLinkRawTxBandwidthGbps"),
		pct("redfish_telemetry_nvidia_nvofa_utilization_percent", "NVIDIA Optimized Fabrics Adapter utilization (NVIDIA GPM)", "NVOfaUtilizationPercent"),
		gbps("redfish_telemetry_nvidia_pcie_raw_rx_bandwidth_gbps", "PCIe raw receive bandwidth in Gbps (NVIDIA GPM)", "PCIeRawRxBandwidthGbps"),
		gbps("redfish_telemetry_nvidia_pcie_raw_tx_bandwidth_gbps", "PCIe raw transmit bandwidth in Gbps (NVIDIA GPM)", "PCIeRawTxBandwidthGbps"),
	}
}

func telemetryPlatformEnvEntries() []Entry {
	// Multiple label shapes for this report; the collector dispatches on
	// sensor ID prefixes/infixes (see sensorPrefix* constants in the
	// collector) to pick which metric a MetricValue populates.
	path := "/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}"
	origPath := "/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}"
	f := func(prop string) string {
		return "MetricValues[?MetricProperty ~ '" + prop + "'].MetricValue (sensor id parsed to pick target metric)"
	}
	gpuLabels := []Label{telSystemIDLabel, telGPUIDLabel}
	gpuMemoryLabels := []Label{telSystemIDLabel, telGPUIDLabel, telMemoryIDLabel}
	chassisLabels := []Label{labelResource("chassis"), telChassisIDLabel}
	cpuLabels := []Label{telSystemIDLabel, telCPUIDLabel}
	ambientLabels := []Label{telSystemIDLabel, telLocationIDLabel, telSensorIDLabel}
	bmcLabels := []Label{telSystemIDLabel}

	return []Entry{
		// Backward-compatible metrics keep their original subsystems.
		{Name: "redfish_gpu_memory_power_watts", Help: "GPU memory (DRAM) power consumption in watts",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("HGX_GPU_SXM_*_DRAM_*_Power_*"), OriginalPath: origPath}, Labels: gpuMemoryLabels},
		{Name: "redfish_gpu_temperature_tlimit_celsius", Help: "GPU TLIMIT temperature headroom in Celsius",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("HGX_GPU_SXM_*_TempLimit_*"), OriginalPath: origPath}, Labels: gpuLabels},
		{Name: "redfish_chassis_gpu_total_power_watts", Help: "Total GPU power consumption for all GPUs in chassis in watts",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("HGX_Chassis_*_TotalGPU_Power"), OriginalPath: origPath}, Labels: chassisLabels},

		// telemetry_-prefixed GPU environment metrics.
		{Name: "redfish_telemetry_gpu_energy_joules_total", Help: "Total GPU energy consumption in joules",
			MetricType: MetricCounter, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("HGX_GPU_*_Energy_*"), OriginalPath: origPath}, Labels: gpuLabels},
		{Name: "redfish_telemetry_gpu_power_watts", Help: "GPU power consumption in watts",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("HGX_GPU_*_Power_*"), OriginalPath: origPath}, Labels: gpuLabels},
		{Name: "redfish_telemetry_gpu_temperature_celsius", Help: "GPU core temperature in Celsius",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("HGX_GPU_*_Temp_*"), OriginalPath: origPath}, Labels: gpuLabels},
		{Name: "redfish_telemetry_gpu_memory_temperature_celsius", Help: "GPU memory temperature in Celsius",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("HGX_GPU_*_DRAM_*_Temp_*"), OriginalPath: origPath}, Labels: gpuMemoryLabels},

		// CPU environment metrics.
		{Name: "redfish_telemetry_cpu_energy_joules_total", Help: "Total CPU energy consumption in joules",
			MetricType: MetricCounter, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("*_CPU_*_Energy_*"), OriginalPath: origPath}, Labels: cpuLabels},
		{Name: "redfish_telemetry_cpu_power_watts", Help: "CPU power consumption in watts",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("*_CPU_*_Power_*"), OriginalPath: origPath}, Labels: cpuLabels},
		{Name: "redfish_telemetry_cpu_vreg_cpu_power_watts", Help: "CPU voltage regulator power in watts",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("*_VREG_CPU_Power_* or *_Vreg_0_CpuPower_*"), OriginalPath: origPath}, Labels: cpuLabels},
		{Name: "redfish_telemetry_cpu_vreg_soc_power_watts", Help: "SoC voltage regulator power in watts",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("*_VREG_SOC_Power_* or *_Vreg_0_SocPower_*"), OriginalPath: origPath}, Labels: cpuLabels},
		{Name: "redfish_telemetry_cpu_temperature_average_celsius", Help: "Average CPU temperature in Celsius",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("*_TempAvg_*"), OriginalPath: origPath}, Labels: cpuLabels},
		{Name: "redfish_telemetry_cpu_temperature_limit_celsius", Help: "CPU temperature limit in Celsius",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("*_TempLimit_*"), OriginalPath: origPath}, Labels: cpuLabels},
		{Name: "redfish_telemetry_cpu_edp_current_limit_watts", Help: "CPU current EDP (Electrical Design Point) limit in watts",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("*_EDP_Current_Limit_* or *_EnforcedEDPc_*"), OriginalPath: origPath}, Labels: cpuLabels},
		{Name: "redfish_telemetry_cpu_edp_peak_limit_watts", Help: "CPU peak EDP (Electrical Design Point) limit in watts",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("*_EDP_Peak_Limit_* or *_EnforcedEDPp_*"), OriginalPath: origPath}, Labels: cpuLabels},

		// Ambient/BMC.
		{Name: "redfish_telemetry_ambient_inlet_temperature_celsius", Help: "Ambient inlet temperature in Celsius",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("*_Inlet_Temp_*"), OriginalPath: origPath}, Labels: ambientLabels},
		{Name: "redfish_telemetry_ambient_exhaust_temperature_celsius", Help: "Ambient exhaust temperature in Celsius",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("*_Exhaust_Temp_*"), OriginalPath: origPath}, Labels: ambientLabels},
		{Name: "redfish_telemetry_bmc_temperature_celsius", Help: "BMC temperature in Celsius",
			MetricType: MetricGauge, ValueType: ValueFloat,
			Value: Source{Path: path, Field: f("HGX_BMC_*_TEMP_* or HGX_BMC_*_Temp_*"), OriginalPath: origPath}, Labels: bmcLabels},
	}
}

func telemetryMetaEntries() []Entry {
	return []Entry{
		{
			Name:       "redfish_telemetry_collection_stale_reports_last",
			Help:       "Quantity of stale reports discovered on the last collection loop",
			MetricType: MetricGauge,
			ValueType:  ValueInt,
			Value: Source{
				Path:  "/redfish/v1/TelemetryService/MetricReports",
				Field: "(derived — count of stale report timestamps observed in the last collection)",
			},
			Labels: nil,
		},
	}
}

var (
	telSystemIDLabel = Label{
		Name:        "system_id",
		Field:       "MetricProperty (parsed from URL segment after /Systems/)",
		Description: "System identifier extracted from each MetricValue's MetricProperty path.",
	}
	telGPUIDLabel = Label{
		Name:        "gpu_id",
		Field:       "MetricProperty (parsed from URL segment after /Processors/)",
		Description: "GPU processor identifier extracted from each MetricValue's MetricProperty path.",
	}
	telMemoryIDLabel = Label{
		Name:        "memory_id",
		Field:       "MetricProperty (parsed from URL segment after /Memory/)",
		Description: "Memory module identifier extracted from each MetricValue's MetricProperty path.",
	}
	telPortIDLabel = Label{
		Name:        "port_id",
		Field:       "MetricProperty (parsed from URL segment after /Ports/)",
		Description: "Port identifier extracted from each MetricValue's MetricProperty path.",
	}
	telInstanceIDLabel = Label{
		Name:        "instance_id",
		Field:       "MetricProperty (parsed from trailing segment)",
		Description: "Per-instance identifier for NVDec/NVJpg utilization metrics.",
	}
	telCPUIDLabel = Label{
		Name:        "cpu_id",
		Field:       "sensor id (parsed CPU_N pattern)",
		Description: "CPU identifier parsed from the sensor id (e.g. CPU_0, CPU_1).",
	}
	telChassisIDLabel = Label{
		Name:        "chassis_id",
		Field:       "sensor id (parsed HGX_Chassis_N pattern)",
		Description: "Chassis identifier parsed from the sensor id.",
	}
	telLocationIDLabel = Label{
		Name:        "location_id",
		Field:       "sensor id (Inlet vs Exhaust position)",
		Description: "Physical sensor location identifier parsed from the sensor id.",
	}
	telSensorIDLabel = Label{
		Name:        "sensor_id",
		Field:       "MetricValues[*].MetricProperty (raw sensor id preserved)",
		Description: "Raw Redfish sensor identifier preserved for ambient metrics.",
	}
)
