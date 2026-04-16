package agent

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds agent configuration
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Auth       AuthConfig       `yaml:"auth"`
	Agent      AgentConfig      `yaml:"agent"`
	Heartbeat  HeartbeatConfig `yaml:"heartbeat"`
	Reconnect  ReconnectConfig  `yaml:"reconnect"`
	Log        LogConfig        `yaml:"log"`
}

// ServerConfig holds server connection configuration
type ServerConfig struct {
	Address string     `yaml:"address"`
	TLS    TLSConfig  `yaml:"tls"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled bool   `yaml:"enabled"`
	CA      string `yaml:"ca"` // CA certificate to verify server
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Key string `yaml:"key"`
}

// AgentConfig holds agent identification
type AgentConfig struct {
	Hostname string `yaml:"hostname"` // empty for auto-detect
	OS       string `yaml:"os"`       // empty for auto-detect
}

// HeartbeatConfig holds heartbeat configuration
type HeartbeatConfig struct {
	Interval int `yaml:"interval"` // heartbeat interval in seconds
}

// ReconnectConfig holds reconnection configuration
type ReconnectConfig struct {
	Interval   int `yaml:"interval"`    // initial reconnect interval in seconds
	MaxAttempts int  `yaml:"max_attempts"` // max reconnect attempts, 0 = infinite
	MaxInterval int  `yaml:"max_interval"` // max reconnect interval in seconds
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level string `yaml:"level"` // debug/info/warn/error
	File  string `yaml:"file"`  // empty for stdout
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.Server.Address == "" {
		config.Server.Address = "localhost:8080"
	}
	if config.Heartbeat.Interval == 0 {
		config.Heartbeat.Interval = 30
	}
	if config.Reconnect.Interval == 0 {
		config.Reconnect.Interval = 5
	}
	if config.Reconnect.MaxInterval == 0 {
		config.Reconnect.MaxInterval = 60
	}
	if config.Log.Level == "" {
		config.Log.Level = "info"
	}

	return &config, nil
}
