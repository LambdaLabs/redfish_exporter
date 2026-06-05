# Powershelf Metrics Contract

Status: **implemented** (Lite-On + Delta adapters) · Owner: ben.kulp · Last updated: 2026-06-05

## Purpose

One vendor-agnostic metric set for rack powershelves, so adding vendor N+1 means writing a
per-vendor *adapter that maps into this contract* not new metric names. A typed `powershelf`
collector reads each vendor's **standard Redfish** tree and emits these canonical series.

## Decisions (locked)

- **Approach:** typed Go `powershelf` collector + thin per-vendor adapters (one detect-and-map
  function each). Shared detection + PSU-status helpers live in `powershelf_collector.go`;
  per-vendor mapping lives in `powershelf_liteon.go` / `powershelf_delta.go`.
- **Canonical names follow the house convention `redfish_<subsystem>_<name>`, subsystem
  `powershelf`** (built via `addToMetricMap`) — e.g. `redfish_powershelf_input_voltage`. The
  tables below list the `<name>`; the emitted series is `redfish_powershelf_` + that.
  *(This renames the Delta series from `powershelf_*` to `redfish_powershelf_*`*
- **`power_supply_id` = `ps1..ps6`** (1-indexed). Delta sensor ids are already `ps<N>_…`;
  Lite-On's `p0..p5` map `+1` to `ps1..ps6`.
- **Values keep the device representation:** percentages stay 0–100 (`powerfactor`,
  `total_efficiency`, `power_load`); energy stays in its native unit (see E1).
- **Status is numeric** via `parseCommonStatusState`/`Health`. Lite-On PSUs report a
  non-standard `State: "Standby"`, added as code **12** (see the catch we hit below).

## Grounding — two BMC-confirmed devices (captures on disk)

| | **Delta `AHE50V660A33KWR`** | **Lite-On `PF-1333-7RB`** |
|---|---|---|
| oob_ip / capture | `10.254.35.25` · `tools/mock-server/testdata/delta/capture.txt` | `10.254.35.10` (`.13` too) · `tools/mock-server/testdata/liteon/capture.txt` |
| Class | 33 kW / 50 V / 660 A ORv3 HPR, 6× PSU `ECD17020047` @5504 W | 33 kW / 50 V / 660 A ORv3 HPR, 6× PSU `SP-2552-6RB` @5500 W |
| Legacy state | scraped by `json_collector` (job `delta-powershelves-redfish`) → 42 legacy `powershelf_*` | **unmonitored** — the Delta-shaped JQ can't read it |
| New collector | `redfish_powershelf_*` × **154** | `redfish_powershelf_*` × **84** (closes the gap) |

## Source of truth per vendor (standard Sensors, not OEM)

Both vendors expose the full reading set as a **standard Redfish `SensorCollection`** on their
powershelf chassis, plus a `PowerSubsystem` for PSU health/state. So both adapters use the same
two calls — `chassis.Sensors()` and `chassis.PowerSubsystem().PowerSupplies()` — and differ only
in how sensor ids map to canonical names and how PSUs are numbered.

| | **Delta** | **Lite-On** |
|---|---|---|
| Chassis (detect by `Manufacturer`) | `Chassis/PowerShelf_0` (`DELTA`) | `Chassis/powershelf` (`LITE-ON`), the alias with non-empty `Sensors` |
| Readings | `…/Sensors` — **identity map**: sensor `Id` *is* the canonical name (`ps1_input_current` → `input_current`+`ps1`; `total_power_in` → `total_power_in`) | `…/Sensors` — **translation map**: `voltage_p0_vin` → `input_voltage`+`ps1`, `chassis_input_power` → `total_power_in`, etc. |
| PSU health/state | `PowerSubsystem/PowerSupplies` (`Id` `1..6` → `ps1..ps6`) | `PowerSubsystem/PowerSupplies` (`Id` `0..5` → `ps1..ps6`) |
| Uncurated sensors | none (all 142 ids are canonical) | routed to the per-ReadingType catch-all |

*(The exiting Delta path scraped the `Oem.deltaenergysystems` blob via JQ; the typed
collector deliberately ignores OEM and reads the standard `Sensors` collection instead.)*

## Canonical metrics

### Per-PSU (label `power_supply_id=ps1..ps6`)

| `<name>` | Delta sensor id | Lite-On sensor id | Notes |
|---|---|---|---|
| `input_voltage` | `ps*_input_voltage` | `voltage_p*_vin` | V (AC ~240) |
| `input_current` | `ps*_input_current` | `current_p*_iin` | A |
| `input_power` | `ps*_input_power` | `power_p*_pin` | W |
| `output_voltage` | `ps*_output_voltage` | `voltage_p*_vout` | V (DC ~50) |
| `output_current` | `ps*_output_current` | `current_p*_iout` | A |
| `output_power` | `ps*_output_power` | `power_p*_pout` | W |
| `standby_output_voltage`/`_current`/`_power` | `ps*_standby_output_*` | — | Delta-only, ~12 V rail |
| `powerfactor` | `ps*_powerfactor` | — | Delta-only, 0–100 |
| `energy_in` / `energy_out` | `ps*_energy_{in,out}` (J, gauge) | — | Delta-only |
| `total_energy_in` | `ps*_total_energy_in` (kWh, counter) | — | Delta-only |
| `fan1` / `fan2` | `ps*_fan{1,2}` (RPM) | `fantach_p*_fan1` (fan1 only) | Lite-On = 1 fan/PSU |
| `temp_input` | `ps*_temp_input` | `temperature_p*_ambient` | intake °C |
| `temp_output` | `ps*_temp_output` | `temperature_p*_exhaust` | exhaust °C |
| `temp_hotspot` | `ps*_temp_hotspot` | `temperature_p*_hotspot` | °C |
| `temp_clip_plus` / `temp_clip_minus` | `ps*_temp_clip_{plus,minus}` | — | Delta-only DC-bus °C |
| `input_frequency` | — | `frequency_p*_freqin` (see F-open) | Hz; **currently unpopulated** |
| `psu_health` | PSU `.Status.Health` | PSU `.Status.Health` | 1=OK,2=Warn,3=Crit |
| `psu_state` | PSU `.Status.State` | PSU `.Status.State` | numeric; Lite-On `Standby`→12 |

### Shelf-level (no `power_supply_id`)

| `<name>` | Delta | Lite-On | Notes |
|---|---|---|---|
| `total_power_in` / `total_power_in_A` | ✅ | `power_chassis_input_power` → `total_power_in` | W (`_A` = phase A, Delta-only) |
| `total_power_out` | ✅ | `power_chassis_output_power` | W |
| `total_current_out` | ✅ | `current_chassis_output_current` | A |
| `total_current_in` | — | `chassis_input_current` → `total_current_in` (curated) | A; Lite-On-only |
| `average_current_out` / `max_voltage_out` | ✅ | — | Delta-only |
| `power_load` | ✅ | `utilization_chassis_load` | 0–100 |
| `total_efficiency` | ✅ | `utilization_chassis_efficiency` | 0–100 |
| `voltage_in_A_{A,B,C}` / `current_in_A_{A,B,C}` | ✅ (3-phase) | — | Delta-only |
| `bmc_12v` / `bmc_3v3` / `bmc_temp` | ✅ | — | Delta-only |
| `hotswap_input_current`/`_power`/`_voltage`, `hotswap_output_voltage`, `hotswap_temp` | ✅ | — | Delta-only |
| `current_share` | — | `current_ishare` | A; Lite-On-only |
| `temp_shelf` | — | `chassis_temperature` → `temp_shelf` (curated) | °C; Lite-On-only |
| `dc_temp_plus` / `dc_temp_minus` | — | `DC_temp_{plus,minus}` → `dc_temp_*` (curated) | °C; Lite-On-only |

### Catch-all (label `sensor_id`)

Any sensor an adapter doesn't curate is emitted under a per-`ReadingType` bucket instead of
being dropped — `sensor_volts`, `sensor_amperes`, `sensor_watts`, `sensor_celsius`,
`sensor_percent`, `sensor_rpm`, labeled with the raw `sensor_id`. Wired in the Lite-On adapter
(`chassis_input_voltage`, `chassis_output_voltage` land in `sensor_volts`). Delta needs none
today (identity map covers all 142), but the helper is vendor-neutral for future/unknown sensors.

## Resolved

- **H1 (Delta PSU health):** Delta exposes it via `PowerSubsystem/PowerSupplies/N.Status`;
  both adapters emit `psu_health`/`psu_state`.
- **E1 (energy):** Delta `energy_in`/`energy_out` are **Joules** gauges (NOT counters, confirmed
  by value range over 24 h); `total_energy_in` is **kWh** counter (`EnergykWh`).
- **C1 (`ishare`):** amperes (`ReadingType=Current`) → `current_share`.
- **F1 (Delta fans):** per-PSU `ps*_fan{1,2}` (RPM) from the Sensors collection.
- **T1 (temps):** intake (`temp_input`, ~24 °C) vs exhaust (`temp_output`, ~46 °C) settled by
  value range; Lite-On `DC_temp_±` are shelf-level → `dc_temp_*` (not the per-PSU `temp_clip_*`).
- **U1 (percent vs ratio):** kept 0–100.
- **`Standby` state:** Lite-On PSUs report a non-standard `State: "Standby"`;
  `parseCommonStatusState` gained code **12** for it (and `StandbyOffline`→3). Changing the
  shared `CommonStateHelp` rippled into the GPU collector tests + a golden file — those HELP
  lines were updated to match.

## Open items

- **`input_frequency` (Lite-On) — punted (low value, Lite-On-only).** The 6 `frequency_p*_freqin`
  sensors (~60 Hz mains) exist but are **not Members** of the chassis `Sensors` collection, so
  `chassis.Sensors()` never returns them — the metric stays unpopulated. They *are* reachable via
  each PSU object's non-standard `Sensors` link array (gofish doesn't type that field, but
  `PowerSupply.RawData` retains it; `schemas.GetSensor(client, uri)` fetches one). Two ways to
  wire it if ever needed: (A) construct `…/Sensors/frequency_p<N>_freqin` per PSU and `GetSensor`;
  (B) parse `PowerSupply.RawData`'s `Sensors` links and pick the `ReadingType=Frequency` one.
  **Decision: punt** — Delta exposes no frequency at all (and no per-PSU `Sensors` link), so this
  is a single-vendor, likely near-constant reading; not worth the code or the cross-vendor divergence.
- **`total_energy_in` scale:** counter confirmed; the ~65 wrap ceiling suggests a scaled
  register — confirm the kWh scale with Delta before relying on `rate()` magnitudes.
