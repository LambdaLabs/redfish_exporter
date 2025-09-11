package main

import (
	"fmt"
	"sync"

	"os"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Hosts    map[string]HostConfig `yaml:"hosts"`
	Groups   map[string]HostConfig `yaml:"groups"`
	Loglevel string                `yaml:"loglevel"`
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
func NewConfigFromFile(configFile string) (*Config, error) {
	var config = &Config{}
	yamlFile, err := os.ReadFile(configFile) //nolint:gosec // Config path must be user-specifiable for deployment
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(yamlFile, config); err != nil {
		return nil, err
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
