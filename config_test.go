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
    gpu_collector:
      collection_deadline: 60s
`,
			wantErrString: "",
			wantConfig: &Config{
				Modules: map[string]Module{
					"foo": {
						GPUCollector: GPUCollectorConfig{
							CollectionDeadlineDuration: 60 * time.Second,
						},
					},
				},
			},
		},
		"GPU Collector without deadline gets a default": {
			inputYAML: `
modules:
  foo:
    gpu_collector:
`,
			wantErrString: "",
			wantConfig: &Config{
				Modules: map[string]Module{
					"foo": {
						GPUCollector: GPUCollectorConfig{
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
    gpu_collector:
  boo:
    gpu_collector:
      collection_deadline: 10s
`,
			wantErrString: "",
			wantConfig: &Config{
				Loglevel: "info",
				Modules: map[string]Module{
					"foo": {
						GPUCollector: GPUCollectorConfig{
							CollectionDeadlineDuration: 30 * time.Second,
						},
					},
					"boo": {
						GPUCollector: GPUCollectorConfig{
							CollectionDeadlineDuration: 10 * time.Second,
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
