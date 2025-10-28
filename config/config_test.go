package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	gta "gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestConfigFromFile(t *testing.T) {
	configFile := "config.example.yml"

	config, err := NewConfigFromFile(configFile)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	assert.Equal(t, "info", config.Loglevel)
	assert.Equal(t, config.Hosts["default"], HostConfig{Username: "user", Password: "pass"})
	assert.Equal(t, config.Groups["group1"], HostConfig{Username: "group1_user", Password: "group1_pass"})
}

func TestModulesConfig(t *testing.T) {
	tT := map[string]struct {
		inputYAML     string
		wantErrString string
		wantConfig    *Config
	}{
		"GPU Collector with a deadline": {
			inputYAML: `
modules:
  foo:
    prober: gpu_collector
    gpu_collector:
      collection_deadline: 60s
`,
			wantErrString: "",
			wantConfig: &Config{
				Modules: map[string]Module{
					"foo": {
						Prober: "gpu_collector",
						GPUCollector: GPUCollectorConfig{
							CollectionDeadlineDuration: 60 * time.Second,
						},
						ChassisCollector: ChassisCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
						ManagerCollector: ManagerCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
						SystemCollector: SystemCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
						TelemetryCollector: TelemetryCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
					},
				},
			},
		},
		"GPU Collector without deadline gets a default": {
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
						Prober: "gpu_collector",
						GPUCollector: GPUCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
						ChassisCollector: ChassisCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
						ManagerCollector: ManagerCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
						SystemCollector: SystemCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
						TelemetryCollector: TelemetryCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
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
    prober: gpu_collector
    chassis_collector:
      collection_deadline: 10s
`,
			wantErrString: "",
			wantConfig: &Config{
				Loglevel: "info",
				Modules: map[string]Module{
					"foo": {
						Prober: "gpu_collector",
						GPUCollector: GPUCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
						ChassisCollector: ChassisCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
						ManagerCollector: ManagerCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
						SystemCollector: SystemCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
						TelemetryCollector: TelemetryCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
					},
					"boo": {
						Prober: "gpu_collector",
						GPUCollector: GPUCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
						ChassisCollector: ChassisCollectorConfig{
							CollectionDeadlineDuration: 10 * time.Second,
						},
						ManagerCollector: ManagerCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
						SystemCollector: SystemCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
						TelemetryCollector: TelemetryCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
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
			wantErrString: "modules require a prober to be set",
			wantConfig: &Config{
				Modules: map[string]Module{},
			},
		},
	}
	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			byteReader := bytes.NewReader([]byte(test.inputYAML))
			gotConfig, err := readConfigFrom(byteReader)
			gta.Assert(t, cmp.DeepEqual(test.wantConfig, gotConfig))
			if test.wantErrString != "" {
				gta.ErrorContains(t, err, test.wantErrString)
			}
		})
	}
}
