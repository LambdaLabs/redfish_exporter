package config

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	gta "gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestConfigFromFile(t *testing.T) {
	configFile := "testdata/config.example.yml"

	config, err := NewConfigFromFile(configFile)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	assert.Equal(t, "info", config.Loglevel)
	assert.Equal(t, config.Hosts["default"], HostConfig{Username: "user", Password: "pass"})
	assert.Equal(t, config.Groups["group1"], HostConfig{Username: "group1_user", Password: "group1_pass"})
	assert.NotNil(t, config.Modules["foo"])
	assert.NotNil(t, config.Modules["foo"].GPUCollector)
	assert.Equal(t, "gpu_collector", config.Modules["foo"].Prober)
}

func TestRedfishClientConfig(t *testing.T) {
	tT := map[string]struct {
		inputYAML     string
		wantErrString string
		wantConfig    *Config
	}{
		"happy path, no user config": {
			inputYAML: `
loglevel: info
`,
			wantErrString: "",
			wantConfig: &Config{
				Loglevel:      "info",
				RedfishClient: DefaultRedfishConfig,
			},
		},
		"happy path, just one value changed": {
			inputYAML: `
redfish_client:
  max_concurrent_requests: 100
`,
			wantErrString: "",
			wantConfig: &Config{
				RedfishClient: RedfishClientConfig{
					MaxConcurrentRequests: 100,
					DialTimeout:           10 * time.Second,
				},
			},
		},
		"happy path, all values changed": {
			inputYAML: `
redfish_client:
  max_concurrent_requests: 100
  dial_timeout: 30s
`,
			wantErrString: "",
			wantConfig: &Config{
				RedfishClient: RedfishClientConfig{
					MaxConcurrentRequests: 100,
					DialTimeout:           30 * time.Second,
				},
			},
		},
	}
	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			byteReader := bytes.NewReader([]byte(test.inputYAML))
			gotConfig, err := readConfigFrom(byteReader)
			if test.wantErrString != "" {
				gta.ErrorContains(t, err, test.wantErrString)
			}
			gta.Assert(t, cmp.DeepEqual(test.wantConfig, gotConfig))
		})
	}
}

func TestModulesConfig(t *testing.T) {
	tT := map[string]struct {
		inputYAML     string
		wantErrString string
		wantConfig    *Config
	}{
		"GPU Collector exists": {
			inputYAML: `
modules:
  foo:
    prober: gpu_collector
    gpu_collector:
`,
			wantErrString: "",
			wantConfig: &Config{
				RedfishClient: DefaultRedfishConfig,
				Modules: map[string]Module{
					"foo": {
						Prober:           "gpu_collector",
						GPUCollector:     GPUCollectorConfig{},
						ChassisCollector: ChassisCollectorConfig{},
						JSONCollector: JSONCollectorConfig{
							Timeout: 30 * time.Second,
						},
						ManagerCollector:   ManagerCollectorConfig{},
						SystemCollector:    SystemCollectorConfig{},
						TelemetryCollector: TelemetryCollectorConfig{},
					},
				},
			},
		},
		"Unmarshal of multiple modules and with log level looks OK": {
			inputYAML: `
loglevel: info
modules:
  foo:
    prober: gpu_collector
    gpu_collector:
  boo:
    prober: chassis_collector
    chassis_collector:
`,
			wantErrString: "",
			wantConfig: &Config{
				Loglevel:      "info",
				RedfishClient: DefaultRedfishConfig,
				Modules: map[string]Module{
					"foo": {
						Prober:           "gpu_collector",
						GPUCollector:     GPUCollectorConfig{},
						ChassisCollector: ChassisCollectorConfig{},
						JSONCollector: JSONCollectorConfig{
							Timeout: 30 * time.Second,
						},
						ManagerCollector:   ManagerCollectorConfig{},
						SystemCollector:    SystemCollectorConfig{},
						TelemetryCollector: TelemetryCollectorConfig{},
					},
					"boo": {
						Prober:           "chassis_collector",
						GPUCollector:     GPUCollectorConfig{},
						ChassisCollector: ChassisCollectorConfig{},
						JSONCollector: JSONCollectorConfig{
							Timeout: 30 * time.Second,
						},
						ManagerCollector:   ManagerCollectorConfig{},
						SystemCollector:    SystemCollectorConfig{},
						TelemetryCollector: TelemetryCollectorConfig{},
					},
				},
			},
		},
		"erroneous config returns error": {
			inputYAML:     `foo:bar:baz`,
			wantErrString: "unmarshal errors:\n  line 1: cannot unmarshal !!str",
			wantConfig:    &Config{},
		},
		"modules require a prober field": {
			inputYAML: `
modules:
  foo:
    gpu_collector:
`,
			wantErrString: "module foo is not valid: module requires a prober to be configured",
			wantConfig: &Config{
				RedfishClient: DefaultRedfishConfig,
				Modules: map[string]Module{
					"foo": {
						JSONCollector: JSONCollectorConfig{
							Timeout: 30 * time.Second,
						},
					},
				},
			},
		},
	}
	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			byteReader := bytes.NewReader([]byte(test.inputYAML))
			gotConfig, err := readConfigFrom(byteReader)
			if test.wantErrString != "" {
				gta.ErrorContains(t, err, test.wantErrString)
			}
			gta.Assert(t, cmp.DeepEqual(test.wantConfig, gotConfig))
		})
	}
}

func TestModulesConfig_JSONCollector(t *testing.T) {
	tT := map[string]struct {
		inputFile     string
		wantErrString string
		wantConfig    *Config
	}{
		"extraction from Delta Electronics OEM Sensors endpoint": {
			inputFile:     "testdata/config.j2m.yaml",
			wantErrString: "",
			wantConfig: &Config{
				RedfishClient: DefaultRedfishConfig,
				Modules: map[string]Module{
					"rf_version": {
						Prober: "json_collector",
						JSONCollector: JSONCollectorConfig{
							Timeout:     30 * time.Second,
							RedfishRoot: `/redfish/v1`,
							JQFilter: `[{
  "name": "redfish_version",
  "help": "Redfish version reported by this device",
  "labels": {
    "redfish_version": .RedfishVersion
  },
  "value": 1.0
}]`,
						},
					},
					"delta_powershelf": {
						Prober: "json_collector",
						JSONCollector: JSONCollectorConfig{
							Timeout:     30 * time.Second,
							RedfishRoot: "/redfish/v1/Chassis/PowerShelf_0/Sensors?$expand=.($levels=1)",
							JQFilter: `[.Oem.deltaenergysystems.AllSensors.Sensors[]] |
map({
  name: (if .DeviceName | test("^ps[0-9]+_") then .DeviceName | sub("^ps[0-9]+_"; "") else .DeviceName end),
  value: .Reading,
  labels: (
    if .DeviceName | test("^ps[0-9]+_") then {"power_supply_id": (.DeviceName | split("_")[0])}
    else {}
    end),
    _raw: .
}) |
map(.help = "Value yielded from the Redfish API /Chassis/PowerShelf_0, expanded 1 level") | map(del(._raw)) | sort_by(.name)`,
						},
						GPUCollector:       GPUCollectorConfig{},
						ChassisCollector:   ChassisCollectorConfig{},
						ManagerCollector:   ManagerCollectorConfig{},
						SystemCollector:    SystemCollectorConfig{},
						TelemetryCollector: TelemetryCollectorConfig{},
					},
				},
			},
		},
		"config with a non-standard deadline": {
			inputFile:     "testdata/json_collector_with_timeout.yaml",
			wantErrString: "",
			wantConfig: &Config{
				RedfishClient: DefaultRedfishConfig,
				Modules: map[string]Module{
					"31s_timeout_collector": {
						Prober: "json_collector",
						JSONCollector: JSONCollectorConfig{
							Timeout:     31 * time.Second,
							RedfishRoot: "/redfish/v1/nop",
							JQFilter:    `nil`,
						},
					},
				},
			},
		},
	}
	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			gotConfig, err := NewConfigFromFile(test.inputFile)
			if test.wantErrString != "" {
				gta.ErrorContains(t, err, test.wantErrString)
			}
			gta.Assert(t, cmp.DeepEqual(test.wantConfig, gotConfig))
		})
	}
}
