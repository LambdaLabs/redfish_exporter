# Module: powershelf

**56 metrics.**

Back to [overview](README.md).

### `redfish_powershelf_input_voltage`

PSU AC input voltage, volts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_psu_health`

PSU health,1(OK),2(Warning),3(Critical)

**Type:** Gauge — **Value:** int\_enum (CommonHealth)

**Enum `CommonHealth`:**

| Code | Label |
|---|---|
| 1 | OK |
| 2 | Warning |
| 3 | Critical |

**Value source:** `/redfish/v1/Chassis/{chassis_id}/PowerSubsystem/PowerSupplies/{psu_id}` → `Status.Health`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_psu_state`

PSU state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)

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

**Value source:** `/redfish/v1/Chassis/{chassis_id}/PowerSubsystem/PowerSupplies/{psu_id}` → `Status.State`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_output_voltage`

PSU DC output (rail) voltage, volts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_input_current`

PSU input current, amperes

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_output_current`

PSU output current, amperes

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_input_power`

PSU input power, watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_output_power`

PSU output power, watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_standby_output_voltage`

PSU standby rail voltage, volts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_standby_output_current`

PSU standby rail current, amperes

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_standby_output_power`

PSU standby rail power, watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_powerfactor`

PSU power factor, percent (0-100)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_energy_in`

PSU input energy (per-interval gauge)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_energy_out`

PSU output energy (per-interval gauge)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_total_energy_in`

PSU accumulated input energy (counter)

**Type:** Counter — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_fan1`

PSU fan 1 speed, RPM

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_fan2`

PSU fan 2 speed, RPM

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_temp_input`

PSU intake (ambient) temperature, celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_temp_output`

PSU exhaust temperature, celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_temp_hotspot`

PSU hotspot temperature, celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_temp_clip_plus`

PSU DC clip+ temperature, celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_temp_clip_minus`

PSU DC clip- temperature, celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_input_frequency`

PSU AC input frequency, hertz

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (sensor id → PSU by vendor adapter)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `power_supply_id` | *(same as value)* | *(constant)* | Canonical PSU id ("ps1".."ps6") produced by the vendor adapter from PowerSupplies[*].Id. |

### `redfish_powershelf_sensor_volts`

uncurated sensor reading, volts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (mapped by ReadingType, uncurated)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `sensor_id` | *(same as value)* | `Id` | Raw Redfish sensor identifier (unmodified by the adapter). |

### `redfish_powershelf_sensor_amperes`

uncurated sensor reading, amperes

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (mapped by ReadingType, uncurated)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `sensor_id` | *(same as value)* | `Id` | Raw Redfish sensor identifier (unmodified by the adapter). |

### `redfish_powershelf_sensor_watts`

uncurated sensor reading, watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (mapped by ReadingType, uncurated)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `sensor_id` | *(same as value)* | `Id` | Raw Redfish sensor identifier (unmodified by the adapter). |

### `redfish_powershelf_sensor_celsius`

uncurated sensor reading, celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (mapped by ReadingType, uncurated)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `sensor_id` | *(same as value)* | `Id` | Raw Redfish sensor identifier (unmodified by the adapter). |

### `redfish_powershelf_sensor_percent`

uncurated sensor reading, percent

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (mapped by ReadingType, uncurated)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `sensor_id` | *(same as value)* | `Id` | Raw Redfish sensor identifier (unmodified by the adapter). |

### `redfish_powershelf_sensor_rpm`

uncurated sensor reading, rpm

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (mapped by ReadingType, uncurated)`

**Labels:**

| Name | Endpoint | Field | Description |
|---|---|---|---|
| `sensor_id` | *(same as value)* | `Id` | Raw Redfish sensor identifier (unmodified by the adapter). |

### `redfish_powershelf_total_power_in`

shelf total input power, watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_total_power_in_A`

shelf total input power on phase A, watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_total_power_out`

shelf total output power, watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_total_current_out`

shelf total output current, amperes

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_average_current_out`

shelf average per-PSU output current, amperes

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_max_voltage_out`

shelf maximum output (rail) voltage, volts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_power_load`

shelf load, percent (0-100)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_total_efficiency`

shelf conversion efficiency, percent (0-100)

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_current_share`

shelf load-share current, amperes

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_voltage_in_A_A`

shelf AC input voltage, phase A, volts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_voltage_in_A_B`

shelf AC input voltage, phase B, volts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_voltage_in_A_C`

shelf AC input voltage, phase C, volts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_current_in_A_A`

shelf AC input current, phase A, amperes

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_current_in_A_B`

shelf AC input current, phase B, amperes

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_current_in_A_C`

shelf AC input current, phase C, amperes

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_bmc_12v`

shelf BMC 12V rail, volts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_bmc_3v3`

shelf BMC 3.3V rail, volts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_bmc_temp`

shelf BMC temperature, celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_hotswap_input_current`

shelf hotswap input current, amperes

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_hotswap_input_power`

shelf hotswap input power, watts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_hotswap_input_voltage`

shelf hotswap input voltage, volts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_hotswap_output_voltage`

shelf hotswap output voltage, volts

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_hotswap_temp`

shelf hotswap temperature, celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_total_current_in`

shelf total input current, amperes

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_temp_shelf`

shelf ambient temperature, celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_dc_temp_plus`

shelf DC bus (+) temperature, celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

### `redfish_powershelf_dc_temp_minus`

shelf DC bus (-) temperature, celsius

**Type:** Gauge — **Value:** float

**Value source:** `/redfish/v1/Chassis/{chassis_id}/Sensors/{sensor_id}` → `Reading (shelf-level sensor id mapped by adapter)`

**Labels:** none.

