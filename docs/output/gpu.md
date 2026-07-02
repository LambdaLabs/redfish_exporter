# Module: gpu

**28 metrics.**

Back to [overview](README.md).

### `redfish_gpu_state`

GPU processor state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

**Type:** Gauge — **Value:** int\_enum (CommonState)

**Enum `CommonState`:**

| Code | Label |
|---|---|
| 1 | Enabled |
| 2 | Disabled |
| 3 | StandbyOffline |
| 4 | StandbySpare |
| 5 | InTest |
| 6 | Starting |
| 7 | Absent |
| 8 | UnavailableOffline |
| 9 | Deferring |
| 10 | Quiesced |
| 11 | Updating |
| 12 | Standby |

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Name` | System *Name* (not Id) — the collector emits Name in the system_id label for most GPU metrics (except memory ECC counters which use Id). |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |

### `redfish_gpu_health`

GPU processor health,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Name` | System *Name* (not Id) — the collector emits Name in the system_id label for most GPU metrics (except memory ECC counters which use Id). |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |

### `redfish_gpu_info`

GPU information with serial number and UUID

**Type:** Info — **Value:** constant

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}` → `(constant 1; identifying data carried on labels)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Name` | System *Name* (not Id) — the collector emits Name in the system_id label for most GPU metrics (except memory ECC counters which use Id). |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `firmware_version` | *(same as value)* | `FirmwareVersion` | GPU firmware version string; "unknown" when absent. |
| `serial_number` | *(same as value)* | `SerialNumber` | GPU serial number; "unknown" when absent. |
| `uuid` | *(same as value)* | `UUID` | GPU UUID; "unknown" when absent. |

### `redfish_gpu_memory_ecc_correctable_total`

current correctable memory ecc errors reported on the gpu

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}/MemoryMetrics` → `LifeTime.CorrectableECCErrorCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system Redfish identifier. |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `memory_id` | *(same as value)* | `Id` | GPU memory module identifier; matches the {memory_id} URL segment. |

### `redfish_gpu_memory_ecc_uncorrectable_total`

current uncorrectable memory ecc errors reported on the gpu

**Type:** Counter — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}/MemoryMetrics` → `LifeTime.UncorrectableECCErrorCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system Redfish identifier. |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `memory_id` | *(same as value)* | `Id` | GPU memory module identifier; matches the {memory_id} URL segment. |

### `redfish_gpu_memory_capacity_mib`

GPU memory capacity in MiB

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}` → `CapacityMiB`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system Redfish identifier. |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `memory_id` | *(same as value)* | `Id` | GPU memory module identifier; matches the {memory_id} URL segment. |

### `redfish_gpu_memory_state`

GPU memory state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

**Type:** Gauge — **Value:** int\_enum (CommonState)

**Enum `CommonState`:**

| Code | Label |
|---|---|
| 1 | Enabled |
| 2 | Disabled |
| 3 | StandbyOffline |
| 4 | StandbySpare |
| 5 | InTest |
| 6 | Starting |
| 7 | Absent |
| 8 | UnavailableOffline |
| 9 | Deferring |
| 10 | Quiesced |
| 11 | Updating |
| 12 | Standby |

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system Redfish identifier. |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `memory_id` | *(same as value)* | `Id` | GPU memory module identifier; matches the {memory_id} URL segment. |

### `redfish_gpu_memory_health`

GPU memory health,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system Redfish identifier. |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `memory_id` | *(same as value)* | `Id` | GPU memory module identifier; matches the {memory_id} URL segment. |

### `redfish_gpu_memory_row_remapping_failed`

GPU memory row remapping failed status (1 if failed)

**Type:** Gauge — **Value:** bool

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}` → `Oem.Nvidia.RowRemappingFailed`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system Redfish identifier. |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `memory_id` | *(same as value)* | `Id` | GPU memory module identifier; matches the {memory_id} URL segment. |

### `redfish_gpu_memory_row_remapping_pending`

GPU memory row remapping pending status (1 if pending)

**Type:** Gauge — **Value:** bool

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}` → `Oem.Nvidia.RowRemappingPending`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system Redfish identifier. |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `memory_id` | *(same as value)* | `Id` | GPU memory module identifier; matches the {memory_id} URL segment. |

### `redfish_gpu_memory_correctable_row_remapping_count`

GPU memory correctable row remapping count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}/MemoryMetrics` → `Oem.Nvidia.RowRemapping.CorrectableRowRemappingCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system Redfish identifier. |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `memory_id` | *(same as value)* | `Id` | GPU memory module identifier; matches the {memory_id} URL segment. |

### `redfish_gpu_memory_uncorrectable_row_remapping_count`

GPU memory uncorrectable row remapping count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}/MemoryMetrics` → `Oem.Nvidia.RowRemapping.UncorrectableRowRemappingCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system Redfish identifier. |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `memory_id` | *(same as value)* | `Id` | GPU memory module identifier; matches the {memory_id} URL segment. |

### `redfish_gpu_memory_high_availability_bank_count`

GPU memory high availability bank count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}/MemoryMetrics` → `Oem.Nvidia.RowRemapping.HighAvailabilityBankCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system Redfish identifier. |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `memory_id` | *(same as value)* | `Id` | GPU memory module identifier; matches the {memory_id} URL segment. |

### `redfish_gpu_memory_low_availability_bank_count`

GPU memory low availability bank count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}/MemoryMetrics` → `Oem.Nvidia.RowRemapping.LowAvailabilityBankCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system Redfish identifier. |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `memory_id` | *(same as value)* | `Id` | GPU memory module identifier; matches the {memory_id} URL segment. |

### `redfish_gpu_memory_no_availability_bank_count`

GPU memory no availability bank count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}/MemoryMetrics` → `Oem.Nvidia.RowRemapping.NoAvailabilityBankCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system Redfish identifier. |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `memory_id` | *(same as value)* | `Id` | GPU memory module identifier; matches the {memory_id} URL segment. |

### `redfish_gpu_memory_partial_availability_bank_count`

GPU memory partial availability bank count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}/MemoryMetrics` → `Oem.Nvidia.RowRemapping.PartialAvailabilityBankCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system Redfish identifier. |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `memory_id` | *(same as value)* | `Id` | GPU memory module identifier; matches the {memory_id} URL segment. |

### `redfish_gpu_memory_max_availability_bank_count`

GPU memory max availability bank count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Memory/{memory_id}/MemoryMetrics` → `Oem.Nvidia.RowRemapping.MaxAvailabilityBankCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system Redfish identifier. |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `memory_id` | *(same as value)* | `Id` | GPU memory module identifier; matches the {memory_id} URL segment. |

### `redfish_gpu_sram_ecc_error_threshold_exceeded`

GPU SRAM ECC error threshold exceeded (1 if exceeded)

**Type:** Gauge — **Value:** bool

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics` → `Oem.Nvidia.SRAMECCErrorThresholdExceeded`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Name` | System *Name* (not Id) — the collector emits Name in the system_id label for most GPU metrics (except memory ECC counters which use Id). |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |

### `redfish_gpu_context_utilization_seconds_total`

Accumulated GPU context utilization duration in seconds

**Type:** Counter — **Value:** duration

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/ProcessorMetrics` → `Oem.Nvidia.AccumulatedGPUContextUtilizationDuration`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Name` | System *Name* (not Id) — the collector emits Name in the system_id label for most GPU metrics (except memory ECC counters which use Id). |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |

### `redfish_gpu_nvlink_state`

NVLink port state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

**Type:** Gauge — **Value:** int\_enum (CommonState)

**Enum `CommonState`:**

| Code | Label |
|---|---|
| 1 | Enabled |
| 2 | Disabled |
| 3 | StandbyOffline |
| 4 | StandbySpare |
| 5 | InTest |
| 6 | Starting |
| 7 | Absent |
| 8 | UnavailableOffline |
| 9 | Deferring |
| 10 | Quiesced |
| 11 | Updating |
| 12 | Standby |

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Name` | System *Name* (not Id) — the collector emits Name in the system_id label for most GPU metrics (except memory ECC counters which use Id). |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `port_id` | *(same as value)* | `Id` | NVLink port identifier (e.g. "NVLink_0"). |
| `port_type` | *(same as value)* | `PortType` | Redfish port type enumeration. |
| `port_protocol` | *(same as value)* | `PortProtocol` | Port protocol (constant "NVLink" after filtering). |

### `redfish_gpu_nvlink_health`

NVLink port health,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Name` | System *Name* (not Id) — the collector emits Name in the system_id label for most GPU metrics (except memory ECC counters which use Id). |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `port_id` | *(same as value)* | `Id` | NVLink port identifier (e.g. "NVLink_0"). |
| `port_type` | *(same as value)* | `PortType` | Redfish port type enumeration. |
| `port_protocol` | *(same as value)* | `PortProtocol` | Port protocol (constant "NVLink" after filtering). |

### `redfish_gpu_nvlink_link_status`

NVLink port link status,1(LinkUp),2(Starting),3(Training),4(LinkDown),5(NoLink)

**Type:** Gauge — **Value:** int\_enum (NVLinkPortLinkStatus)

**Enum `NVLinkPortLinkStatus`:**

| Code | Label |
|---|---|
| 1 | LinkUp |
| 2 | Starting |
| 3 | Training |
| 4 | LinkDown |
| 5 | NoLink |

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}` → `LinkStatus`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Name` | System *Name* (not Id) — the collector emits Name in the system_id label for most GPU metrics (except memory ECC counters which use Id). |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `port_id` | *(same as value)* | `Id` | NVLink port identifier (e.g. "NVLink_0"). |
| `port_type` | *(same as value)* | `PortType` | Redfish port type enumeration. |
| `port_protocol` | *(same as value)* | `PortProtocol` | Port protocol (constant "NVLink" after filtering). |

### `redfish_gpu_nvlink_runtime_error`

NVLink runtime error status (1 if error)

**Type:** Gauge — **Value:** bool

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics` → `Oem.Nvidia.NVLinkErrors.RuntimeError`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Name` | System *Name* (not Id) — the collector emits Name in the system_id label for most GPU metrics (except memory ECC counters which use Id). |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `port_id` | *(same as value)* | `Id` | NVLink port identifier (e.g. "NVLink_0"). |
| `port_type` | *(same as value)* | `PortType` | Redfish port type enumeration. |
| `port_protocol` | *(same as value)* | `PortProtocol` | Port protocol (constant "NVLink" after filtering). |

### `redfish_gpu_nvlink_training_error`

NVLink training error status (1 if error)

**Type:** Gauge — **Value:** bool

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics` → `Oem.Nvidia.NVLinkErrors.TrainingError`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Name` | System *Name* (not Id) — the collector emits Name in the system_id label for most GPU metrics (except memory ECC counters which use Id). |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `port_id` | *(same as value)* | `Id` | NVLink port identifier (e.g. "NVLink_0"). |
| `port_type` | *(same as value)* | `PortType` | Redfish port type enumeration. |
| `port_protocol` | *(same as value)* | `PortProtocol` | Port protocol (constant "NVLink" after filtering). |

### `redfish_gpu_nvlink_link_error_recovery_count`

NVLink error recovery count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics` → `Oem.Nvidia.LinkErrorRecoveryCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Name` | System *Name* (not Id) — the collector emits Name in the system_id label for most GPU metrics (except memory ECC counters which use Id). |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `port_id` | *(same as value)* | `Id` | NVLink port identifier (e.g. "NVLink_0"). |
| `port_type` | *(same as value)* | `PortType` | Redfish port type enumeration. |
| `port_protocol` | *(same as value)* | `PortProtocol` | Port protocol (constant "NVLink" after filtering). |

### `redfish_gpu_nvlink_link_downed_count`

NVLink link downed count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics` → `Oem.Nvidia.LinkDownedCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Name` | System *Name* (not Id) — the collector emits Name in the system_id label for most GPU metrics (except memory ECC counters which use Id). |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `port_id` | *(same as value)* | `Id` | NVLink port identifier (e.g. "NVLink_0"). |
| `port_type` | *(same as value)* | `PortType` | Redfish port type enumeration. |
| `port_protocol` | *(same as value)* | `PortProtocol` | Port protocol (constant "NVLink" after filtering). |

### `redfish_gpu_nvlink_symbol_errors`

NVLink symbol error count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics` → `Oem.Nvidia.SymbolErrors`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Name` | System *Name* (not Id) — the collector emits Name in the system_id label for most GPU metrics (except memory ECC counters which use Id). |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `port_id` | *(same as value)* | `Id` | NVLink port identifier (e.g. "NVLink_0"). |
| `port_type` | *(same as value)* | `PortType` | Redfish port type enumeration. |
| `port_protocol` | *(same as value)* | `PortProtocol` | Port protocol (constant "NVLink" after filtering). |

### `redfish_gpu_nvlink_bit_error_rate`

NVLink bit error rate

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{gpu_id}/Ports/{port_id}/Metrics` → `Oem.Nvidia.BitErrorRate`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Name` | System *Name* (not Id) — the collector emits Name in the system_id label for most GPU metrics (except memory ECC counters which use Id). |
| `gpu_id` | *(same as value)* | `Id` | GPU processor Redfish identifier; matches the {gpu_id} URL segment. |
| `port_id` | *(same as value)* | `Id` | NVLink port identifier (e.g. "NVLink_0"). |
| `port_type` | *(same as value)* | `PortType` | Redfish port type enumeration. |
| `port_protocol` | *(same as value)* | `PortProtocol` | Port protocol (constant "NVLink" after filtering). |

