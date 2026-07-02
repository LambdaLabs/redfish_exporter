# Module: manager

**5 metrics.**

Back to [overview](README.md).

### `redfish_manager_state`

manager state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Managers/{manager_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `manager_id` | *(same as value)* | `Id` | Redfish manager identifier; matches the {manager_id} URL segment. |
| `name` | *(same as value)* | `Name` | Human-readable manager name (e.g. "BMC"). |
| `model` | *(same as value)* | `Model` | Manager hardware model string as reported by the BMC vendor. |
| `type` | *(same as value)* | `ManagerType` | Redfish ManagerType enumeration (e.g. "BMC", "EnclosureManager"). |
| `firmware_version` | *(same as value)* | `FirmwareVersion` | Firmware version string of the manager. |

### `redfish_manager_health_state`

manager health,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Managers/{manager_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `manager_id` | *(same as value)* | `Id` | Redfish manager identifier; matches the {manager_id} URL segment. |
| `name` | *(same as value)* | `Name` | Human-readable manager name (e.g. "BMC"). |
| `model` | *(same as value)* | `Model` | Manager hardware model string as reported by the BMC vendor. |
| `type` | *(same as value)* | `ManagerType` | Redfish ManagerType enumeration (e.g. "BMC", "EnclosureManager"). |
| `firmware_version` | *(same as value)* | `FirmwareVersion` | Firmware version string of the manager. |

### `redfish_manager_power_state`

manager power state

**Type:** Gauge — **Value:** int\_enum (PowerState)

**Enum `PowerState`:**

| Code | Label |
|---|---|
| 1 | On |
| 2 | Off |
| 3 | PoweringOn |
| 4 | PoweringOff |

**Value source:** `/redfish/v1/Managers/{manager_id}` → `PowerState`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `manager_id` | *(same as value)* | `Id` | Redfish manager identifier; matches the {manager_id} URL segment. |
| `name` | *(same as value)* | `Name` | Human-readable manager name (e.g. "BMC"). |
| `model` | *(same as value)* | `Model` | Manager hardware model string as reported by the BMC vendor. |
| `type` | *(same as value)* | `ManagerType` | Redfish ManagerType enumeration (e.g. "BMC", "EnclosureManager"). |
| `firmware_version` | *(same as value)* | `FirmwareVersion` | Firmware version string of the manager. |

### `redfish_manager_log_service_state`

manager log service state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Managers/{manager_id}/LogServices/{log_service_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `manager_id` | `/redfish/v1/Managers/{manager_id}` | `Id` | Redfish manager identifier this log service belongs to. |
| `log_service` | *(same as value)* | `Name` | Human-readable log service name. |
| `log_service_id` | *(same as value)* | `Id` | Redfish log service identifier; matches the {log_service_id} URL segment. |
| `log_service_enabled` | *(same as value)* | `ServiceEnabled` | Whether the log service is enabled ("true"/"false" as string). |
| `log_service_overwrite_policy` | *(same as value)* | `OverWritePolicy` | Log ring-buffer overwrite policy (e.g. "WrapsWhenFull", "NeverOverwrites"). |

### `redfish_manager_log_service_health_state`

manager log service health state,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Managers/{manager_id}/LogServices/{log_service_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `manager_id` | `/redfish/v1/Managers/{manager_id}` | `Id` | Redfish manager identifier this log service belongs to. |
| `log_service` | *(same as value)* | `Name` | Human-readable log service name. |
| `log_service_id` | *(same as value)* | `Id` | Redfish log service identifier; matches the {log_service_id} URL segment. |
| `log_service_enabled` | *(same as value)* | `ServiceEnabled` | Whether the log service is enabled ("true"/"false" as string). |
| `log_service_overwrite_policy` | *(same as value)* | `OverWritePolicy` | Log ring-buffer overwrite policy (e.g. "WrapsWhenFull", "NeverOverwrites"). |

