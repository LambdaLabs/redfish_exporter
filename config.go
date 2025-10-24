package main

import (
	"fmt"
	"io"
	"sync"
	"time"

	"os"

	yaml "gopkg.in/yaml.v3"
)

var (
	DefaultGPUCollector = GPUCollectorConfig{
		CollectionDeadlineDuration: 30 * time.Second,
	}
	DefaultModule = Module{
		GPUCollector: DefaultGPUCollector,
	}
)

type GPUCollectorConfig struct {
	CollectionDeadlineDuration time.Duration `yaml:"collection_deadline"`
}

func (g *GPUCollectorConfig) UnmarshalYAML(unmarshal func(any) error) error {
	*g = DefaultGPUCollector
	type plain GPUCollectorConfig

	if err := unmarshal((*plain)(g)); err != nil {
		return err
	}

	return nil
}

type Module struct {
	GPUCollector GPUCollectorConfig `yaml:"gpu_collector"`
}

func (m *Module) UnmarshalYAML(unmarshal func(any) error) error {
	*m = DefaultModule
	type plain Module
	if err := unmarshal((*plain)(m)); err != nil {
		return err
	}
	return nil
}

type Config struct {
	Hosts    map[string]HostConfig `yaml:"hosts"`
	Groups   map[string]HostConfig `yaml:"groups"`
	Loglevel string                `yaml:"loglevel"`
	Modules  map[string]Module     `yaml:"modules"`
}

func (c *Config) UnmarshalYAML(unmarshal func(any) error) error {
	type plain Config
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

type SafeConfig struct {
	sync.RWMutex
	Config *Config
}

type HostConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Read exporter config from file
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

func (sc *SafeConfig) AppLogLevel() string {
	sc.Lock()
	defer sc.Unlock()
	logLevel := sc.Config.Loglevel
	if logLevel != "" {
		return logLevel
	}
	return "info"
}
