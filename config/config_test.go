package config

import (
	"bytes"
	"testing"

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
				Modules: map[string]Module{
					"foo": {
						Prober:             "gpu_collector",
						GPUCollector:       GPUCollectorConfig{},
						ChassisCollector:   ChassisCollectorConfig{},
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
				Loglevel: "info",
				Modules: map[string]Module{
					"foo": {
						Prober:             "gpu_collector",
						GPUCollector:       GPUCollectorConfig{},
						ChassisCollector:   ChassisCollectorConfig{},
						ManagerCollector:   ManagerCollectorConfig{},
						SystemCollector:    SystemCollectorConfig{},
						TelemetryCollector: TelemetryCollectorConfig{},
					},
					"boo": {
						Prober:             "chassis_collector",
						GPUCollector:       GPUCollectorConfig{},
						ChassisCollector:   ChassisCollectorConfig{},
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
				Modules: map[string]Module{
					"foo": {},
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
				Modules: map[string]Module{
					"delta_powershelf": {
						Prober: "json_collector",
						JSONCollector: {
							RedfishRoot: "/redfish/v1/Chassis/PowerShelf_0/Sensors",
							JQuery: `      [.Oem.deltaenergysystems.AllSensors.Sensors[]] | map({
        name: (if .DeviceName | test("^ps[0-9]+_") then .DeviceName | sub("^ps[0-9]+_"; "") else .DeviceName end),
        value: .Reading,
        labels: (
          if .DeviceName | test("^ps[0-9]+_") then {"power_supply_id": (.DeviceName | split("_")[0])}
          else {}
          end),
          _raw: .
      }) |
      map(
        .help = "Value yielded from the Redfish API endpoint: " + ._raw.DataSourceUri + ",type: " + ._raw.ReadingType + ",
        unit: " + ._raw.ReadingUnits) |
        map(del(._raw)) | sort_by(.name)`,
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
