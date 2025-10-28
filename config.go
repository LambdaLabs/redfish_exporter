package main

import (
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"os"

	"github.com/LambdaLabs/redfish_exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
	yaml "gopkg.in/yaml.v3"
)

var (
	// DefaultGPUCollector is a default unless the user provides particular values.
	DefaultGPUCollector = GPUCollectorConfig{
		CollectionDeadlineDuration: 30 * time.Second,
	}
	// DefaultChassisCollector is a default unless the user provides particular values.
	DefaultChassisCollector = ChassisCollectorConfig{
		CollectionDeadlineDuration: 30 * time.Second,
	}
	// DefaultManagerCollector is a default unless the user provides particular values.
	DefaultManagerCollector = ManagerCollectorConfig{
		CollectionDeadlineDuration: 30 * time.Second,
	}
	// DefaultSystemCollector is a default unless the user provides particular values.
	DefaultSystemCollector = SystemCollectorConfig{
		CollectionDeadlineDuration: 30 * time.Second,
	}
	// DefaultTelemetryCollector is a default unless the user provides particular values.
	DefaultTelemetryCollector = TelemetryCollectorConfig{
		CollectionDeadlineDuration: 30 * time.Second,
	}
	// DefaultModule is a default Module
	DefaultModule = Module{
		ChassisCollector:   DefaultChassisCollector,
		GPUCollector:       DefaultGPUCollector,
		ManagerCollector:   DefaultManagerCollector,
		SystemCollector:    DefaultSystemCollector,
		TelemetryCollector: DefaultTelemetryCollector,
	}
	// DefaultModuleConfig is used as a default when building a collector slice.
	// In a future release, this will be removed and users will be expected to
	// define one or more modules in their config, and reference those as HTTP query params.
	DefaultModuleConfig = map[string]Module{
		"gpu_collector": {
			Prober:       "gpu_collector",
			GPUCollector: DefaultGPUCollector,
		},
		"chassis_collector": {
			Prober:           "chassis_collector",
			ChassisCollector: DefaultChassisCollector,
		},
		"manager_collector": {
			Prober:           "manager_collector",
			ManagerCollector: DefaultManagerCollector,
		},
		"system_collector": {
			Prober:          "system_collector",
			SystemCollector: DefaultSystemCollector,
		},
		"telemetry_collector": {
			Prober:             "telemetry_collector",
			TelemetryCollector: DefaultTelemetryCollector,
		},
	}
)

// ChassisCollectorConfig is a prober configuration.
type ChassisCollectorConfig struct {
	CollectionDeadlineDuration time.Duration `yaml:"collection_deadline"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (c *ChassisCollectorConfig) UnmarshalYAML(unmarshal func(any) error) error {
	*c = DefaultChassisCollector
	type plain ChassisCollectorConfig

	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	return nil
}

// GPUCollectorConfig is a prober configuration.
type GPUCollectorConfig struct {
	CollectionDeadlineDuration time.Duration `yaml:"collection_deadline"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (g *GPUCollectorConfig) UnmarshalYAML(unmarshal func(any) error) error {
	*g = DefaultGPUCollector
	type plain GPUCollectorConfig

	if err := unmarshal((*plain)(g)); err != nil {
		return err
	}

	return nil
}

// ManagerCollectorConfig is a prober configuration.
type ManagerCollectorConfig struct {
	CollectionDeadlineDuration time.Duration `yaml:"collection_deadline"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (m *ManagerCollectorConfig) UnmarshalYAML(unmarshal func(any) error) error {
	*m = DefaultManagerCollector
	type plain ManagerCollectorConfig

	if err := unmarshal((*plain)(m)); err != nil {
		return err
	}

	return nil
}

// SystemCollectorConfig is a prober configuration.
type SystemCollectorConfig struct {
	CollectionDeadlineDuration time.Duration `yaml:"collection_deadline"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (s *SystemCollectorConfig) UnmarshalYAML(unmarshal func(any) error) error {
	*s = DefaultSystemCollector
	type plain SystemCollectorConfig

	if err := unmarshal((*plain)(s)); err != nil {
		return err
	}

	return nil
}

// TelemetryCollectorConfig is a prober configuration.
type TelemetryCollectorConfig struct {
	CollectionDeadlineDuration time.Duration `yaml:"collection_deadline"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (t *TelemetryCollectorConfig) UnmarshalYAML(unmarshal func(any) error) error {
	*t = DefaultTelemetryCollector
	type plain TelemetryCollectorConfig

	if err := unmarshal((*plain)(t)); err != nil {
		return err
	}

	return nil
}

// Module is a struct which represents some particular behavior the redfish_exporter should have
// when executed against a host.
// Modules are expected to specify a 'prober', and then a particular collector.
type Module struct {
	Prober             string                   `yaml:"prober"`
	GPUCollector       GPUCollectorConfig       `yaml:"gpu_collector"`
	ChassisCollector   ChassisCollectorConfig   `yaml:"chassis_collector"`
	ManagerCollector   ManagerCollectorConfig   `yaml:"manager_collector"`
	SystemCollector    SystemCollectorConfig    `yaml:"system_collector"`
	TelemetryCollector TelemetryCollectorConfig `yaml:"telemetry_collector"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (m *Module) UnmarshalYAML(unmarshal func(any) error) error {
	*m = DefaultModule
	type plain Module
	if err := unmarshal((*plain)(m)); err != nil {
		return err
	}
	if m.Prober == "" {
		return fmt.Errorf("modules require a prober to be set")
	}
	return nil
}

// Collector creates and returns a prometheus.Collector from the Module, based on the Prober.
// Both redfish client and logger are common dependencies for any redfish_exporter collector,
// and must be provided as inputs.
func (m *Module) Collector(rfClient *gofish.APIClient, logger *slog.Logger) prometheus.Collector {
	switch m.Prober {
	case "gpu_collector":
		return collector.NewGPUCollector(rfClient, logger)
	case "chassis_collector":
		return collector.NewChassisCollector(rfClient, logger)
	case "manager_collector":
		return collector.NewManagerCollector(rfClient, logger)
	case "system_collector":
		return collector.NewSystemCollector(rfClient, logger)
	case "telemetry_collector":
		return collector.NewTelemetryCollector(rfClient, logger)
	default:
	}
	return nil
}

// Config represents the redfish_exporter config file
type Config struct {
	Hosts    map[string]HostConfig `yaml:"hosts"`
	Groups   map[string]HostConfig `yaml:"groups"`
	Loglevel string                `yaml:"loglevel"`
	Modules  map[string]Module     `yaml:"modules"`
}

// UnmarshalYAML is a custom YAML unmarshaler.
// It is heavily inspired by blackbox_exporter.
func (c *Config) UnmarshalYAML(unmarshal func(any) error) error {
	type plain Config
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

// SafeConfig is a mutex-enabled Config.
type SafeConfig struct {
	sync.RWMutex
	Config *Config
}

// GetModules exposes the modules map from this SafeConfig
func (sc *SafeConfig) GetModules() map[string]Module {
	return sc.Config.Modules
}

// HostConfig holds the Redfish Username/Password for a host or group of hosts
type HostConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Read exporter config from an input file path.
func NewConfigFromFile(configFilePath string) (*Config, error) {
	file, err := os.Open(configFilePath)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	return readConfigFrom(file)
}

func readConfigFrom(r io.Reader) (*Config, error) {
	config := &Config{}
	if err := yaml.NewDecoder(r).Decode(config); err != nil {
		return config, err
	}

	return config, nil
}

// ReloadConfig reads a given configuration file.
// If successfully read, the SafeConfig mutex is obtained and config structure rebuilt.
func (sc *SafeConfig) ReloadConfig(configFile string) error {
	var c, err = NewConfigFromFile(configFile)
	if err != nil {
		return err
	}

	sc.Lock()
	sc.Config = c
	sc.Unlock()

	return nil
}

// HostConfigForTarget safely looks up a specific target auth configuration from the config file.
func (sc *SafeConfig) HostConfigForTarget(target string) (*HostConfig, error) {
	sc.Lock()
	defer sc.Unlock()
	if hostConfig, ok := sc.Config.Hosts[target]; ok {
		return &HostConfig{
			Username: hostConfig.Username,
			Password: hostConfig.Password,
		}, nil
	}
	if hostConfig, ok := sc.Config.Hosts["default"]; ok {
		return &HostConfig{
			Username: hostConfig.Username,
			Password: hostConfig.Password,
		}, nil
	}
	return &HostConfig{}, fmt.Errorf("no credentials found for target %s", target)
}

// HostConfigForGroup checks the configuration for a matching group config and returns the configured HostConfig for
// that matched group.
func (sc *SafeConfig) HostConfigForGroup(group string) (*HostConfig, error) {
	sc.Lock()
	defer sc.Unlock()
	if hostConfig, ok := sc.Config.Groups[group]; ok {
		return &hostConfig, nil
	}
	return &HostConfig{}, fmt.Errorf("no credentials found for group %s", group)
}

// AppLogLevel applies a log level to the application.
func (sc *SafeConfig) AppLogLevel() string {
	sc.Lock()
	defer sc.Unlock()
	logLevel := sc.Config.Loglevel
	if logLevel != "" {
		return logLevel
	}
	return "info"
}
