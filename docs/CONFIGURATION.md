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

# HELP redfish_chassis_leak_detector_health chassis leak detector health state,1(OK),2(Warning),3(Critical)
# TYPE redfish_chassis_leak_detector_health gauge

# HELP redfish_chassis_model_info organization responsible for producing the chassis, the name by which the manufacturer generally refers to the chassis, and a part number and sku assigned by the organization that is responsible for producing or manufacturing the chassis
# TYPE redfish_chassis_model_info gauge

# HELP redfish_chassis_network_adapter_state chassis network adapter state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating)
# TYPE redfish_chassis_network_adapter_state gauge

# HELP redfish_chassis_state state of chassis,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating)
# TYPE redfish_chassis_state gauge
```

Exposes no user configuration.

### `<gpu_collector>`
[source](../collector/gpu_collector.go)

The GPU Collector primarily exposes a wealth of Nvidia GPU-related data using a mix of Oem and Processor API data.
**Note:** This collector iterates many Redfish paths and depending on network round trips, may take a significant amount of time (30+ seconds) to return data.

Against a Lambda lab system, the collector yields the following timeseries:

```
# HELP redfish_gpu_context_utilization_seconds_total Accumulated GPU context utilization duration in seconds
# TYPE redfish_gpu_context_utilization_seconds_total counter

# HELP redfish_gpu_health health of gpu reported by system,1(OK),2(Warning),3(Critical)
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

# HELP redfish_gpu_processor_health GPU processor health,1(OK),2(Warning),3(Critical)
# TYPE redfish_gpu_processor_health gauge

# HELP redfish_gpu_processor_state GPU processor state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating)
# TYPE redfish_gpu_processor_state gauge

# HELP redfish_gpu_processor_total_cores GPU processor total cores
# TYPE redfish_gpu_processor_total_cores gauge

# HELP redfish_gpu_processor_total_threads GPU processor total threads
# TYPE redfish_gpu_processor_total_threads gauge

# HELP redfish_gpu_sram_ecc_error_threshold_exceeded GPU SRAM ECC error threshold exceeded (1 if exceeded)
# TYPE redfish_gpu_sram_ecc_error_threshold_exceeded gauge
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
