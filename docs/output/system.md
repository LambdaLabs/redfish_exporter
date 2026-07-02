# Module: system

**48 metrics.**

Back to [overview](README.md).

### `redfish_system_state`

system state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Systems/{system_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "system" identifying the metric group; not sourced from Redfish. |
| `system_id` | *(same as value)* | `Id` | System Redfish identifier; matches the {system_id} URL segment. |

### `redfish_system_health_state`

system health,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "system" identifying the metric group; not sourced from Redfish. |
| `system_id` | *(same as value)* | `Id` | System Redfish identifier; matches the {system_id} URL segment. |

### `redfish_system_power_state`

system power state

**Type:** Gauge — **Value:** int\_enum (PowerState)

**Enum `PowerState`:**

| Code | Label |
|---|---|
| 1 | On |
| 2 | Off |
| 3 | PoweringOn |
| 4 | PoweringOff |

**Value source:** `/redfish/v1/Systems/{system_id}` → `PowerState`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "system" identifying the metric group; not sourced from Redfish. |
| `system_id` | *(same as value)* | `Id` | System Redfish identifier; matches the {system_id} URL segment. |

### `redfish_system_bios_info`

host BIOS version (info metric, always 1)

**Type:** Info — **Value:** constant

**Value source:** `/redfish/v1/Systems/{system_id}` → `(constant 1; identifying data carried on labels)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `bios_version` | *(same as value)* | `BiosVersion` | Host BIOS version string; empty BiosVersion values (e.g. HGX baseboards) are skipped. |
| `model` | *(same as value)* | `Model` | System model string as reported by the manufacturer. |

### `redfish_system_total_memory_state`

system overall memory state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Systems/{system_id}` → `MemorySummary.Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "system" identifying the metric group; not sourced from Redfish. |
| `system_id` | *(same as value)* | `Id` | System Redfish identifier; matches the {system_id} URL segment. |

### `redfish_system_total_memory_health_state`

system overall memory health,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}` → `MemorySummary.Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "system" identifying the metric group; not sourced from Redfish. |
| `system_id` | *(same as value)* | `Id` | System Redfish identifier; matches the {system_id} URL segment. |

### `redfish_system_total_memory_size`

system total memory size, GiB

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Systems/{system_id}` → `MemorySummary.TotalSystemMemoryGiB`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "system" identifying the metric group; not sourced from Redfish. |
| `system_id` | *(same as value)* | `Id` | System Redfish identifier; matches the {system_id} URL segment. |

### `redfish_system_total_processor_state`

system overall processor state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Systems/{system_id}` → `ProcessorSummary.Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "system" identifying the metric group; not sourced from Redfish. |
| `system_id` | *(same as value)* | `Id` | System Redfish identifier; matches the {system_id} URL segment. |

### `redfish_system_total_processor_health_state`

system overall processor health,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}` → `ProcessorSummary.Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "system" identifying the metric group; not sourced from Redfish. |
| `system_id` | *(same as value)* | `Id` | System Redfish identifier; matches the {system_id} URL segment. |

### `redfish_system_total_processor_count`

system total processor count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}` → `ProcessorSummary.Count`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "system" identifying the metric group; not sourced from Redfish. |
| `system_id` | *(same as value)* | `Id` | System Redfish identifier; matches the {system_id} URL segment. |

### `redfish_system_memory_state`

system memory state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Systems/{system_id}/Memory/{memory_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "memory" identifying the metric group; not sourced from Redfish. |
| `memory` | *(same as value)* | `Name` | Human-readable memory module name. |
| `memory_id` | *(same as value)* | `Id` | Memory module Redfish identifier; matches the {memory_id} URL segment. |

### `redfish_system_memory_health_state`

system memory health state,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}/Memory/{memory_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "memory" identifying the metric group; not sourced from Redfish. |
| `memory` | *(same as value)* | `Name` | Human-readable memory module name. |
| `memory_id` | *(same as value)* | `Id` | Memory module Redfish identifier; matches the {memory_id} URL segment. |

### `redfish_system_memory_capacity`

system memory capacity, MiB

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Systems/{system_id}/Memory/{memory_id}` → `CapacityMiB`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "memory" identifying the metric group; not sourced from Redfish. |
| `memory` | *(same as value)* | `Name` | Human-readable memory module name. |
| `memory_id` | *(same as value)* | `Id` | Memory module Redfish identifier; matches the {memory_id} URL segment. |

### `redfish_system_processor_state`

system processor state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{processor_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "processor" identifying the metric group; not sourced from Redfish. |
| `processor` | *(same as value)* | `Name` | Human-readable processor name. |
| `processor_id` | *(same as value)* | `Id` | Processor Redfish identifier; matches the {processor_id} URL segment. |

### `redfish_system_processor_health_state`

system processor health state,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{processor_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "processor" identifying the metric group; not sourced from Redfish. |
| `processor` | *(same as value)* | `Name` | Human-readable processor name. |
| `processor_id` | *(same as value)* | `Id` | Processor Redfish identifier; matches the {processor_id} URL segment. |

### `redfish_system_processor_health_rollup`

system processor health rollup,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{processor_id}` → `Status.HealthRollup`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "processor" identifying the metric group; not sourced from Redfish. |
| `processor` | *(same as value)* | `Name` | Human-readable processor name. |
| `processor_id` | *(same as value)* | `Id` | Processor Redfish identifier; matches the {processor_id} URL segment. |

### `redfish_system_processor_total_threads`

system processor total threads

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{processor_id}` → `TotalThreads`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "processor" identifying the metric group; not sourced from Redfish. |
| `processor` | *(same as value)* | `Name` | Human-readable processor name. |
| `processor_id` | *(same as value)* | `Id` | Processor Redfish identifier; matches the {processor_id} URL segment. |

### `redfish_system_processor_total_cores`

system processor total cores

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{processor_id}` → `TotalCores`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "processor" identifying the metric group; not sourced from Redfish. |
| `processor` | *(same as value)* | `Name` | Human-readable processor name. |
| `processor_id` | *(same as value)* | `Id` | Processor Redfish identifier; matches the {processor_id} URL segment. |

### `redfish_system_processor_pcie_errors_l0_to_recovery_count`

system processor PCIe L0 to recovery state transition count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{processor_id}/ProcessorMetrics` → `PCIeErrors.L0ToRecoveryCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "processor" identifying the metric group; not sourced from Redfish. |
| `processor` | *(same as value)* | `Name` | Human-readable processor name. |
| `processor_id` | *(same as value)* | `Id` | Processor Redfish identifier; matches the {processor_id} URL segment. |

### `redfish_system_processor_pcie_errors_correctable_count`

system processor PCIe correctable error count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{processor_id}/ProcessorMetrics` → `PCIeErrors.CorrectableErrorCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "processor" identifying the metric group; not sourced from Redfish. |
| `processor` | *(same as value)* | `Name` | Human-readable processor name. |
| `processor_id` | *(same as value)* | `Id` | Processor Redfish identifier; matches the {processor_id} URL segment. |

### `redfish_system_processor_pcie_errors_fatal_count`

system processor PCIe fatal error count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{processor_id}/ProcessorMetrics` → `PCIeErrors.FatalErrorCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "processor" identifying the metric group; not sourced from Redfish. |
| `processor` | *(same as value)* | `Name` | Human-readable processor name. |
| `processor_id` | *(same as value)* | `Id` | Processor Redfish identifier; matches the {processor_id} URL segment. |

### `redfish_system_processor_pcie_errors_non_fatal_count`

system processor PCIe non-fatal error count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{processor_id}/ProcessorMetrics` → `PCIeErrors.NonFatalErrorCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "processor" identifying the metric group; not sourced from Redfish. |
| `processor` | *(same as value)* | `Name` | Human-readable processor name. |
| `processor_id` | *(same as value)* | `Id` | Processor Redfish identifier; matches the {processor_id} URL segment. |

### `redfish_system_processor_pcie_errors_nak_received_count`

system processor PCIe NAK received count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{processor_id}/ProcessorMetrics` → `PCIeErrors.NAKReceivedCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "processor" identifying the metric group; not sourced from Redfish. |
| `processor` | *(same as value)* | `Name` | Human-readable processor name. |
| `processor_id` | *(same as value)* | `Id` | Processor Redfish identifier; matches the {processor_id} URL segment. |

### `redfish_system_processor_pcie_errors_nak_sent_count`

system processor PCIe NAK sent count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{processor_id}/ProcessorMetrics` → `PCIeErrors.NAKSentCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "processor" identifying the metric group; not sourced from Redfish. |
| `processor` | *(same as value)* | `Name` | Human-readable processor name. |
| `processor_id` | *(same as value)* | `Id` | Processor Redfish identifier; matches the {processor_id} URL segment. |

### `redfish_system_processor_pcie_errors_replay_count`

system processor PCIe replay count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{processor_id}/ProcessorMetrics` → `PCIeErrors.ReplayCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "processor" identifying the metric group; not sourced from Redfish. |
| `processor` | *(same as value)* | `Name` | Human-readable processor name. |
| `processor_id` | *(same as value)* | `Id` | Processor Redfish identifier; matches the {processor_id} URL segment. |

### `redfish_system_processor_pcie_errors_replay_rollover_count`

system processor PCIe replay rollover count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{processor_id}/ProcessorMetrics` → `PCIeErrors.ReplayRolloverCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "processor" identifying the metric group; not sourced from Redfish. |
| `processor` | *(same as value)* | `Name` | Human-readable processor name. |
| `processor_id` | *(same as value)* | `Id` | Processor Redfish identifier; matches the {processor_id} URL segment. |

### `redfish_system_processor_cache_lifetime_uncorrectable_ecc_error_count`

system processor cache lifetime uncorrectable ECC error count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{processor_id}/ProcessorMetrics` → `CacheMetricsTotal.LifeTime.UncorrectableECCErrorCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "processor" identifying the metric group; not sourced from Redfish. |
| `processor` | *(same as value)* | `Name` | Human-readable processor name. |
| `processor_id` | *(same as value)* | `Id` | Processor Redfish identifier; matches the {processor_id} URL segment. |

### `redfish_system_processor_cache_lifetime_correctable_ecc_error_count`

system processor cache lifetime correctable ECC error count

**Type:** Gauge — **Value:** int

**Value source:** `/redfish/v1/Systems/{system_id}/Processors/{processor_id}/ProcessorMetrics` → `CacheMetricsTotal.LifeTime.CorrectableECCErrorCount`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "processor" identifying the metric group; not sourced from Redfish. |
| `processor` | *(same as value)* | `Name` | Human-readable processor name. |
| `processor_id` | *(same as value)* | `Id` | Processor Redfish identifier; matches the {processor_id} URL segment. |

### `redfish_system_storage_volume_state`

system storage volume state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Systems/{system_id}/Storage/{storage_controller_id}/Volumes/{volume_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "volume" identifying the metric group; not sourced from Redfish. |
| `volume` | *(same as value)* | `Name` | Human-readable volume name. |
| `volume_id` | *(same as value)* | `Id` | Volume Redfish identifier. |

### `redfish_system_storage_volume_health_state`

system storage volume health state,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}/Storage/{storage_controller_id}/Volumes/{volume_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "volume" identifying the metric group; not sourced from Redfish. |
| `volume` | *(same as value)* | `Name` | Human-readable volume name. |
| `volume_id` | *(same as value)* | `Id` | Volume Redfish identifier. |

### `redfish_system_storage_volume_capacity`

system storage volume capacity, Bytes

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Systems/{system_id}/Storage/{storage_controller_id}/Volumes/{volume_id}` → `CapacityBytes`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "volume" identifying the metric group; not sourced from Redfish. |
| `volume` | *(same as value)* | `Name` | Human-readable volume name. |
| `volume_id` | *(same as value)* | `Id` | Volume Redfish identifier. |

### `redfish_system_storage_drive_state`

system storage drive state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Systems/{system_id}/Storage/{storage_controller_id}/Drives/{drive_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "drive" identifying the metric group; not sourced from Redfish. |
| `drive` | *(same as value)* | `Name` | Human-readable drive name. |
| `drive_id` | *(same as value)* | `Id` | Drive Redfish identifier. |
| `storage_controller_id` | `/redfish/v1/Systems/{system_id}/Storage/{storage_controller_id}` | `Id` | Parent Storage resource Id (the collector labels this as storage_controller_id). |

### `redfish_system_storage_drive_health_state`

system storage drive health state,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}/Storage/{storage_controller_id}/Drives/{drive_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "drive" identifying the metric group; not sourced from Redfish. |
| `drive` | *(same as value)* | `Name` | Human-readable drive name. |
| `drive_id` | *(same as value)* | `Id` | Drive Redfish identifier. |
| `storage_controller_id` | `/redfish/v1/Systems/{system_id}/Storage/{storage_controller_id}` | `Id` | Parent Storage resource Id (the collector labels this as storage_controller_id). |

### `redfish_system_storage_drive_capacity`

system storage drive capacity, Bytes

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Systems/{system_id}/Storage/{storage_controller_id}/Drives/{drive_id}` → `CapacityBytes`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "drive" identifying the metric group; not sourced from Redfish. |
| `drive` | *(same as value)* | `Name` | Human-readable drive name. |
| `drive_id` | *(same as value)* | `Id` | Drive Redfish identifier. |
| `storage_controller_id` | `/redfish/v1/Systems/{system_id}/Storage/{storage_controller_id}` | `Id` | Parent Storage resource Id (the collector labels this as storage_controller_id). |

### `redfish_system_storage_controller_state`

system storage controller state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Systems/{system_id}/Storage/{storage_controller_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "storage_controller" identifying the metric group; not sourced from Redfish. |
| `storage_controller` | *(same as value)* | `Name` | Human-readable storage controller name. |
| `storage_controller_id` | *(same as value)* | `Id` | Storage controller Redfish identifier. |

### `redfish_system_storage_controller_health_state`

system storage controller health state,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}/Storage/{storage_controller_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "storage_controller" identifying the metric group; not sourced from Redfish. |
| `storage_controller` | *(same as value)* | `Name` | Human-readable storage controller name. |
| `storage_controller_id` | *(same as value)* | `Id` | Storage controller Redfish identifier. |

### `redfish_system_pcie_device_state`

system pcie device state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Systems/{system_id}/PCIeDevices/{pcie_device_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "pcie_device" identifying the metric group; not sourced from Redfish. |
| `pcie_device` | *(same as value)* | `Name` | PCIe device human-readable name. |
| `pcie_device_id` | *(same as value)* | `Id` | PCIe device Redfish identifier. |
| `pcie_device_partnumber` | *(same as value)* | `PartNumber` | Manufacturer part number of the PCIe device. |
| `pcie_device_type` | *(same as value)* | `DeviceType` | PCIe device type enumeration. |
| `pcie_serial_number` | *(same as value)* | `SerialNumber` | Serial number of the PCIe device. |

### `redfish_system_pcie_device_health_state`

system pcie device health state,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}/PCIeDevices/{pcie_device_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "pcie_device" identifying the metric group; not sourced from Redfish. |
| `pcie_device` | *(same as value)* | `Name` | PCIe device human-readable name. |
| `pcie_device_id` | *(same as value)* | `Id` | PCIe device Redfish identifier. |
| `pcie_device_partnumber` | *(same as value)* | `PartNumber` | Manufacturer part number of the PCIe device. |
| `pcie_device_type` | *(same as value)* | `DeviceType` | PCIe device type enumeration. |
| `pcie_serial_number` | *(same as value)* | `SerialNumber` | Serial number of the PCIe device. |

### `redfish_system_pcie_function_state`

system pcie function state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Systems/{system_id}/PCIeFunctions/{pcie_function_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "pcie_function" identifying the metric group; not sourced from Redfish. |
| `pcie_function_name` | *(same as value)* | `Name` | PCIe function human-readable name. |
| `pcie_function_id` | *(same as value)* | `Id` | PCIe function Redfish identifier (stringified integer). |
| `pci_function_deviceclass` | *(same as value)* | `DeviceClass` | PCIe function device class enumeration. |
| `pci_function_type` | *(same as value)* | `FunctionType` | PCIe function type (e.g. Physical, Virtual). |

### `redfish_system_pcie_function_health_state`

system pcie device function state,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}/PCIeFunctions/{pcie_function_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "pcie_function" identifying the metric group; not sourced from Redfish. |
| `pcie_function_name` | *(same as value)* | `Name` | PCIe function human-readable name. |
| `pcie_function_id` | *(same as value)* | `Id` | PCIe function Redfish identifier (stringified integer). |
| `pci_function_deviceclass` | *(same as value)* | `DeviceClass` | PCIe function device class enumeration. |
| `pci_function_type` | *(same as value)* | `FunctionType` | PCIe function type (e.g. Physical, Virtual). |

### `redfish_system_network_interface_state`

system network interface state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Systems/{system_id}/NetworkInterfaces/{network_interface_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "network_interface" identifying the metric group; not sourced from Redfish. |
| `network_interface` | *(same as value)* | `Name` | Network interface human-readable name. |
| `network_interface_id` | *(same as value)* | `Id` | Network interface Redfish identifier. |

### `redfish_system_network_interface_health_state`

system network interface health state,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}/NetworkInterfaces/{network_interface_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "network_interface" identifying the metric group; not sourced from Redfish. |
| `network_interface` | *(same as value)* | `Name` | Network interface human-readable name. |
| `network_interface_id` | *(same as value)* | `Id` | Network interface Redfish identifier. |

### `redfish_system_ethernet_interface_state`

system ethernet interface state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Systems/{system_id}/EthernetInterfaces/{ethernet_interface_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "ethernet_interface" identifying the metric group; not sourced from Redfish. |
| `ethernet_interface` | *(same as value)* | `Name` | Ethernet interface human-readable name. |
| `ethernet_interface_id` | *(same as value)* | `Id` | Ethernet interface Redfish identifier. |
| `ethernet_interface_speed` | *(same as value)* | `SpeedMbps` | Interface speed formatted as "<N> Mbps". |

### `redfish_system_ethernet_interface_health_state`

system ethernet interface health state,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}/EthernetInterfaces/{ethernet_interface_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "ethernet_interface" identifying the metric group; not sourced from Redfish. |
| `ethernet_interface` | *(same as value)* | `Name` | Ethernet interface human-readable name. |
| `ethernet_interface_id` | *(same as value)* | `Id` | Ethernet interface Redfish identifier. |
| `ethernet_interface_speed` | *(same as value)* | `SpeedMbps` | Interface speed formatted as "<N> Mbps". |

### `redfish_system_ethernet_interface_link_status`

system ethernet interface link status,1(LinkUp),2(NoLink),3(LinkDown)

**Type:** Gauge — **Value:** int\_enum (EthernetLinkStatus)

**Enum `EthernetLinkStatus`:**

| Code | Label |
|---|---|
| 1 | LinkUp |
| 2 | NoLink |
| 3 | LinkDown |

**Value source:** `/redfish/v1/Systems/{system_id}/EthernetInterfaces/{ethernet_interface_id}` → `LinkStatus`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "ethernet_interface" identifying the metric group; not sourced from Redfish. |
| `ethernet_interface` | *(same as value)* | `Name` | Ethernet interface human-readable name. |
| `ethernet_interface_id` | *(same as value)* | `Id` | Ethernet interface Redfish identifier. |
| `ethernet_interface_speed` | *(same as value)* | `SpeedMbps` | Interface speed formatted as "<N> Mbps". |

### `redfish_system_ethernet_interface_link_enabled`

system ethernet interface if the link is enabled

**Type:** Gauge — **Value:** bool

**Value source:** `/redfish/v1/Systems/{system_id}/EthernetInterfaces/{ethernet_interface_id}` → `InterfaceEnabled`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "ethernet_interface" identifying the metric group; not sourced from Redfish. |
| `ethernet_interface` | *(same as value)* | `Name` | Ethernet interface human-readable name. |
| `ethernet_interface_id` | *(same as value)* | `Id` | Ethernet interface Redfish identifier. |
| `ethernet_interface_speed` | *(same as value)* | `SpeedMbps` | Interface speed formatted as "<N> Mbps". |

### `redfish_system_log_service_state`

system log service state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Systems/{system_id}/LogServices/{log_service_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system identifier this log service belongs to. |
| `log_service` | *(same as value)* | `Name` | Human-readable log service name. |
| `log_service_id` | *(same as value)* | `Id` | Log service Redfish identifier. |
| `log_service_enabled` | *(same as value)* | `ServiceEnabled` | Whether the log service is enabled. |
| `log_service_overwrite_policy` | *(same as value)* | `OverWritePolicy` | Ring-buffer overwrite policy. |

### `redfish_system_log_service_health_state`

system log service health state,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Systems/{system_id}/LogServices/{log_service_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `system_id` | `/redfish/v1/Systems/{system_id}` | `Id` | Parent system identifier this log service belongs to. |
| `log_service` | *(same as value)* | `Name` | Human-readable log service name. |
| `log_service_id` | *(same as value)* | `Id` | Log service Redfish identifier. |
| `log_service_enabled` | *(same as value)* | `ServiceEnabled` | Whether the log service is enabled. |
| `log_service_overwrite_policy` | *(same as value)* | `OverWritePolicy` | Ring-buffer overwrite policy. |

