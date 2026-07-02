# Module: telemetry

**92 metrics.**

Back to [overview](README.md).

### `redfish_telemetry_cache_ecc_correctable_total`

Total correctable ECC errors in GPU cache (L2/SRAM)

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}` → `MetricValues[?MetricProperty ~ 'CacheMetricsTotal/LifeTime/CorrectableECCErrorCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_cache_ecc_uncorrectable_total`

Total uncorrectable ECC errors in GPU cache (L2/SRAM)

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}` → `MetricValues[?MetricProperty ~ 'CacheMetricsTotal/LifeTime/UncorrectableECCErrorCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_pcie_correctable_errors_total`

Total PCIe correctable errors

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}` → `MetricValues[?MetricProperty ~ 'PCIeErrors/CorrectableErrorCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_pcie_nonfatal_errors_total`

Total PCIe non-fatal errors

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}` → `MetricValues[?MetricProperty ~ 'PCIeErrors/NonFatalErrorCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_pcie_fatal_errors_total`

Total PCIe fatal errors

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}` → `MetricValues[?MetricProperty ~ 'PCIeErrors/FatalErrorCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_pcie_l0_to_recovery_total`

Total PCIe L0 to recovery transitions

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}` → `MetricValues[?MetricProperty ~ 'PCIeErrors/L0ToRecoveryCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_pcie_replay_total`

Total PCIe replay events

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}` → `MetricValues[?MetricProperty ~ 'PCIeErrors/ReplayCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_pcie_replay_rollover_total`

Total PCIe replay rollover events

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}` → `MetricValues[?MetricProperty ~ 'PCIeErrors/ReplayRolloverCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_pcie_nak_sent_total`

Total PCIe NAK sent

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}` → `MetricValues[?MetricProperty ~ 'PCIeErrors/NAKSentCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_pcie_nak_received_total`

Total PCIe NAK received

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}` → `MetricValues[?MetricProperty ~ 'PCIeErrors/NAKReceivedCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_pcie_unsupported_request_total`

Total PCIe unsupported requests

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}` → `MetricValues[?MetricProperty ~ 'PCIeErrors/UnsupportedRequestCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_power_throttle_duration_seconds_total`

Total time GPU was throttled due to power limits

**Type:** Counter — **Value:** duration

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}` → `MetricValues[?MetricProperty ~ 'PowerLimitThrottleDuration'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_thermal_throttle_duration_seconds_total`

Total time GPU was throttled due to thermal limits

**Type:** Counter — **Value:** duration

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}` → `MetricValues[?MetricProperty ~ 'ThermalLimitThrottleDuration'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_hardware_violation_throttle_duration_seconds_total`

Total time GPU was throttled due to hardware violations

**Type:** Counter — **Value:** duration

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/HardwareViolationThrottleDuration'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_software_violation_throttle_duration_seconds_total`

Total time GPU was throttled due to software violations

**Type:** Counter — **Value:** duration

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/GlobalSoftwareViolationThrottleDuration'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_memory_ecc_correctable_lifetime_total`

Lifetime correctable DRAM ECC errors

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_MemoryMetrics_{n}` → `MetricValues[?MetricProperty ~ 'LifeTime/CorrectableECCErrorCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Memory/{memory_id}/MemoryMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `memory_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Memory/)` | Memory module identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_memory_ecc_uncorrectable_lifetime_total`

Lifetime uncorrectable DRAM ECC errors

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_MemoryMetrics_{n}` → `MetricValues[?MetricProperty ~ 'LifeTime/UncorrectableECCErrorCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Memory/{memory_id}/MemoryMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `memory_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Memory/)` | Memory module identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_memory_bandwidth_percent`

Memory bandwidth utilization percentage

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_MemoryMetrics_{n}` → `MetricValues[?MetricProperty ~ 'BandwidthPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Memory/{memory_id}/MemoryMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `memory_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Memory/)` | Memory module identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_memory_capacity_utilization_percent`

Memory capacity utilization percentage

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_MemoryMetrics_{n}` → `MetricValues[?MetricProperty ~ 'CapacityUtilizationPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Memory/{memory_id}/MemoryMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `memory_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Memory/)` | Memory module identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_memory_operating_speed_mhz`

Memory operating speed in MHz

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_MemoryMetrics_{n}` → `MetricValues[?MetricProperty ~ 'OperatingSpeedMHz'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Memory/{memory_id}/MemoryMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `memory_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Memory/)` | Memory module identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_conventional_reset_entry_total`

Total conventional reset entry events

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorResetMetrics_{n}` → `MetricValues[?MetricProperty ~ 'ConventionalResetEntryCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Oem/Nvidia/ProcessorResetMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_conventional_reset_exit_total`

Total conventional reset exit events

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorResetMetrics_{n}` → `MetricValues[?MetricProperty ~ 'ConventionalResetExitCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Oem/Nvidia/ProcessorResetMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_fundamental_reset_entry_total`

Total fundamental reset entry events

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorResetMetrics_{n}` → `MetricValues[?MetricProperty ~ 'FundamentalResetEntryCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Oem/Nvidia/ProcessorResetMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_fundamental_reset_exit_total`

Total fundamental reset exit events

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorResetMetrics_{n}` → `MetricValues[?MetricProperty ~ 'FundamentalResetExitCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Oem/Nvidia/ProcessorResetMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_irot_reset_exit_total`

Total IRoT (Internal Root of Trust) reset exit events

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorResetMetrics_{n}` → `MetricValues[?MetricProperty ~ 'IRoTResetExitCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Oem/Nvidia/ProcessorResetMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_pf_flr_reset_entry_total`

Total PF FLR (Physical Function Function-Level Reset) entry events

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorResetMetrics_{n}` → `MetricValues[?MetricProperty ~ 'PF_FLR_ResetEntryCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Oem/Nvidia/ProcessorResetMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_pf_flr_reset_exit_total`

Total PF FLR (Physical Function Function-Level Reset) exit events

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorResetMetrics_{n}` → `MetricValues[?MetricProperty ~ 'PF_FLR_ResetExitCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Oem/Nvidia/ProcessorResetMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_last_reset_type_info`

Last reset type (1=Conventional, 2=Fundamental, 3=IRoT, 4=PF_FLR)

**Type:** Gauge — **Value:** int\_enum (LastResetType)

**Enum `LastResetType`:**

| Code | Label |
|---|---|
| 1 | Conventional |
| 2 | Fundamental |
| 3 | IRoT |
| 4 | PF_FLR |

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorResetMetrics_{n}` → `MetricValues[?MetricProperty ~ 'LastResetType'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Oem/Nvidia/ProcessorResetMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_current_speed_gbps`

Current port link speed in Gbps

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'CurrentSpeedGbps'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_rx_bytes_total`

Total bytes received on port

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'RXBytes'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_tx_bytes_total`

Total bytes transmitted on port

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'TXBytes'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_rx_errors_total`

Total receive errors on port

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'RXErrors'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_rx_frames_total`

Total frames received on port

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Networking/RXFrames'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_tx_frames_total`

Total frames transmitted on port

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Networking/TXFrames'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_tx_discards_total`

Total transmit discards on port

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Networking/TXDiscards'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_intentional_link_down_count_total`

Total intentional link down events (NVIDIA OEM)

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/IntentionalLinkDownCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_unintentional_link_down_count_total`

Total unintentional link down events (NVIDIA OEM)

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/UnintentionalLinkDownCount'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_link_down_reason_code`

Last link down reason code (NVIDIA OEM)

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/LinkDownReasonCode'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_neighbor_mtu_discards_total`

Total neighbor MTU discards (NVIDIA OEM)

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/NeighborMTUDiscards'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_qp1_dropped_total`

Total QP1 packets dropped (NVIDIA OEM)

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/QP1Dropped'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_rx_remote_physical_errors_total`

Total RX remote physical errors (NVIDIA OEM)

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/RXRemotePhysicalErrors'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_rx_switch_relay_errors_total`

Total RX switch relay errors (NVIDIA OEM)

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/RXSwitchRelayErrors'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_vl15_dropped_total`

Total VL15 packets dropped (NVIDIA OEM)

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/VL15Dropped'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_rx_no_protocol_bytes_total`

Total RX bytes without protocol (NVIDIA OEM)

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/RXNoProtocolBytes'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_tx_no_protocol_bytes_total`

Total TX bytes without protocol (NVIDIA OEM)

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/TXNoProtocolBytes'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_vl15_tx_bytes_total`

Total VL15 bytes transmitted (NVIDIA OEM)

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/VL15TXBytes'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_vl15_tx_packets_total`

Total VL15 packets transmitted (NVIDIA OEM)

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/VL15TXPackets'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_rx_width`

Current receive link width (NVIDIA OEM)

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/RXWidth'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_tx_width`

Current transmit link width (NVIDIA OEM)

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/TXWidth'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_tx_wait_total`

Total TX wait time (NVIDIA OEM)

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/TXWait'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_port_nvidia_total_raw_ber`

Total raw bit error rate (NVIDIA OEM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorPortMetrics_{n}` → `MetricValues[?MetricProperty ~ 'Oem/Nvidia/TotalRawBER'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `port_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Ports/)` | Port identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_tensor_core_activity_percent`

Tensor core activity percentage (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'TensorCoreActivityPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_sm_activity_percent`

Streaming Multiprocessor activity percentage (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'SMActivityPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_sm_occupancy_percent`

Streaming Multiprocessor occupancy percentage (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'SMOccupancyPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_fp16_activity_percent`

FP16 floating point activity percentage (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'FP16ActivityPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_fp32_activity_percent`

FP32 floating point activity percentage (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'FP32ActivityPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_fp64_activity_percent`

FP64 floating point activity percentage (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'FP64ActivityPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_integer_activity_percent`

Integer operation activity percentage (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'IntegerActivityUtilizationPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_dmma_utilization_percent`

Double precision Matrix Multiply-Accumulate utilization (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'DMMAUtilizationPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_hmma_utilization_percent`

Half precision Matrix Multiply-Accumulate utilization (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'HMMAUtilizationPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_imma_utilization_percent`

Integer Matrix Multiply-Accumulate utilization (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'IMMAUtilizationPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_graphics_engine_activity_percent`

Graphics engine activity percentage (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'GraphicsEngineActivityPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_nvdec_utilization_percent`

Video decoder overall utilization (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'NVDecUtilizationPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_nvjpg_utilization_percent`

JPEG decoder overall utilization (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'NVJpgUtilizationPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_nvdec_instance_utilization_percent`

Video decoder instance utilization (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'NVDecInstanceUtilizationPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `instance_id` | *(same as value)* | `MetricProperty (parsed from trailing segment)` | Per-instance identifier for NVDec/NVJpg utilization metrics. |

### `redfish_telemetry_nvidia_nvjpg_instance_utilization_percent`

JPEG decoder instance utilization (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'NVJpgInstanceUtilizationPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `instance_id` | *(same as value)* | `MetricProperty (parsed from trailing segment)` | Per-instance identifier for NVDec/NVJpg utilization metrics. |

### `redfish_telemetry_nvidia_nvlink_data_rx_bandwidth_gbps`

NVLink data receive bandwidth in Gbps (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'NVLinkDataRxBandwidthGbps'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_nvlink_data_tx_bandwidth_gbps`

NVLink data transmit bandwidth in Gbps (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'NVLinkDataTxBandwidthGbps'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_nvlink_raw_rx_bandwidth_gbps`

NVLink raw receive bandwidth in Gbps including overhead (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'NVLinkRawRxBandwidthGbps'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_nvlink_raw_tx_bandwidth_gbps`

NVLink raw transmit bandwidth in Gbps including overhead (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'NVLinkRawTxBandwidthGbps'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_nvofa_utilization_percent`

NVIDIA Optimized Fabrics Adapter utilization (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'NVOfaUtilizationPercent'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_pcie_raw_rx_bandwidth_gbps`

PCIe raw receive bandwidth in Gbps (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'PCIeRawRxBandwidthGbps'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_nvidia_pcie_raw_tx_bandwidth_gbps`

PCIe raw transmit bandwidth in Gbps (NVIDIA GPM)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_ProcessorGPMMetrics_{n}` → `MetricValues[?MetricProperty ~ 'PCIeRawTxBandwidthGbps'].MetricValue`

**Original source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_gpu_memory_power_watts`

GPU memory (DRAM) power consumption in watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ 'HGX_GPU_SXM_*_DRAM_*_Power_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `memory_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Memory/)` | Memory module identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_gpu_temperature_tlimit_celsius`

GPU TLIMIT temperature headroom in Celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ 'HGX_GPU_SXM_*_TempLimit_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_chassis_gpu_total_power_watts`

Total GPU power consumption for all GPUs in chassis in watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ 'HGX_Chassis_*_TotalGPU_Power'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "chassis" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | *(same as value)* | `sensor id (parsed HGX_Chassis_N pattern)` | Chassis identifier parsed from the sensor id. |

### `redfish_telemetry_gpu_energy_joules_total`

Total GPU energy consumption in joules

**Type:** Counter — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ 'HGX_GPU_*_Energy_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_gpu_power_watts`

GPU power consumption in watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ 'HGX_GPU_*_Power_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_gpu_temperature_celsius`

GPU core temperature in Celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ 'HGX_GPU_*_Temp_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_gpu_memory_temperature_celsius`

GPU memory temperature in Celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ 'HGX_GPU_*_DRAM_*_Temp_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `gpu_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Processors/)` | GPU processor identifier extracted from each MetricValue's MetricProperty path. |
| `memory_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Memory/)` | Memory module identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_cpu_energy_joules_total`

Total CPU energy consumption in joules

**Type:** Counter — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ '*_CPU_*_Energy_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `cpu_id` | *(same as value)* | `sensor id (parsed CPU_N pattern)` | CPU identifier parsed from the sensor id (e.g. CPU_0, CPU_1). |

### `redfish_telemetry_cpu_power_watts`

CPU power consumption in watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ '*_CPU_*_Power_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `cpu_id` | *(same as value)* | `sensor id (parsed CPU_N pattern)` | CPU identifier parsed from the sensor id (e.g. CPU_0, CPU_1). |

### `redfish_telemetry_cpu_vreg_cpu_power_watts`

CPU voltage regulator power in watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ '*_VREG_CPU_Power_* or *_Vreg_0_CpuPower_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `cpu_id` | *(same as value)* | `sensor id (parsed CPU_N pattern)` | CPU identifier parsed from the sensor id (e.g. CPU_0, CPU_1). |

### `redfish_telemetry_cpu_vreg_soc_power_watts`

SoC voltage regulator power in watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ '*_VREG_SOC_Power_* or *_Vreg_0_SocPower_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `cpu_id` | *(same as value)* | `sensor id (parsed CPU_N pattern)` | CPU identifier parsed from the sensor id (e.g. CPU_0, CPU_1). |

### `redfish_telemetry_cpu_temperature_average_celsius`

Average CPU temperature in Celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ '*_TempAvg_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `cpu_id` | *(same as value)* | `sensor id (parsed CPU_N pattern)` | CPU identifier parsed from the sensor id (e.g. CPU_0, CPU_1). |

### `redfish_telemetry_cpu_temperature_limit_celsius`

CPU temperature limit in Celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ '*_TempLimit_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `cpu_id` | *(same as value)* | `sensor id (parsed CPU_N pattern)` | CPU identifier parsed from the sensor id (e.g. CPU_0, CPU_1). |

### `redfish_telemetry_cpu_edp_current_limit_watts`

CPU current EDP (Electrical Design Point) limit in watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ '*_EDP_Current_Limit_* or *_EnforcedEDPc_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `cpu_id` | *(same as value)* | `sensor id (parsed CPU_N pattern)` | CPU identifier parsed from the sensor id (e.g. CPU_0, CPU_1). |

### `redfish_telemetry_cpu_edp_peak_limit_watts`

CPU peak EDP (Electrical Design Point) limit in watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ '*_EDP_Peak_Limit_* or *_EnforcedEDPp_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `cpu_id` | *(same as value)* | `sensor id (parsed CPU_N pattern)` | CPU identifier parsed from the sensor id (e.g. CPU_0, CPU_1). |

### `redfish_telemetry_ambient_inlet_temperature_celsius`

Ambient inlet temperature in Celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ '*_Inlet_Temp_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `location_id` | *(same as value)* | `sensor id (Inlet vs Exhaust position)` | Physical sensor location identifier parsed from the sensor id. |
| `sensor_id` | *(same as value)* | `MetricValues[*].MetricProperty (raw sensor id preserved)` | Raw Redfish sensor identifier preserved for ambient metrics. |

### `redfish_telemetry_ambient_exhaust_temperature_celsius`

Ambient exhaust temperature in Celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ '*_Exhaust_Temp_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |
| `location_id` | *(same as value)* | `sensor id (Inlet vs Exhaust position)` | Physical sensor location identifier parsed from the sensor id. |
| `sensor_id` | *(same as value)* | `MetricValues[*].MetricProperty (raw sensor id preserved)` | Raw Redfish sensor identifier preserved for ambient metrics. |

### `redfish_telemetry_bmc_temperature_celsius`

BMC temperature in Celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/TelemetryService/MetricReports/HGX_PlatformEnvironmentMetrics_{n}` → `MetricValues[?MetricProperty ~ 'HGX_BMC_*_TEMP_* or HGX_BMC_*_Temp_*'].MetricValue (sensor id parsed to pick target metric)`

**Original source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | *(same as value)* | `MetricProperty (parsed from URL segment after /Systems/)` | System identifier extracted from each MetricValue's MetricProperty path. |

### `redfish_telemetry_collection_stale_reports_last`

Quantity of stale reports discovered on the last collection loop

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/TelemetryService/MetricReports` → `(derived — count of stale report timestamps observed in the last collection)`

**Labels:** none.

