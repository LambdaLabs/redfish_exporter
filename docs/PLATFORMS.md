# Platform differences

Where a platform's Redfish implementation makes this exporter behave differently from the
description in [CONFIGURATION.md](./CONFIGURATION.md), the reason is recorded here.

## Chassis: `Sensors` instead of `Thermal`/`Power`

Newer platforms — notably the GB200/GB300 NVL72 trays and the MGX NVSwitch tray — do not
implement the deprecated `Thermal` and `Power` schemas at all, and express the same
readings through `Sensors` instead. For those chassis the chassis collector falls back to
the `Sensors` collection, folding readings into the existing metric families wherever the
meaning is unambiguous, so a Sensors-only platform produces the same series names as one
that implements `Thermal`/`Power`:

| Redfish `ReadingType` | Metric |
| --- | --- |
| `Temperature` | `redfish_chassis_temperature_celsius` (+ `_sensor_health`, `_sensor_state`) |
| `Rotational` | `redfish_chassis_fan_rpm` (+ min/max/percentage/threshold series, `redfish_chassis_fan_health`, `_fan_state`) |
| `Voltage` | `redfish_chassis_power_voltage_volts` (+ `_state`) |
| anything else | `redfish_chassis_sensor_{watts,amperes,joules,hertz,percent,reading}` (+ `_health`, `_state`) |

The collection is consulted only on a chassis that advertises neither `Thermal` nor `Power`.
Collecting it alongside them would publish that chassis's temperatures twice under the same
series name, which fails the whole scrape at registration rather than merely thinning it.

A chassis advertising no `Sensors` collection is never asked for one. This matters more
than it sounds: the `ERoT`/`IRoT` roots are roughly a third of the chassis on an NVL72 tray
or an HGX baseboard, and synthesising `<chassis>/Sensors` for them would cost a 404 apiece
on every scrape.

Readings are not inferred from sensor naming: fan PWM duty cycle and CPU core utilisation
are both `ReadingType: Percent` and neither carries a distinguishing `PhysicalContext`, so
`Percent` reaches the catch-all rather than being assumed to be a fan speed.

Leak detectors are left out of this pass. A detector is dual-surfaced: a `LeakDetector`
resource carries its state, and a `Sensor` of the same Id carries an analog voltage. That
sensor is `ReadingType: Voltage` like any other, so folding it in would report a leak signal
as a rail measurement — the detector Ids enumerated from `ThermalSubsystem` are used to skip
it, leaving it to the leak detection path, which is the only thing able to interpret it
against its thresholds.

The collection is fetched with `$expand` so the BMC inlines every member body, costing one
request per chassis rather than one per sensor. A BMC that does not honour `$expand` is
**not** fanned out to one request per sensor — that would multiply load against the BMCs
least able to absorb it — so the unexpanded members are skipped with a warning instead. A
paginated collection is reported the same way rather than followed page by page, for the
same reason. Nothing is dropped silently.
