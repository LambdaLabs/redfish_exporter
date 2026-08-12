# Configuration

Generic types include:

- `<string>` regular string values
- `<float64>` regular float64 value as expected in Golang
- `<timeout>` a string representation of a Golang time.Duration, e.g. `30s`

```yaml
hosts:
  [ <string>: <hostdetail> ]
  default: <hostdetail>
groups:
  [ <string>: <hostdetail> ]
modules: [ <string>: <module> ]
```

## `<hostdetail>`
``` yaml
username: <string>
password: <string>
```

Note that the `default` entry above is useful in order to avoid the exporter failing when attempting to collect from a host not explicitly defined in `hosts`.

## `<module>`
Users of `blackbox_exporter` will be familiar with the concept of [modules (aka probers)](https://github.com/prometheus/blackbox_exporter/blob/master/CONFIGURATION.md#module).
Since `v0.2.0-alpha` released in October 2025, this fork of the redfish_exporter supports a similar concept.
With modules, users may enable only certain bits of collector behavior, tailoring data gathering to specific hosts.
As an example, a host without GPUs wouldn't want to spend network round-trips using the `gpu_collector` prober module, and that may be left out on the client (Prometheus) end.

**Note:** To retain backwards compatibility, the redfish_exporter will default to probing a system using all collectors below except the JSON collector. If this is not desired behavior, users must define a `modules:` configuration in the redfish_exporter, and specify one or more `module` HTTP query parameters on the client end.

**In a future release, users will be required to specify modules in the exporter and client**

Modules follow the below configuration schema:

``` yaml
# The type of prober this module uses
prober: <prober_string>

[ chassis_collector: <chassis_collector> ]
[ gpu_collector: <gpu_collector> ]
[ json_collector: <json_collector> ]
[ manager_collector: <manager_collector> ]
[ system_collector: <system_collector> ]
[ telemetry_collector: <telemetry_collector> ]
```

### `<prober_string>`

One of:

- `chassis_collector`
- `gpu_collector`
- `json_collector`
- `manager_collector`
- `system_collector`
- `telemetry_collector`

### `<chassis_collector>`
[source](../collector/chassis_collector.go)

The Chassis Collector primarily exposes health data from the Chassis API. Against a Lambda lab system, the collector yields the following timeseries:

```
# HELP redfish_chassis_health health of chassis,1(OK),2(Warning),3(Critical)
# TYPE redfish_chassis_health gauge

# HELP redfish_chassis_health_rollup health rollup of chassis,1(OK),2(Warning),3(Critical)
# TYPE redfish_chassis_health_rollup gauge

# HELP redfish_chassis_leak_detection_health health of the chassis leak detection subsystem as a whole,1(OK),2(Warning),3(Critical)
# TYPE redfish_chassis_leak_detection_health gauge

# HELP redfish_chassis_leak_detection_state state of the chassis leak detection subsystem as a whole,1(Enabled),2(Disabled),...
# TYPE redfish_chassis_leak_detection_state gauge

# HELP redfish_chassis_leak_detector_enabled whether this chassis leak detector is enabled, 1(enabled),0(disabled); a disabled detector reports Unavailable state and does not trigger events
# TYPE redfish_chassis_leak_detector_enabled gauge

# HELP redfish_chassis_leak_detector_health chassis leak detector health state,1(OK),2(Warning),3(Critical)
# TYPE redfish_chassis_leak_detector_health gauge

# HELP redfish_chassis_leak_detector_info chassis leak detector type and physical location, always 1
# TYPE redfish_chassis_leak_detector_info gauge

# HELP redfish_chassis_leak_detector_state chassis leak detector state; this is the signal to alert on, and a Critical state is a detector trip that the companion voltage classifies as wet or as contamination,1(OK),2(Warning),3(Critical),4(Unavailable),5(Absent)
# TYPE redfish_chassis_leak_detector_state gauge

# HELP redfish_chassis_leak_detector_volts chassis leak detector reading in volts; falls toward the lower critical threshold as moisture is detected
# TYPE redfish_chassis_leak_detector_volts gauge

# HELP redfish_chassis_leak_detector_volts_lower_threshold_critical voltage at or below which this chassis leak detector reports a critical leak
# TYPE redfish_chassis_leak_detector_volts_lower_threshold_critical gauge

# HELP redfish_chassis_leak_detector_volts_upper_threshold_critical voltage at or above which this chassis leak detector is considered faulty, typically an open or shorted sense line rather than a leak
# TYPE redfish_chassis_leak_detector_volts_upper_threshold_critical gauge

# HELP redfish_chassis_model_info organization responsible for producing the chassis, the name by which the manufacturer generally refers to the chassis, and a part number and sku assigned by the organization that is responsible for producing or manufacturing the chassis
# TYPE redfish_chassis_model_info gauge

# HELP redfish_chassis_network_adapter_state chassis network adapter state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating)
# TYPE redfish_chassis_network_adapter_state gauge

# HELP redfish_chassis_state state of chassis,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating)
# TYPE redfish_chassis_state gauge
```

Newer platforms — notably the GB200/GB300 NVL72 trays and the MGX NVSwitch tray — do not
implement the deprecated `Thermal` and `Power` schemas at all, and express the same
readings through `Sensors` instead. For those chassis the collector falls back to the
`Sensors` collection, folding readings into the metric families above wherever the meaning
is unambiguous, so a Sensors-only platform produces the same series names as one that
implements `Thermal`/`Power`:

| Redfish `ReadingType` | Metric |
| --- | --- |
| `Temperature` | `redfish_chassis_temperature_celsius` (+ `_sensor_health`, `_sensor_state`) |
| `Rotational` | `redfish_chassis_fan_rpm` (+ min/max/percentage/threshold series) |
| `Voltage` | `redfish_chassis_power_voltage_volts` (+ `_state`) |
| anything else | `redfish_chassis_sensor_{watts,amperes,joules,hertz,percent,reading}` |

`Sensors` is consulted for two distinct reasons, and the reason decides what is emitted:

- **Standing in for the legacy schemas**, on a chassis advertising neither `Thermal` nor
  `Power`. The whole collection is emitted.
- **For the leak detectors alone**, on a chassis that carries them, since their voltages
  live nowhere else. Only those sensors are emitted — otherwise a chassis implementing
  `Thermal` *and* leak detection would publish its temperatures twice under the same series
  name, which fails the whole scrape at registration.

A chassis advertising no `Sensors` collection is never asked for one. This matters more
than it sounds: the `ERoT`/`IRoT` roots are roughly a third of the chassis on an NVL72 tray
or an HGX baseboard, and synthesising `<chassis>/Sensors` for them would cost a 404 apiece
on every scrape.

Readings are not inferred from sensor naming: fan PWM duty cycle and CPU core utilisation
are both `ReadingType: Percent` and neither carries a distinguishing `PhysicalContext`, so
`Percent` reaches the catch-all rather than being assumed to be a fan speed. Sensors that
are simply not wanted are dropped by `sensor_exclude` (below) rather than by a rule in the
collector.

The collection is fetched with `$expand` so the BMC inlines every member body, costing one
request per chassis rather than one per sensor. Measured against captured BMC dumps, the net
change in requests per scrape is **−2 to +2** — the new `Sensors` requests are paid for by a
duplicate `LeakDetection` request this release also removes:

| Platform | before | after |
| --- | --- | --- |
| SYS-A21GE-NBRT (B200, 39 chassis) | 116 | 115 |
| SYS-821GE-TNHR (H100/H200, 43) | 118–119 | 117–118 |
| ARS-121GL-NB3 (GB300 tray, 42) | 124 | 124 |
| N5500_LD (MGX NVSwitch tray, 12) | 48 | 48 |
| GH200 (4) | 20 | 22 |

A BMC that does not honour `$expand` is **not** fanned out to one request per sensor — that
would multiply load against the BMCs least able to absorb it. Only the leak detector sensors
are then fetched individually and the bulk sensor telemetry is skipped with a warning, so
request count stays flat and the safety-relevant readings still arrive.

#### Leak detection

`redfish_chassis_leak_detector_state` is the signal to alert on. A `Critical` state is a
detector *trip* rather than a confirmed leak — see the classification below — but every trip
warrants a response. `..._health` describes the health of the detector *device*, which can
remain `OK` while a trip is reported, so it must not be used as a leak indicator on its own.

**Alert on `== 3` rather than `>= 2`.** Unlike the health encoding, values above 3 do not
mean "worse than critical": 4 (`Unavailable`) and 5 (`Absent`) mean the detector is not
reporting. A `> 2` expression would fire a critical trip for an absent detector. Use `== 2`
for a warning-level trip, `== 3` for a critical one, and `>= 4` to alert separately on the
blind spot.

**Alert on `..._state`; classify with `..._volts`.** Across five captured GB300 trays,
three show a detector reporting `DetectorState: Critical` while its companion voltage sits
at a nominal ~1.72 V, well clear of the lower critical threshold. All three were real
detector trips and none was a coolant leak: the cause was dust bridging the sense contacts.
The detector measures resistance across those contacts, so anything conductive enough to
bridge them trips it, and a dust bridge trips the discrete state without pulling the divider
voltage far enough to cross its own threshold.

That disagreement is the diagnosis, available at alert time rather than after someone opens
the tray. Page on the state, and let the voltage decide which runbook fires:

```promql
# 1. Confirmed wet-out: the trip is corroborated by the analog reading. Coolant leak.
redfish_chassis_leak_detector_state == 3
  and on (instance, chassis_id, leak_detection_id, leak_detector_id)
    redfish_chassis_leak_detector_volts <= redfish_chassis_leak_detector_volts_lower_threshold_critical

# 2. Trip with the voltage present and nominal: contamination across the sense contacts is
#    the likeliest cause. Real, and worth a physical inspection, but not a leak response.
(redfish_chassis_leak_detector_state == 3
  and on (instance, chassis_id, leak_detection_id, leak_detector_id)
    redfish_chassis_leak_detector_volts)
  unless on (instance, chassis_id, leak_detection_id, leak_detector_id)
    redfish_chassis_leak_detector_volts <= redfish_chassis_leak_detector_volts_lower_threshold_critical

# 3. Trip on a detector that publishes no voltage at all (the MGX NVSwitch tray). Not
#    classifiable without opening the tray, so treat it at leak severity.
redfish_chassis_leak_detector_state == 3
  unless on (instance, chassis_id, leak_detection_id, leak_detector_id)
    redfish_chassis_leak_detector_volts
```

**The arms must cover the whole of `..._state == 3` between them**, which is why the third
exists: without it, rule 2 also matches every detector that publishes no voltage, and a real
leak on the NVSwitch tray would route to the contamination runbook. Shipping only rule 1
would suppress all three captured events outright. If you take one rule rather than three,
take the bare `..._state == 3` — the classification refines the alert, it never filters it.

Coverage varies by platform, so `..._state` is the only signal available everywhere:

- **GB300 compute tray (ARS-121GL-NB3):** 4 detectors on the tray shelf chassis, each with
  a companion `Sensor` giving `..._volts` and its thresholds. These are resistive moisture
  ropes on a voltage divider — dry reads high (~1.72 V) and water pulls the voltage *down*
  toward the lower critical threshold, so the leak alarm is a **lower** critical crossing.
  The threshold value is firmware-dependent (1.3 V and 1.65 V both appear across captured
  trays), which is why it is exported as a metric rather than written into an alert. It is
  only exposed on the `Sensor` resource, never in the telemetry metric report.
  `..._volts_upper_threshold_critical` (2.0 V where reported) is *not* a wetter-still
  reading: an open or shorted sense line reads high, so crossing it means the detector has
  stopped being able to see a leak.
- **MGX NVSwitch tray (P3809):** 7 detectors (including an aggregate, `leakage_aggr`) on one
  chassis, with six further chassis exposing an empty detector collection. Discrete only —
  no companion voltage sensors. Note the aggregate is structurally indistinguishable from
  the individual detectors, so an unqualified alert will fire twice for one physical leak.
- Some BMCs advertise a `LeakDetection` link that then returns 404; this is logged at debug
  level and emits nothing.

#### Configuration

```yaml
chassis_collector:
  # Regular expressions matched against each chassis Id. Include is applied first.
  [ chassis_include: <regexp> ]
  [ chassis_exclude: <regexp> ]

  # Regular expression matched against each Sensor Id. Matching sensors emit no metrics.
  # Trims series count, not request count: the collection arrives in one request either
  # way. Leak detector sensors are never excluded by this pattern.
  #
  # Omitting the key and setting it to "" are different: omitted means the default below,
  # empty means exclude nothing.
  [ sensor_exclude: <regexp> | default = "_CoreUtil_[0-9]+$" ]

  # Subsystem opt-outs. Every default is false, so an empty config collects everything.
  [ disable_thermal: <bool> ]
  [ disable_thermal_subsystem: <bool> ]   # also disables leak detection
  [ disable_power: <bool> ]
  [ disable_network_adapters: <bool> ]
  [ disable_sensors: <bool> ]             # also drops the leak detector voltages
```

`sensor_exclude` defaults to `"_CoreUtil_[0-9]+$"`. A GB300 tray publishes 144 per-core CPU
utilisation sensors, which is more series than the rest of the chassis collector produces
for that tray combined, and the same data is available in-band from `node_exporter` at
higher resolution. The telemetry collector already declines the identical sensors, so this
keeps the two collectors from disagreeing about the same hardware; expressing it as a
configuration default rather than as a rule in the collector keeps the judgement visible and
overridable, and leaves the collector free of any inference from sensor names.

The default applies wherever the key is absent — a hand-written `chassis_collector:` block,
`chassis_collector: {}`, and the built-in modules all behave identically. To collect those
sensors, set the pattern to the empty string:

```yaml
chassis_collector:
  sensor_exclude: ""
```

##### Scraping leak detection on a short interval

A full chassis scrape walks every chassis and can issue over a hundred requests on a tray
with many chassis, which at the default `max_concurrent_requests: 1` serialises. The
built-in `leak_detection` module trims that to what a leak alert needs:

```yaml
modules:
  leak_detection:
    prober: chassis_collector
    chassis_collector:
      disable_thermal: true
      disable_power: true
      disable_network_adapters: true
      # Optional, and vendor specific. Worth setting: it is the difference between
      # walking every chassis and walking only the one that carries detectors.
      # Supermicro NVL72 compute tray:
      # chassis_include: "^Chassis_[0-9]+$"
      # MGX NVSwitch tray:
      # chassis_include: "^MGX_BMC_[0-9]+$"
```

Setting `disable_thermal` and `disable_power` together does more than skip those two
fetches: it also narrows the `Sensors` pass to the leak detectors. Without that, on exactly
the liquid-cooled platforms this module targets — where no chassis implements either legacy
schema — `Sensors` would stand in for what was just disabled and the module would cost
almost as much as a full scrape.

Measured requests per scrape, against captured dumps:

| | full chassis scrape | `leak_detection` | + `chassis_include` |
| --- | --- | --- | --- |
| ARS-121GL-NB3 (GB300 tray) | 124 | 78 | 11 |
| N5500_LD (MGX NVSwitch tray) | 48 | 42 | 14 |
| SYS-A21GE-NBRT (B200) | 115 | 64 | — |

`chassis_include` and `chassis_exclude` are applied to the collection's member links before
any chassis body is fetched, so a filtered scrape costs one request per *matching* chassis
rather than one per chassis. That is what makes the module cheap enough to poll on a short
interval: an unfiltered `leak_detection` scrape of a GB300 tray spends 43 of its 78 requests
just enumerating the forty-two chassis it is about to discard.

Filtering costs one extra request per scrape — the service root, read to find the
`ChassisCollection` link rather than assuming it — and matches on the trailing segment of
each member URI, which by Redfish convention is the chassis `Id`. The pattern is applied
again to the fetched `Id`, so a BMC that breaks that convention filters correctly, just
without the saving.

### `<gpu_collector>`
[source](../collector/gpu_collector.go)

The GPU Collector primarily exposes a wealth of Nvidia GPU-related data using a mix of Oem and Processor API data.
**Note:** This collector iterates many Redfish paths and depending on network round trips, may take a significant amount of time (30+ seconds) to return data.

Against a Lambda lab system, the collector yields the following timeseries:

```text
# HELP redfish_exporter_collector_duration_seconds Collector time duration.
# TYPE redfish_exporter_collector_duration_seconds gauge

# HELP redfish_gpu_context_utilization_seconds_total Accumulated GPU context utilization duration in seconds
# TYPE redfish_gpu_context_utilization_seconds_total counter

# HELP redfish_gpu_health GPU processor health,1(OK),2(Warning),3(Critical)
# TYPE redfish_gpu_health gauge

# HELP redfish_gpu_info GPU information with serial number and UUID
# TYPE redfish_gpu_info gauge

# HELP redfish_gpu_memory_capacity_mib GPU memory capacity in MiB
# TYPE redfish_gpu_memory_capacity_mib gauge

# HELP redfish_gpu_memory_correctable_row_remapping_count GPU memory correctable row remapping count
# TYPE redfish_gpu_memory_correctable_row_remapping_count gauge

# HELP redfish_gpu_memory_ecc_correctable current correctable memory ecc errors reported on the gpu
# TYPE redfish_gpu_memory_ecc_correctable counter

# HELP redfish_gpu_memory_ecc_uncorrectable current uncorrectable memory ecc errors reported on the gpu
# TYPE redfish_gpu_memory_ecc_uncorrectable counter

# HELP redfish_gpu_memory_health GPU memory health,1(OK),2(Warning),3(Critical)
# TYPE redfish_gpu_memory_health gauge

# HELP redfish_gpu_memory_high_availability_bank_count GPU memory high availability bank count
# TYPE redfish_gpu_memory_high_availability_bank_count gauge

# HELP redfish_gpu_memory_low_availability_bank_count GPU memory low availability bank count
# TYPE redfish_gpu_memory_low_availability_bank_count gauge

# HELP redfish_gpu_memory_max_availability_bank_count GPU memory max availability bank count
# TYPE redfish_gpu_memory_max_availability_bank_count gauge

# HELP redfish_gpu_memory_no_availability_bank_count GPU memory no availability bank count
# TYPE redfish_gpu_memory_no_availability_bank_count gauge

# HELP redfish_gpu_memory_partial_availability_bank_count GPU memory partial availability bank count
# TYPE redfish_gpu_memory_partial_availability_bank_count gauge

# HELP redfish_gpu_memory_row_remapping_failed GPU memory row remapping failed status (1 if failed)
# TYPE redfish_gpu_memory_row_remapping_failed gauge

# HELP redfish_gpu_memory_row_remapping_pending GPU memory row remapping pending status (1 if pending)
# TYPE redfish_gpu_memory_row_remapping_pending gauge

# HELP redfish_gpu_memory_state GPU memory state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating)
# TYPE redfish_gpu_memory_state gauge

# HELP redfish_gpu_memory_uncorrectable_row_remapping_count GPU memory uncorrectable row remapping count
# TYPE redfish_gpu_memory_uncorrectable_row_remapping_count gauge

# HELP redfish_gpu_nvlink_bit_error_rate NVLink bit error rate
# TYPE redfish_gpu_nvlink_bit_error_rate gauge

# HELP redfish_gpu_nvlink_health NVLink port health,1(OK),2(Warning),3(Critical)
# TYPE redfish_gpu_nvlink_health gauge

# HELP redfish_gpu_nvlink_link_downed_count NVLink link downed count
# TYPE redfish_gpu_nvlink_link_downed_count gauge

# HELP redfish_gpu_nvlink_link_error_recovery_count NVLink error recovery count
# TYPE redfish_gpu_nvlink_link_error_recovery_count gauge

# HELP redfish_gpu_nvlink_runtime_error NVLink runtime error status (1 if error)
# TYPE redfish_gpu_nvlink_runtime_error gauge

# HELP redfish_gpu_nvlink_state NVLink port state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating)
# TYPE redfish_gpu_nvlink_state gauge

# HELP redfish_gpu_nvlink_symbol_errors NVLink symbol error count
# TYPE redfish_gpu_nvlink_symbol_errors gauge

# HELP redfish_gpu_nvlink_training_error NVLink training error status (1 if error)
# TYPE redfish_gpu_nvlink_training_error gauge

# HELP redfish_gpu_sram_ecc_error_threshold_exceeded GPU SRAM ECC error threshold exceeded (1 if exceeded)
# TYPE redfish_gpu_sram_ecc_error_threshold_exceeded gauge

# HELP redfish_gpu_state GPU processor state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating)
# TYPE redfish_gpu_state gauge

# HELP redfish_up redfish up
# TYPE redfish_up gauge
```

Exposes no user configuration.

### `<json_collector>`
[source](../collector/json_collector.go)

The JSON Collector allows probing Redfish APIs at a user-defined path, and deriving Prometheus gauge values from the result by use of JQ (using [gojq](https://github.com/itchyny/gojq) in particular).

The intent of this collector is to allow for greater end-user flexibility and agility in heterogeneous hardware environments.

OEM data is one good example of this; OEMs often evolve their own specific extensions and data faster than those updates percolate to Redfish libraries or even into publicly-available documentation. In those cases, the JSON collector may be a good fit to emit telemetry from arbitrary endpoints versus extending existing or adding new collectors to the core codebase to cover edge cases in OEM Redfish implementations.

Care should be taken when using this prober. Telemetry data is only as useful as it is both clear and understandable. Both of those criteria require finesse for what is at its core a transparent transformer around JSON data.

**NOTE:** Internally, the collector emits Gauge values. Care must be taken to collect and emit gauge-like data (power and fan data are good examples of gauges). Counter values _may_ work, but would not be canonically correct for use in Prometheus.

A recommended development cycle for this prober might be:
- Save the raw JSON output from a desired Redfish API path
- Iterate on the JQ script locally, targeting that saved output. See below for details on required JSON schema.
- Load the script and redfish path into a local copy of the redfish_exporter config (see below)
- Perform a single scrape through the exporter, backtesting against the same original source system
- Iterate on the script, if necessary

In order to transform data, the prober needs to execute a JQ script which yields data in the following output format:

``` json5
[
{
  # name of the resulting Prometheus timeseries
  "name": "<string>",
  # help text for the series. Prefer a static string, multiple values for a help string for the same timeseries name is invalid to Prometheus and will result in errors
  "help": "<string>",
  # value of the timeseries. Must be a float64.
  "value": <float64>,
  # OPTIONAL labels to assign to the timeseries.
  "labels": {
    "<string>": "<string>"
  }
},
...
]
```

This output is iterated over in the code, and gauge values are emitted as a result.

``` yaml
# Configurable timeout context, JQ parsing _must_ complete within this timeout.
context_timeout: <timeout> | default = 30s
# Redfish API path to probe
redfishPath: <string>
# JQ script to execute against the redfishPath. Note that the resulting format must match what is documented above.
jq: <string>
```

Examples of how this prober might be configured may be found in the [config testdata directory](../config/testdata/), and we welcome contributions adding more exotic ones.

### `<manager_collector>`
[source](../collector/manager_collector.go)

The Manager Collector exposes health data from the Manager API (e.g. the BMC itself). Against a Lambda lab system, the collector yields the following timeseries:

```
# HELP redfish_manager_health_state manager health,1(OK),2(Warning),3(Critical)
# TYPE redfish_manager_health_state gauge

# HELP redfish_manager_power_state manager power state
# TYPE redfish_manager_power_state gauge

# HELP redfish_manager_state manager state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating)
# TYPE redfish_manager_state gauge
```

Exposes no user configuration.

### `<system_collector>`
[source](../collector/system_collector.go)

The System Collector primarily exposes health data from the Systems API, as well as some amount of capacity and PCIe data. Against a Lambda lab system, the collector yields the following timeseries:

```
# HELP redfish_exporter_collector_duration_seconds Collector time duration.
# TYPE redfish_exporter_collector_duration_seconds gauge

# HELP redfish_system_health_state system health,1(OK),2(Warning),3(Critical)
# TYPE redfish_system_health_state gauge

# HELP redfish_system_memory_capacity system memory capacity, MiB
# TYPE redfish_system_memory_capacity gauge

# HELP redfish_system_memory_health_state system memory health state,1(OK),2(Warning),3(Critical)
# TYPE redfish_system_memory_health_state gauge

# HELP redfish_system_memory_state system memory state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating)
# TYPE redfish_system_memory_state gauge

# HELP redfish_system_power_state system power state
# TYPE redfish_system_power_state gauge

# HELP redfish_system_processor_cache_lifetime_correctable_ecc_error_count system processor cache lifetime correctable ECC error count
# TYPE redfish_system_processor_cache_lifetime_correctable_ecc_error_count gauge

# HELP redfish_system_processor_cache_lifetime_uncorrectable_ecc_error_count system processor cache lifetime uncorrectable ECC error count
# TYPE redfish_system_processor_cache_lifetime_uncorrectable_ecc_error_count gauge

# HELP redfish_system_processor_health_rollup system processor health rollup,1(OK),2(Warning),3(Critical)
# TYPE redfish_system_processor_health_rollup gauge

# HELP redfish_system_processor_health_state system processor health state,1(OK),2(Warning),3(Critical)
# TYPE redfish_system_processor_health_state gauge

# HELP redfish_system_processor_pcie_errors_correctable_count system processor PCIe correctable error count
# TYPE redfish_system_processor_pcie_errors_correctable_count gauge

# HELP redfish_system_processor_pcie_errors_fatal_count system processor PCIe fatal error count
# TYPE redfish_system_processor_pcie_errors_fatal_count gauge

# HELP redfish_system_processor_pcie_errors_l0_to_recovery_count system processor PCIe L0 to recovery state transition count
# TYPE redfish_system_processor_pcie_errors_l0_to_recovery_count gauge

# HELP redfish_system_processor_pcie_errors_nak_received_count system processor PCIe NAK received count
# TYPE redfish_system_processor_pcie_errors_nak_received_count gauge

# HELP redfish_system_processor_pcie_errors_nak_sent_count system processor PCIe NAK sent count
# TYPE redfish_system_processor_pcie_errors_nak_sent_count gauge

# HELP redfish_system_processor_pcie_errors_non_fatal_count system processor PCIe non-fatal error count
# TYPE redfish_system_processor_pcie_errors_non_fatal_count gauge

# HELP redfish_system_processor_pcie_errors_replay_count system processor PCIe replay count
# TYPE redfish_system_processor_pcie_errors_replay_count gauge

# HELP redfish_system_processor_pcie_errors_replay_rollover_count system processor PCIe replay rollover count
# TYPE redfish_system_processor_pcie_errors_replay_rollover_count gauge

# HELP redfish_system_processor_state system processor state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating)
# TYPE redfish_system_processor_state gauge

# HELP redfish_system_processor_total_cores system processor total cores
# TYPE redfish_system_processor_total_cores gauge

# HELP redfish_system_processor_total_threads system processor total threads
# TYPE redfish_system_processor_total_threads gauge

# HELP redfish_system_state system state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating)
# TYPE redfish_system_state gauge

# HELP redfish_system_storage_drive_capacity system storage drive capacity, Bytes
# TYPE redfish_system_storage_drive_capacity gauge

# HELP redfish_system_storage_drive_state system storage drive state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating)
# TYPE redfish_system_storage_drive_state gauge
```

Exposes no user configuration.

### `<telemetry_collector>`
[source](../collector/telemetry_collector.go)

The Telemetry Collector exposes a wealth of data using the TelemetryService API.
It was designed to capture much OEM data for Nvidia systems, and as such may not be less useful in non-GPU hardware environments.

Reports are routed by exact match on their ID with any trailing instance index removed, so
`HGX_ProcessorMetrics_0` and `HGX_ProcessorMetrics_1` share a handler. Matching is exact
rather than by substring because substrings are ambiguous here: `ProcessorPortGPMMetrics`
and `ProcessorPortMetrics` are different reports, and a bare `ProcessorMetrics` is a
substring of `CpuProcessorMetrics`.

**A report with no handler is reported, not silently dropped.** It emits
`redfish_telemetry_unhandled_report{report_id="..."}` carrying the number of discarded metric
values. A non-zero value means a platform is publishing telemetry this exporter does not
parse — the usual cause of an unnoticed coverage gap on new hardware. That metric is the
alertable form; the collector additionally logs one warn line per scrape naming every
unhandled report at once, rather than one line each, since an HGX baseboard has around
sixteen of them on every scrape and that is a standing property of the platform rather than
an incident.

Known unhandled reports at time of writing: `HGX_NVSwitchPortMetrics` (3,552 metric values
on a B200 — per-NVLink CRC, replay, recovery and training errors), `HGX_NVSwitchMetrics`
(NVSwitch PCIe and lifetime ECC counters), `HGX_PCIeRetimerMetrics`,
`HGX_PCIeRetimerPortMetrics`, `HGX_NetworkAdapterPortMetrics`, and the
`{Min,Max,Avg}PowerConsumption{Hour,Day,Week}` rollups. NVSwitch chassis temperature and
power are unaffected by this — they come from the chassis collector's `Sensors` pass, which
is the only source at all on the dedicated MGX NVSwitch tray since that BMC exposes no
`TelemetryService`.

Against a Lambda lab system, the collector yields the following timeseries:

```
# HELP redfish_gpu_temperature_tlimit_celsius GPU TLIMIT temperature headroom in Celsius
# TYPE redfish_gpu_temperature_tlimit_celsius gauge

# HELP redfish_telemetry_ambient_exhaust_temperature_celsius Ambient exhaust temperature in Celsius
# TYPE redfish_telemetry_ambient_exhaust_temperature_celsius gauge

# HELP redfish_telemetry_ambient_inlet_temperature_celsius Ambient inlet temperature in Celsius
# TYPE redfish_telemetry_ambient_inlet_temperature_celsius gauge

# HELP redfish_telemetry_bmc_temperature_celsius BMC temperature in Celsius
# TYPE redfish_telemetry_bmc_temperature_celsius gauge

# HELP redfish_telemetry_cache_ecc_correctable_total Total correctable ECC errors in GPU cache (L2/SRAM)
# TYPE redfish_telemetry_cache_ecc_correctable_total counter

# HELP redfish_telemetry_cache_ecc_uncorrectable_total Total uncorrectable ECC errors in GPU cache (L2/SRAM)
# TYPE redfish_telemetry_cache_ecc_uncorrectable_total counter

# HELP redfish_telemetry_collection_stale_reports_last Quantity of stale reports discovered on the last collection loop
# TYPE redfish_telemetry_collection_stale_reports_last gauge

# HELP redfish_telemetry_unhandled_report Number of metric values in a metric report that no handler claimed, and which were therefore discarded
# TYPE redfish_telemetry_unhandled_report gauge

# HELP redfish_telemetry_component_health health of a component reported via TelemetryService,1(OK),2(Warning),3(Critical)
# TYPE redfish_telemetry_component_health gauge

# HELP redfish_telemetry_component_health_rollup health rollup of a component reported via TelemetryService,1(OK),2(Warning),3(Critical)
# TYPE redfish_telemetry_component_health_rollup gauge

# HELP redfish_telemetry_cpu_frequency_mhz CPU core frequency in MHz
# TYPE redfish_telemetry_cpu_frequency_mhz gauge

# HELP redfish_telemetry_cpu_memory_controller_frequency_mhz CPU memory controller frequency in MHz
# TYPE redfish_telemetry_cpu_memory_controller_frequency_mhz gauge

# HELP redfish_telemetry_cpu_vreg_voltage_volts CPU voltage regulator output in volts
# TYPE redfish_telemetry_cpu_vreg_voltage_volts gauge

# HELP redfish_telemetry_cpu_memory_page_retirement_count Number of retired memory pages (NVIDIA OEM)
# TYPE redfish_telemetry_cpu_memory_page_retirement_count gauge

# HELP redfish_telemetry_cpu_edp_violation_state Whether the CPU is in an electrical design point violation state, 1(violating),0(nominal) (NVIDIA OEM)
# TYPE redfish_telemetry_cpu_edp_violation_state gauge

# HELP redfish_telemetry_cpu_power_break_performance_state Whether the CPU is in a power-break performance state, 1(active),0(nominal) (NVIDIA OEM)
# TYPE redfish_telemetry_cpu_power_break_performance_state gauge

# HELP redfish_telemetry_port_nvidia_nvlink_data_rx_bandwidth_gbps Per-port NVLink data RX bandwidth in Gbps (NVIDIA OEM)
# TYPE redfish_telemetry_port_nvidia_nvlink_data_rx_bandwidth_gbps gauge
# (also _data_tx_, _raw_rx_ and _raw_tx_ variants)

# HELP redfish_telemetry_conventional_reset_entry_total Total conventional reset entry events
# TYPE redfish_telemetry_conventional_reset_entry_total counter

# HELP redfish_telemetry_conventional_reset_exit_total Total conventional reset exit events
# TYPE redfish_telemetry_conventional_reset_exit_total counter

# HELP redfish_telemetry_cpu_edp_current_limit_watts CPU current EDP (Electrical Design Point) limit in watts
# TYPE redfish_telemetry_cpu_edp_current_limit_watts gauge

# HELP redfish_telemetry_cpu_edp_peak_limit_watts CPU peak EDP (Electrical Design Point) limit in watts
# TYPE redfish_telemetry_cpu_edp_peak_limit_watts gauge

# HELP redfish_telemetry_cpu_energy_joules_total Total CPU energy consumption in joules
# TYPE redfish_telemetry_cpu_energy_joules_total counter

# HELP redfish_telemetry_cpu_power_watts CPU power consumption in watts
# TYPE redfish_telemetry_cpu_power_watts gauge

# HELP redfish_telemetry_cpu_temperature_average_celsius Average CPU temperature in Celsius
# TYPE redfish_telemetry_cpu_temperature_average_celsius gauge

# HELP redfish_telemetry_cpu_temperature_limit_celsius CPU temperature limit in Celsius
# TYPE redfish_telemetry_cpu_temperature_limit_celsius gauge

# HELP redfish_telemetry_fundamental_reset_entry_total Total fundamental reset entry events
# TYPE redfish_telemetry_fundamental_reset_entry_total counter

# HELP redfish_telemetry_fundamental_reset_exit_total Total fundamental reset exit events
# TYPE redfish_telemetry_fundamental_reset_exit_total counter

# HELP redfish_telemetry_gpu_memory_temperature_celsius GPU memory temperature in Celsius
# TYPE redfish_telemetry_gpu_memory_temperature_celsius gauge

# HELP redfish_telemetry_gpu_temperature_celsius GPU core temperature in Celsius
# TYPE redfish_telemetry_gpu_temperature_celsius gauge

# HELP redfish_telemetry_hardware_violation_throttle_duration_seconds_total Total time GPU was throttled due to hardware violations
# TYPE redfish_telemetry_hardware_violation_throttle_duration_seconds_total counter

# HELP redfish_telemetry_irot_reset_exit_total Total IRoT (Internal Root of Trust) reset exit events
# TYPE redfish_telemetry_irot_reset_exit_total counter

# HELP redfish_telemetry_last_reset_type_info Last reset type (1=Conventional, 2=Fundamental, 3=IRoT, 4=PF_FLR)
# TYPE redfish_telemetry_last_reset_type_info gauge

# HELP redfish_telemetry_memory_bandwidth_percent Memory bandwidth utilization percentage
# TYPE redfish_telemetry_memory_bandwidth_percent gauge

# HELP redfish_telemetry_memory_capacity_utilization_percent Memory capacity utilization percentage
# TYPE redfish_telemetry_memory_capacity_utilization_percent gauge

# HELP redfish_telemetry_memory_ecc_correctable_lifetime_total Lifetime correctable DRAM ECC errors
# TYPE redfish_telemetry_memory_ecc_correctable_lifetime_total counter

# HELP redfish_telemetry_memory_ecc_uncorrectable_lifetime_total Lifetime uncorrectable DRAM ECC errors
# TYPE redfish_telemetry_memory_ecc_uncorrectable_lifetime_total counter

# HELP redfish_telemetry_memory_operating_speed_mhz Memory operating speed in MHz
# TYPE redfish_telemetry_memory_operating_speed_mhz gauge

# HELP redfish_telemetry_nvidia_dmma_utilization_percent Double precision Matrix Multiply-Accumulate utilization (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_dmma_utilization_percent gauge

# HELP redfish_telemetry_nvidia_fp16_activity_percent FP16 floating point activity percentage (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_fp16_activity_percent gauge

# HELP redfish_telemetry_nvidia_fp32_activity_percent FP32 floating point activity percentage (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_fp32_activity_percent gauge

# HELP redfish_telemetry_nvidia_fp64_activity_percent FP64 floating point activity percentage (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_fp64_activity_percent gauge

# HELP redfish_telemetry_nvidia_graphics_engine_activity_percent Graphics engine activity percentage (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_graphics_engine_activity_percent gauge

# HELP redfish_telemetry_nvidia_hmma_utilization_percent Half precision Matrix Multiply-Accumulate utilization (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_hmma_utilization_percent gauge

# HELP redfish_telemetry_nvidia_imma_utilization_percent Integer Matrix Multiply-Accumulate utilization (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_imma_utilization_percent gauge

# HELP redfish_telemetry_nvidia_integer_activity_percent Integer operation activity percentage (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_integer_activity_percent gauge

# HELP redfish_telemetry_nvidia_nvdec_instance_utilization_percent Video decoder instance utilization (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_nvdec_instance_utilization_percent gauge

# HELP redfish_telemetry_nvidia_nvdec_utilization_percent Video decoder overall utilization (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_nvdec_utilization_percent gauge

# HELP redfish_telemetry_nvidia_nvjpg_instance_utilization_percent JPEG decoder instance utilization (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_nvjpg_instance_utilization_percent gauge

# HELP redfish_telemetry_nvidia_nvjpg_utilization_percent JPEG decoder overall utilization (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_nvjpg_utilization_percent gauge

# HELP redfish_telemetry_nvidia_nvlink_data_rx_bandwidth_gbps NVLink data receive bandwidth in Gbps (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_nvlink_data_rx_bandwidth_gbps gauge

# HELP redfish_telemetry_nvidia_nvlink_data_tx_bandwidth_gbps NVLink data transmit bandwidth in Gbps (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_nvlink_data_tx_bandwidth_gbps gauge

# HELP redfish_telemetry_nvidia_nvlink_raw_rx_bandwidth_gbps NVLink raw receive bandwidth in Gbps including overhead (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_nvlink_raw_rx_bandwidth_gbps gauge

# HELP redfish_telemetry_nvidia_nvlink_raw_tx_bandwidth_gbps NVLink raw transmit bandwidth in Gbps including overhead (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_nvlink_raw_tx_bandwidth_gbps gauge

# HELP redfish_telemetry_nvidia_nvofa_utilization_percent NVIDIA Optimized Fabrics Adapter utilization (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_nvofa_utilization_percent gauge

# HELP redfish_telemetry_nvidia_pcie_raw_rx_bandwidth_gbps PCIe raw receive bandwidth in Gbps (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_pcie_raw_rx_bandwidth_gbps gauge

# HELP redfish_telemetry_nvidia_pcie_raw_tx_bandwidth_gbps PCIe raw transmit bandwidth in Gbps (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_pcie_raw_tx_bandwidth_gbps gauge

# HELP redfish_telemetry_nvidia_sm_activity_percent Streaming Multiprocessor activity percentage (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_sm_activity_percent gauge

# HELP redfish_telemetry_nvidia_sm_occupancy_percent Streaming Multiprocessor occupancy percentage (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_sm_occupancy_percent gauge

# HELP redfish_telemetry_nvidia_tensor_core_activity_percent Tensor core activity percentage (NVIDIA GPM)
# TYPE redfish_telemetry_nvidia_tensor_core_activity_percent gauge

# HELP redfish_telemetry_pcie_correctable_errors_total Total PCIe correctable errors
# TYPE redfish_telemetry_pcie_correctable_errors_total counter

# HELP redfish_telemetry_pcie_fatal_errors_total Total PCIe fatal errors
# TYPE redfish_telemetry_pcie_fatal_errors_total counter

# HELP redfish_telemetry_pcie_l0_to_recovery_total Total PCIe L0 to recovery transitions
# TYPE redfish_telemetry_pcie_l0_to_recovery_total counter

# HELP redfish_telemetry_pcie_nak_received_total Total PCIe NAK received
# TYPE redfish_telemetry_pcie_nak_received_total counter

# HELP redfish_telemetry_pcie_nak_sent_total Total PCIe NAK sent
# TYPE redfish_telemetry_pcie_nak_sent_total counter

# HELP redfish_telemetry_pcie_nonfatal_errors_total Total PCIe non-fatal errors
# TYPE redfish_telemetry_pcie_nonfatal_errors_total counter

# HELP redfish_telemetry_pcie_replay_rollover_total Total PCIe replay rollover events
# TYPE redfish_telemetry_pcie_replay_rollover_total counter

# HELP redfish_telemetry_pcie_replay_total Total PCIe replay events
# TYPE redfish_telemetry_pcie_replay_total counter

# HELP redfish_telemetry_pf_flr_reset_entry_total Total PF FLR (Physical Function Function-Level Reset) entry events
# TYPE redfish_telemetry_pf_flr_reset_entry_total counter

# HELP redfish_telemetry_pf_flr_reset_exit_total Total PF FLR (Physical Function Function-Level Reset) exit events
# TYPE redfish_telemetry_pf_flr_reset_exit_total counter

# HELP redfish_telemetry_port_nvidia_intentional_link_down_count_total Total intentional link down events (NVIDIA OEM)
# TYPE redfish_telemetry_port_nvidia_intentional_link_down_count_total counter

# HELP redfish_telemetry_port_nvidia_neighbor_mtu_discards_total Total neighbor MTU discards (NVIDIA OEM)
# TYPE redfish_telemetry_port_nvidia_neighbor_mtu_discards_total counter

# HELP redfish_telemetry_port_nvidia_qp1_dropped_total Total QP1 packets dropped (NVIDIA OEM)
# TYPE redfish_telemetry_port_nvidia_qp1_dropped_total counter

# HELP redfish_telemetry_port_nvidia_rx_no_protocol_bytes_total Total RX bytes without protocol (NVIDIA OEM)
# TYPE redfish_telemetry_port_nvidia_rx_no_protocol_bytes_total counter

# HELP redfish_telemetry_port_nvidia_rx_remote_physical_errors_total Total RX remote physical errors (NVIDIA OEM)
# TYPE redfish_telemetry_port_nvidia_rx_remote_physical_errors_total counter

# HELP redfish_telemetry_port_nvidia_rx_switch_relay_errors_total Total RX switch relay errors (NVIDIA OEM)
# TYPE redfish_telemetry_port_nvidia_rx_switch_relay_errors_total counter

# HELP redfish_telemetry_port_nvidia_total_raw_ber Total raw bit error rate (NVIDIA OEM)
# TYPE redfish_telemetry_port_nvidia_total_raw_ber gauge

# HELP redfish_telemetry_port_nvidia_tx_no_protocol_bytes_total Total TX bytes without protocol (NVIDIA OEM)
# TYPE redfish_telemetry_port_nvidia_tx_no_protocol_bytes_total counter

# HELP redfish_telemetry_port_nvidia_tx_wait_total Total TX wait time (NVIDIA OEM)
# TYPE redfish_telemetry_port_nvidia_tx_wait_total counter

# HELP redfish_telemetry_port_nvidia_unintentional_link_down_count_total Total unintentional link down events (NVIDIA OEM)
# TYPE redfish_telemetry_port_nvidia_unintentional_link_down_count_total counter

# HELP redfish_telemetry_port_nvidia_vl15_dropped_total Total VL15 packets dropped (NVIDIA OEM)
# TYPE redfish_telemetry_port_nvidia_vl15_dropped_total counter

# HELP redfish_telemetry_port_nvidia_vl15_tx_bytes_total Total VL15 bytes transmitted (NVIDIA OEM)
# TYPE redfish_telemetry_port_nvidia_vl15_tx_bytes_total counter

# HELP redfish_telemetry_port_nvidia_vl15_tx_packets_total Total VL15 packets transmitted (NVIDIA OEM)
# TYPE redfish_telemetry_port_nvidia_vl15_tx_packets_total counter

# HELP redfish_telemetry_port_rx_bytes_total Total bytes received on port
# TYPE redfish_telemetry_port_rx_bytes_total counter

# HELP redfish_telemetry_port_rx_errors_total Total receive errors on port
# TYPE redfish_telemetry_port_rx_errors_total counter

# HELP redfish_telemetry_port_rx_frames_total Total frames received on port
# TYPE redfish_telemetry_port_rx_frames_total counter

# HELP redfish_telemetry_port_tx_bytes_total Total bytes transmitted on port
# TYPE redfish_telemetry_port_tx_bytes_total counter

# HELP redfish_telemetry_port_tx_discards_total Total transmit discards on port
# TYPE redfish_telemetry_port_tx_discards_total counter

# HELP redfish_telemetry_port_tx_frames_total Total frames transmitted on port
# TYPE redfish_telemetry_port_tx_frames_total counter

# HELP redfish_telemetry_power_throttle_duration_seconds_total Total time GPU was throttled due to power limits
# TYPE redfish_telemetry_power_throttle_duration_seconds_total counter

# HELP redfish_telemetry_software_violation_throttle_duration_seconds_total Total time GPU was throttled due to software violations
# TYPE redfish_telemetry_software_violation_throttle_duration_seconds_total counter

# HELP redfish_telemetry_thermal_throttle_duration_seconds_total Total time GPU was throttled due to thermal limits
# TYPE redfish_telemetry_thermal_throttle_duration_seconds_total counter
```

Exposes no user configuration.

# Extra

## Regenerating the above metrics HELP/TYPE

Maybe a good automation target? Until then, for a given `${TARGET}` representing expected output and given a specific `${MODULE}`, GNU AWK will save the day for wholesale replacing of the above examples:

```shell
curl -s "http://localhost:9610/redfish?target=${TARGET}&module=${MODULE}" | awk '/^# (TYPE|HELP)/ { if (/^# TYPE/) {print; print ""} else {print} }'
``` 
