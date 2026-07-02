# Module: chassis

**38 metrics.**

Back to [overview](README.md).

### `redfish_chassis_health`

health of chassis,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Chassis/{chassis_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "chassis" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | *(same as value)* | `Id` | Chassis Redfish identifier; matches the {chassis_id} URL segment. |

### `redfish_chassis_health_rollup`

health rollup of chassis,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Chassis/{chassis_id}` → `Status.HealthRollup`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "chassis" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | *(same as value)* | `Id` | Chassis Redfish identifier; matches the {chassis_id} URL segment. |

### `redfish_chassis_state`

state of chassis,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Chassis/{chassis_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "chassis" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | *(same as value)* | `Id` | Chassis Redfish identifier; matches the {chassis_id} URL segment. |

### `redfish_chassis_model_info`

organization responsible for producing the chassis, the name by which the manufacturer generally refers to the chassis, and a part number and sku assigned by the organization that is responsible for producing or manufacturing the chassis

**Type:** Info — **Value:** constant

**Value source:** `/redfish/v1/Chassis/{chassis_id}` → `(constant 1; identifying data carried on labels)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "chassis" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | *(same as value)* | `Id` | Chassis Redfish identifier; matches the {chassis_id} URL segment. |
| `manufacturer` | *(same as value)* | `Manufacturer` | Chassis manufacturer name as reported by Redfish. |
| `model` | *(same as value)* | `Model` | Chassis model string as reported by the manufacturer. |
| `part_number` | *(same as value)* | `PartNumber` | Manufacturer-assigned chassis part number. |
| `sku` | *(same as value)* | `SKU` | Manufacturer-assigned chassis SKU. |

### `redfish_chassis_temperature_sensor_state`

status state of temperature on this chassis component,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Thermal` → `Temperatures[*].Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "temperature" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `sensor` | *(same as value)* | `Temperatures[*].Name` | Human-readable temperature sensor name (e.g. "CPU1 Temp"). |
| `sensor_id` | *(same as value)* | `Temperatures[*].MemberId` | Stable per-chassis temperature sensor identifier. |

### `redfish_chassis_temperature_sensor_health`

status health of temperature on this chassis component,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Thermal` → `Temperatures[*].Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "temperature" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `sensor` | *(same as value)* | `Temperatures[*].Name` | Human-readable temperature sensor name (e.g. "CPU1 Temp"). |
| `sensor_id` | *(same as value)* | `Temperatures[*].MemberId` | Stable per-chassis temperature sensor identifier. |

### `redfish_chassis_temperature_celsius`

celsius of temperature on this chassis component

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Thermal` → `Temperatures[*].ReadingCelsius`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "temperature" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `sensor` | *(same as value)* | `Temperatures[*].Name` | Human-readable temperature sensor name (e.g. "CPU1 Temp"). |
| `sensor_id` | *(same as value)* | `Temperatures[*].MemberId` | Stable per-chassis temperature sensor identifier. |

### `redfish_chassis_fan_health`

fan health on this chassis component,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Thermal` → `Fans[*].Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "fan" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `fan` | *(same as value)* | `Fans[*].Name` | Human-readable fan name. |
| `fan_id` | *(same as value)* | `Fans[*].MemberId` | Stable per-chassis fan identifier. |
| `fan_unit` | *(same as value)* | `Fans[*].ReadingUnits` | Fan reading unit lowercased (e.g. "rpm", "percent"). |

### `redfish_chassis_fan_state`

fan state on this chassis component,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Thermal` → `Fans[*].Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "fan" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `fan` | *(same as value)* | `Fans[*].Name` | Human-readable fan name. |
| `fan_id` | *(same as value)* | `Fans[*].MemberId` | Stable per-chassis fan identifier. |
| `fan_unit` | *(same as value)* | `Fans[*].ReadingUnits` | Fan reading unit lowercased (e.g. "rpm", "percent"). |

### `redfish_chassis_fan_rpm`

fan RPM or percentage on this chassis component

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Thermal` → `Fans[*].Reading`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "fan" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `fan` | *(same as value)* | `Fans[*].Name` | Human-readable fan name. |
| `fan_id` | *(same as value)* | `Fans[*].MemberId` | Stable per-chassis fan identifier. |
| `fan_unit` | *(same as value)* | `Fans[*].ReadingUnits` | Fan reading unit lowercased (e.g. "rpm", "percent"). |

### `redfish_chassis_fan_rpm_percentage`

fan RPM, as a percentage of the min-max RPMs possible, on this chassis component

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Thermal` → `(derived from Fans[*].Reading, MinReadingRange, MaxReadingRange, thresholds)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "fan" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `fan` | *(same as value)* | `Fans[*].Name` | Human-readable fan name. |
| `fan_id` | *(same as value)* | `Fans[*].MemberId` | Stable per-chassis fan identifier. |
| `fan_unit` | *(same as value)* | `Fans[*].ReadingUnits` | Fan reading unit lowercased (e.g. "rpm", "percent"). |

### `redfish_chassis_fan_rpm_min`

lowest possible fan RPM or percentage, on this chassis component

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Thermal` → `Fans[*].MinReadingRange`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "fan" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `fan` | *(same as value)* | `Fans[*].Name` | Human-readable fan name. |
| `fan_id` | *(same as value)* | `Fans[*].MemberId` | Stable per-chassis fan identifier. |
| `fan_unit` | *(same as value)* | `Fans[*].ReadingUnits` | Fan reading unit lowercased (e.g. "rpm", "percent"). |

### `redfish_chassis_fan_rpm_max`

highest possible fan RPM or percentage, on this chassis component

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Thermal` → `Fans[*].MaxReadingRange`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "fan" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `fan` | *(same as value)* | `Fans[*].Name` | Human-readable fan name. |
| `fan_id` | *(same as value)* | `Fans[*].MemberId` | Stable per-chassis fan identifier. |
| `fan_unit` | *(same as value)* | `Fans[*].ReadingUnits` | Fan reading unit lowercased (e.g. "rpm", "percent"). |

### `redfish_chassis_fan_rpm_lower_threshold_critical`

threshold below the normal range fan RPM or percentage, but not fatal, on this chassis component

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Thermal` → `Fans[*].LowerThresholdCritical`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "fan" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `fan` | *(same as value)* | `Fans[*].Name` | Human-readable fan name. |
| `fan_id` | *(same as value)* | `Fans[*].MemberId` | Stable per-chassis fan identifier. |
| `fan_unit` | *(same as value)* | `Fans[*].ReadingUnits` | Fan reading unit lowercased (e.g. "rpm", "percent"). |

### `redfish_chassis_fan_rpm_lower_threshold_non_critical`

threshold below the normal range fan RPM or percentage, but not critical, on this chassis component

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Thermal` → `Fans[*].LowerThresholdNonCritical`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "fan" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `fan` | *(same as value)* | `Fans[*].Name` | Human-readable fan name. |
| `fan_id` | *(same as value)* | `Fans[*].MemberId` | Stable per-chassis fan identifier. |
| `fan_unit` | *(same as value)* | `Fans[*].ReadingUnits` | Fan reading unit lowercased (e.g. "rpm", "percent"). |

### `redfish_chassis_fan_rpm_lower_threshold_fatal`

threshold below the normal range fan RPM or percentage, and is fatal, on this chassis component

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Thermal` → `Fans[*].LowerThresholdFatal`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "fan" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `fan` | *(same as value)* | `Fans[*].Name` | Human-readable fan name. |
| `fan_id` | *(same as value)* | `Fans[*].MemberId` | Stable per-chassis fan identifier. |
| `fan_unit` | *(same as value)* | `Fans[*].ReadingUnits` | Fan reading unit lowercased (e.g. "rpm", "percent"). |

### `redfish_chassis_fan_rpm_upper_threshold_critical`

threshold above the normal range fan RPM or percentage, but not fatal, on this chassis component

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Thermal` → `Fans[*].UpperThresholdCritical`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "fan" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `fan` | *(same as value)* | `Fans[*].Name` | Human-readable fan name. |
| `fan_id` | *(same as value)* | `Fans[*].MemberId` | Stable per-chassis fan identifier. |
| `fan_unit` | *(same as value)* | `Fans[*].ReadingUnits` | Fan reading unit lowercased (e.g. "rpm", "percent"). |

### `redfish_chassis_fan_rpm_upper_threshold_non_critical`

threshold above the normal range fan RPM or percentage, but not critical, on this chassis component

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Thermal` → `Fans[*].UpperThresholdNonCritical`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "fan" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `fan` | *(same as value)* | `Fans[*].Name` | Human-readable fan name. |
| `fan_id` | *(same as value)* | `Fans[*].MemberId` | Stable per-chassis fan identifier. |
| `fan_unit` | *(same as value)* | `Fans[*].ReadingUnits` | Fan reading unit lowercased (e.g. "rpm", "percent"). |

### `redfish_chassis_fan_rpm_upper_threshold_fatal`

threshold above the normal range fan RPM or percentage, and is fatal, on this chassis component

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Thermal` → `Fans[*].UpperThresholdFatal`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "fan" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `fan` | *(same as value)* | `Fans[*].Name` | Human-readable fan name. |
| `fan_id` | *(same as value)* | `Fans[*].MemberId` | Stable per-chassis fan identifier. |
| `fan_unit` | *(same as value)* | `Fans[*].ReadingUnits` | Fan reading unit lowercased (e.g. "rpm", "percent"). |

### `redfish_chassis_leak_detector_health`

chassis leak detector health state,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Chassis/{chassis_id}/ThermalSubsystem/LeakDetection/LeakDetectors/{leak_detector_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "leak_detector" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `leak_detection_id` | *(same as value)* | *(constant)* | Constant string "LeakDetection" identifying the parent collection. |
| `leak_detector_id` | *(same as value)* | `Id` | Leak detector Redfish identifier; matches the {leak_detector_id} URL segment. |

### `redfish_chassis_power_voltage_state`

power voltage state of chassis component,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Power` → `Voltages[*].Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "power_voltage" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `power_voltage` | *(same as value)* | `Voltages[*].Name` | Human-readable voltage sensor name. |
| `power_voltage_id` | *(same as value)* | `Voltages[*].MemberId` | Stable per-chassis voltage sensor identifier. |

### `redfish_chassis_power_voltage_volts`

power voltage volts number of chassis component

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Power` → `Voltages[*].ReadingVolts`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "power_voltage" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `power_voltage` | *(same as value)* | `Voltages[*].Name` | Human-readable voltage sensor name. |
| `power_voltage_id` | *(same as value)* | `Voltages[*].MemberId` | Stable per-chassis voltage sensor identifier. |

### `redfish_chassis_power_average_consumed_watts`

power wattage watts number of chassis component

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Power` → `PowerControl[*].PowerMetrics.AverageConsumedWatts`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "power_wattage" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `power_voltage` | *(same as value)* | `PowerControl[*].Name` | PowerControl entry name. Reuses the voltage label slot (exporter quirk). |
| `power_voltage_id` | *(same as value)* | `PowerControl[*].MemberId` | PowerControl entry MemberId. Reuses the voltage label slot (exporter quirk). |

### `redfish_chassis_power_powersupply_state`

powersupply state of chassis component,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Power` → `PowerSupplies[*].Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "power_supply" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `power_supply` | *(same as value)* | `PowerSupplies[*].Name` | Human-readable power supply name. |
| `power_supply_id` | *(same as value)* | `PowerSupplies[*].MemberId` | PSU MemberId; falls back to PowerSupplies[*].SerialNumber when MemberId is empty. |

### `redfish_chassis_power_powersupply_health`

powersupply health of chassis component,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Power` → `PowerSupplies[*].Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "power_supply" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `power_supply` | *(same as value)* | `PowerSupplies[*].Name` | Human-readable power supply name. |
| `power_supply_id` | *(same as value)* | `PowerSupplies[*].MemberId` | PSU MemberId; falls back to PowerSupplies[*].SerialNumber when MemberId is empty. |

### `redfish_chassis_power_powersupply_power_efficiency_percentage`

rated efficiency, as a percentage, of the associated power supply on this chassis

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Power` → `PowerSupplies[*].EfficiencyPercent`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "power_supply" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `power_supply` | *(same as value)* | `PowerSupplies[*].Name` | Human-readable power supply name. |
| `power_supply_id` | *(same as value)* | `PowerSupplies[*].MemberId` | PSU MemberId; falls back to PowerSupplies[*].SerialNumber when MemberId is empty. |

### `redfish_chassis_power_powersupply_last_power_output_watts`

average power output, measured in Watts, of the associated power supply on this chassis

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Power` → `PowerSupplies[*].LastPowerOutputWatts`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "power_supply" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `power_supply` | *(same as value)* | `PowerSupplies[*].Name` | Human-readable power supply name. |
| `power_supply_id` | *(same as value)* | `PowerSupplies[*].MemberId` | PSU MemberId; falls back to PowerSupplies[*].SerialNumber when MemberId is empty. |

### `redfish_chassis_power_powersupply_power_input_watts`

measured input power, in Watts, of powersupply on this chassis

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Power` → `PowerSupplies[*].PowerInputWatts`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "power_supply" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `power_supply` | *(same as value)* | `PowerSupplies[*].Name` | Human-readable power supply name. |
| `power_supply_id` | *(same as value)* | `PowerSupplies[*].MemberId` | PSU MemberId; falls back to PowerSupplies[*].SerialNumber when MemberId is empty. |

### `redfish_chassis_power_powersupply_power_output_watts`

measured output power, in Watts, of powersupply on this chassis

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Power` → `PowerSupplies[*].PowerOutputWatts`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "power_supply" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `power_supply` | *(same as value)* | `PowerSupplies[*].Name` | Human-readable power supply name. |
| `power_supply_id` | *(same as value)* | `PowerSupplies[*].MemberId` | PSU MemberId; falls back to PowerSupplies[*].SerialNumber when MemberId is empty. |

### `redfish_chassis_power_powersupply_power_capacity_watts`

power_capacity_watts of powersupply on this chassis

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Power` → `PowerSupplies[*].PowerCapacityWatts`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "power_supply" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `power_supply` | *(same as value)* | `PowerSupplies[*].Name` | Human-readable power supply name. |
| `power_supply_id` | *(same as value)* | `PowerSupplies[*].MemberId` | PSU MemberId; falls back to PowerSupplies[*].SerialNumber when MemberId is empty. |

### `redfish_chassis_network_adapter_state`

chassis network adapter state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Chassis/{chassis_id}/NetworkAdapters/{network_adapter_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "network_adapter" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `network_adapter` | *(same as value)* | `Name` | Human-readable network adapter name. |
| `network_adapter_id` | *(same as value)* | `Id` | Network adapter Redfish identifier; matches {network_adapter_id}. |

### `redfish_chassis_network_adapter_health_state`

chassis network adapter health state,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Chassis/{chassis_id}/NetworkAdapters/{network_adapter_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "network_adapter" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `network_adapter` | *(same as value)* | `Name` | Human-readable network adapter name. |
| `network_adapter_id` | *(same as value)* | `Id` | Network adapter Redfish identifier; matches {network_adapter_id}. |

### `redfish_chassis_network_port_state`

chassis network port state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Chassis/{chassis_id}/NetworkAdapters/{network_adapter_id}/NetworkPorts/{network_port_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "network_port" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `network_adapter` | `/redfish/v1/Chassis/{chassis_id}/NetworkAdapters/{network_adapter_id}` | `Name` | Parent network adapter name. |
| `network_adapter_id` | `/redfish/v1/Chassis/{chassis_id}/NetworkAdapters/{network_adapter_id}` | `Id` | Parent network adapter Redfish identifier. |
| `network_port` | *(same as value)* | `Name` | Human-readable network port name. |
| `network_port_id` | *(same as value)* | `Id` | Network port Redfish identifier; matches {network_port_id}. |
| `network_port_type` | *(same as value)* | `ActiveLinkTechnology` | Active link technology (e.g. "Ethernet", "FibreChannel"). |
| `network_port_speed` | *(same as value)* | `CurrentLinkSpeedMbps` | Current link speed formatted as "<N> Mbps". |
| `network_port_connectiont_type` | *(same as value)* | `FCPortConnectionType` | FC port connection type (label name preserves historical typo). |
| `network_physical_port_number` | *(same as value)* | `PhysicalPortNumber` | Physical port number on the adapter. |

### `redfish_chassis_network_port_health_state`

chassis network port health state,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Chassis/{chassis_id}/NetworkAdapters/{network_adapter_id}/NetworkPorts/{network_port_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "network_port" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `network_adapter` | `/redfish/v1/Chassis/{chassis_id}/NetworkAdapters/{network_adapter_id}` | `Name` | Parent network adapter name. |
| `network_adapter_id` | `/redfish/v1/Chassis/{chassis_id}/NetworkAdapters/{network_adapter_id}` | `Id` | Parent network adapter Redfish identifier. |
| `network_port` | *(same as value)* | `Name` | Human-readable network port name. |
| `network_port_id` | *(same as value)* | `Id` | Network port Redfish identifier; matches {network_port_id}. |
| `network_port_type` | *(same as value)* | `ActiveLinkTechnology` | Active link technology (e.g. "Ethernet", "FibreChannel"). |
| `network_port_speed` | *(same as value)* | `CurrentLinkSpeedMbps` | Current link speed formatted as "<N> Mbps". |
| `network_port_connectiont_type` | *(same as value)* | `FCPortConnectionType` | FC port connection type (label name preserves historical typo). |
| `network_physical_port_number` | *(same as value)* | `PhysicalPortNumber` | Physical port number on the adapter. |

### `redfish_chassis_network_port_link_state`

chassis network port link state state,1(Up),0(Down)

**Type:** Gauge — **Value:** int\_enum (PortLinkState)

**Enum `PortLinkState`:**

| Code | Label |
|---|---|
| 0 | Down |
| 1 | Up |

**Value source:** `/redfish/v1/Chassis/{chassis_id}/NetworkAdapters/{network_adapter_id}/NetworkPorts/{network_port_id}` → `LinkStatus`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "network_port" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Parent chassis identifier propagated from the enclosing Chassis resource. |
| `network_adapter` | `/redfish/v1/Chassis/{chassis_id}/NetworkAdapters/{network_adapter_id}` | `Name` | Parent network adapter name. |
| `network_adapter_id` | `/redfish/v1/Chassis/{chassis_id}/NetworkAdapters/{network_adapter_id}` | `Id` | Parent network adapter Redfish identifier. |
| `network_port` | *(same as value)* | `Name` | Human-readable network port name. |
| `network_port_id` | *(same as value)* | `Id` | Network port Redfish identifier; matches {network_port_id}. |
| `network_port_type` | *(same as value)* | `ActiveLinkTechnology` | Active link technology (e.g. "Ethernet", "FibreChannel"). |
| `network_port_speed` | *(same as value)* | `CurrentLinkSpeedMbps` | Current link speed formatted as "<N> Mbps". |
| `network_port_connectiont_type` | *(same as value)* | `FCPortConnectionType` | FC port connection type (label name preserves historical typo). |
| `network_physical_port_number` | *(same as value)* | `PhysicalPortNumber` | Physical port number on the adapter. |

### `redfish_chassis_physical_security_sensor_state`

indicates the known state of the physical security sensor, such as if it is hardware intrusion detected,1(Normal),2(TamperingDetected),3(HardwareIntrusion)

**Type:** Gauge — **Value:** int\_enum (IntrusionSensor)

**Enum `IntrusionSensor`:**

| Code | Label |
|---|---|
| 1 | Normal |
| 2 | TamperingDetected |
| 3 | HardwareIntrusion |

**Value source:** `/redfish/v1/Chassis/{chassis_id}` → `PhysicalSecurity.IntrusionSensor`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `resource` | *(same as value)* | *(constant)* | Constant string "physical_security" identifying the metric group; not sourced from Redfish. |
| `chassis_id` | *(same as value)* | `Id` | Chassis Redfish identifier; matches the {chassis_id} URL segment. |
| `intrusion_sensor_number` | *(same as value)* | `PhysicalSecurity.IntrusionSensorNumber` | Intrusion sensor slot number (stringified). |
| `intrusion_sensor_rearm` | *(same as value)* | `PhysicalSecurity.IntrusionSensorReArm` | How the sensor is re-armed after tripping (e.g. "Manual", "Automatic"). |

### `redfish_chassis_log_service_state`

chassis log service state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Chassis/{chassis_id}/LogServices/{log_service_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Redfish chassis identifier this log service belongs to. |
| `log_service` | *(same as value)* | `Name` | Human-readable log service name. |
| `log_service_id` | *(same as value)* | `Id` | Log service Redfish identifier; matches {log_service_id}. |
| `log_service_enabled` | *(same as value)* | `ServiceEnabled` | Whether the log service is enabled ("true"/"false" as string). |
| `log_service_overwrite_policy` | *(same as value)* | `OverWritePolicy` | Ring-buffer overwrite policy (e.g. "WrapsWhenFull", "NeverOverwrites"). |

### `redfish_chassis_log_service_health_state`

chassis log service health state,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Chassis/{chassis_id}/LogServices/{log_service_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `chassis_id` | `/redfish/v1/Chassis/{chassis_id}` | `Id` | Redfish chassis identifier this log service belongs to. |
| `log_service` | *(same as value)* | `Name` | Human-readable log service name. |
| `log_service_id` | *(same as value)* | `Id` | Log service Redfish identifier; matches {log_service_id}. |
| `log_service_enabled` | *(same as value)* | `ServiceEnabled` | Whether the log service is enabled ("true"/"false" as string). |
| `log_service_overwrite_policy` | *(same as value)* | `OverWritePolicy` | Ring-buffer overwrite policy (e.g. "WrapsWhenFull", "NeverOverwrites"). |

