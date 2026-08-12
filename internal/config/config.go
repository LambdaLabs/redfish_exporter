package config

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

var (
	// DefaultSensorExclude drops per-core CPU utilisation from the chassis Sensors pass.
	//
	// A GB300 tray publishes 144 of these (ProcessorModule_N_CPU_0_CoreUtil_M), more series
	// than the rest of the chassis collector produces for that tray combined, and
	// node_exporter already reports the same thing in-band at higher resolution.
	//
	// Expressed as configuration rather than a rule in the collector, which never infers
	// meaning from a sensor's name. Set `sensor_exclude: ""` to collect them.
	DefaultSensorExclude = `_CoreUtil_[0-9]+$`
	// DefaultChassisCollector is a default unless the user provides particular values.
	DefaultChassisCollector = ChassisCollectorConfig{}
	// DefaultGPUCollector is a default unless the user provides particular values.
	DefaultGPUCollector = GPUCollectorConfig{}
	//DefaultJSONCollector is a default unless the user provides particular values.
	DefaultJSONCollector = JSONCollectorConfig{
		Timeout: 30 * time.Second,
	}
	// DefaultManagerCollector is a default unless the user provides particular values.
	DefaultManagerCollector = ManagerCollectorConfig{}
	// DefaultSystemCollector is a default unless the user provides particular values.
	DefaultSystemCollector = SystemCollectorConfig{}
	// DefaultTelemetryCollector is a default unless the user provides particular values.
	DefaultTelemetryCollector = TelemetryCollectorConfig{}
	// DefaultModuleConfig is used as a default when building a collector slice.
	// In a future release, this will be removed and users will be expected to
	// define one or more modules in their config, and reference those as HTTP query params.
	DefaultModuleConfig = map[string]Module{
		"chassis_collector": {
			Prober:           "chassis_collector",
			ChassisCollector: DefaultChassisCollector,
		},
		"gpu_collector": {
			Prober:       "gpu_collector",
			GPUCollector: DefaultGPUCollector,
		},
		"json_collector": {
			Prober:        "json_collector",
			JSONCollector: DefaultJSONCollector,
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
		// leak_detection is a chassis_collector narrowed to coolant leak monitoring, so
		// that leak state can be scraped on a much shorter interval than a full chassis
		// scrape allows. See docs/CONFIGURATION.md.
		//
		// Disabling Thermal and Power together is load-bearing rather than incidental:
		// on the liquid-cooled platforms this targets no chassis implements either, so
		// the Sensors pass would otherwise stand in for what was just disabled. Setting
		// both narrows that pass to the leak detectors.
		//
		// Deliberately not in the rf_exporter_default bundle, and deliberately unfiltered
		// — chassis naming is vendor specific. Set chassis_include to match only the
		// chassis carrying detectors ("^Chassis_[0-9]+$" on a Supermicro NVL72 tray,
		// "^MGX_BMC_[0-9]+$" on an MGX NVSwitch tray) for a large further saving.
		"leak_detection": {
			Prober: "chassis_collector",
			ChassisCollector: ChassisCollectorConfig{
				DisableThermal:         true,
				DisablePower:           true,
				DisableNetworkAdapters: true,
			},
		},
	}
	DefaultRedfishConfig = RedfishClientConfig{
		MaxConcurrentRequests: 1,
		DialTimeout:           10 * time.Second,
	}
)

// newViperInstance returns a Viper instance configured with AutomaticEnv.
// If envPrefix is non-empty, it is set as the env prefix (e.g. "MYPREFIX"
// causes MYPREFIX_LOGLEVEL to map to loglevel).
// The key replacer maps "." → "_" so nested keys like
// "redfish_client.dial_timeout" map to env var "REDFISH_CLIENT_DIAL_TIMEOUT".
func newViperInstance(envPrefix string) *viper.Viper {
	v := viper.New()
	if envPrefix != "" {
		v.SetEnvPrefix(envPrefix)
	}
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("loglevel", "")
	v.SetDefault("shutdown_timeout", 60*time.Second)
	v.SetDefault("redfish_client.max_concurrent_requests", DefaultRedfishConfig.MaxConcurrentRequests)
	v.SetDefault("redfish_client.dial_timeout", DefaultRedfishConfig.DialTimeout)

	return v
}

// unmarshalViperConfig decodes the viper instance into a Config, applying
// defaults for zero values and running validation.
func unmarshalViperConfig(v *viper.Viper) (*Config, error) {
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, err
	}

	// Apply defaults for JSONCollector timeout — cannot use SetDefault since
	// module names are not known upfront.
	for name, mod := range config.Modules {
		if mod.JSONCollector.Timeout == 0 {
			mod.JSONCollector.Timeout = DefaultJSONCollector.Timeout
			config.Modules[name] = mod
		}
	}

	return config, config.Validate()
}

// ChassisCollectorConfig is a prober configuration. See docs/CONFIGURATION.md.
//
// Every field's zero value preserves the collector's historical behaviour, which keeps
// `chassis_collector: {}` a no-op. SensorExclude is the exception; see below.
type ChassisCollectorConfig struct {
	// ChassisInclude, when non-empty, is a regular expression matched against each
	// chassis Id. Only matching chassis are collected. Applied before ChassisExclude.
	ChassisInclude string `mapstructure:"chassis_include"`
	// ChassisExclude, when non-empty, is a regular expression matched against each
	// chassis Id. Matching chassis are skipped.
	ChassisExclude string `mapstructure:"chassis_exclude"`
	// SensorExclude is a regular expression matched against each Sensor Id. Matching
	// sensors emit no metrics.
	//
	// It is a pointer because an absent key and an empty pattern have to mean different
	// things: absent is "whatever ships by default", empty is "exclude nothing". Read it
	// through SensorExcludePattern rather than directly.
	//
	// Leak detector sensors are never excluded by it. Silently dropping a safety signal
	// through a broad pattern is not a tradeoff worth offering.
	SensorExclude *string `mapstructure:"sensor_exclude"`

	// DisableThermal skips the legacy Thermal schema (temperatures and fans).
	DisableThermal bool `mapstructure:"disable_thermal"`
	// DisableThermalSubsystem skips ThermalSubsystem, and with it leak detection.
	DisableThermalSubsystem bool `mapstructure:"disable_thermal_subsystem"`
	// DisablePower skips the legacy Power schema (voltages and power supplies).
	DisablePower bool `mapstructure:"disable_power"`
	// DisableNetworkAdapters skips NetworkAdapters and their NetworkPorts. This is
	// often the most expensive subsystem, at one request per adapter plus one per port.
	DisableNetworkAdapters bool `mapstructure:"disable_network_adapters"`
	// DisableSensors skips the Sensors collection entirely, including the leak detector
	// voltages.
	//
	// Setting DisableThermal and DisablePower together instead narrows the Sensors pass
	// to the leak detectors: an operator who has opted out of bulk thermal and power data
	// has not asked for it back under a different schema.
	DisableSensors bool `mapstructure:"disable_sensors"`
}

// SensorExcludePattern returns the sensor exclusion pattern in effect, substituting
// DefaultSensorExclude when the key was not configured at all.
//
// Defaulting here rather than at unmarshal time means every construction path agrees: a
// module parsed from YAML, a built-in module and a zero-value struct all behave the same.
func (c ChassisCollectorConfig) SensorExcludePattern() string {
	if c.SensorExclude == nil {
		return DefaultSensorExclude
	}
	return *c.SensorExclude
}

// GPUCollectorConfig is a prober configuration.
type GPUCollectorConfig struct{}

type JSONCollectorConfig struct {
	Timeout     time.Duration `mapstructure:"context_timeout"`
	RedfishRoot string        `mapstructure:"redfishRoot"`
	JQFilter    string        `mapstructure:"jq"`
}

// ManagerCollectorConfig is a prober configuration.
type ManagerCollectorConfig struct{}

// SystemCollectorConfig is a prober configuration.
type SystemCollectorConfig struct{}

// TelemetryCollectorConfig is a prober configuration.
type TelemetryCollectorConfig struct{}

// PowershelfCollectorConfig is a prober configuration
type PowershelfCollectorConfig struct{}

// Module is a struct which represents some particular behavior the redfish_exporter should have
// when executed against a host.
// Modules are expected to specify a 'prober', and then a particular collector.
type Module struct {
	Prober              string                    `mapstructure:"prober"`
	ChassisCollector    ChassisCollectorConfig    `mapstructure:"chassis_collector"`
	GPUCollector        GPUCollectorConfig        `mapstructure:"gpu_collector"`
	JSONCollector       JSONCollectorConfig       `mapstructure:"json_collector"`
	ManagerCollector    ManagerCollectorConfig    `mapstructure:"manager_collector"`
	SystemCollector     SystemCollectorConfig     `mapstructure:"system_collector"`
	TelemetryCollector  TelemetryCollectorConfig  `mapstructure:"telemetry_collector"`
	PowershelfCollector PowershelfCollectorConfig `mapstructure:"powershelf_collector"`
}

func (m *Module) Validate() error {
	if m.Prober == "" {
		return fmt.Errorf("module requires a prober to be configured")
	}

	return nil
}

// RedfishClientConfig describes configuration passed to the gofish API client for Redfish
type RedfishClientConfig struct {
	// MaxConcurrentRequests sets the gofish client maximum permitted concurrent requests.
	MaxConcurrentRequests int64         `mapstructure:"max_concurrent_requests"`
	DialTimeout           time.Duration `mapstructure:"dial_timeout"`
}

// Config represents the redfish_exporter config file
type Config struct {
	Hosts           map[string]HostConfig `mapstructure:"hosts"`
	Groups          map[string]HostConfig `mapstructure:"groups"`
	Loglevel        string                `mapstructure:"loglevel"`
	ShutdownTimeout time.Duration         `mapstructure:"shutdown_timeout"`
	Modules         map[string]Module     `mapstructure:"modules"`
	RedfishClient   RedfishClientConfig   `mapstructure:"redfish_client"`
}

// Validate runs end-of-parsing validations against a Config.
func (c *Config) Validate() error {
	for modName, mod := range c.Modules {
		if err := mod.Validate(); err != nil {
			return fmt.Errorf("module %s is not valid: %w", modName, err)
		}
	}
	return nil
}

// SafeConfig is a mutex-enabled Config.
type SafeConfig struct {
	sync.RWMutex
	Config *Config
	v      *viper.Viper
}

// NewSafeConfig creates a SafeConfig with a viper instance configured once.
func NewSafeConfig(envPrefix string) *SafeConfig {
	return &SafeConfig{
		Config: &Config{},
		v:      newViperInstance(envPrefix),
	}
}

// GetModules exposes the modules map from this SafeConfig
func (sc *SafeConfig) GetModules() map[string]Module {
	return sc.Config.Modules
}

// HostConfig holds the Redfish Username/Password for a host or group of hosts
type HostConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// ReloadConfig reads a given configuration file.
// If successfully read, the SafeConfig mutex is obtained and config structure rebuilt.
func (sc *SafeConfig) ReloadConfig(configFile string) error {
	sc.v.SetConfigFile(configFile)
	if err := sc.v.ReadInConfig(); err != nil {
		return err
	}
	c, err := unmarshalViperConfig(sc.v)
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

func (sc *SafeConfig) RedfishClientConfig() RedfishClientConfig {
	sc.Lock()
	defer sc.Unlock()
	return sc.Config.RedfishClient
}
