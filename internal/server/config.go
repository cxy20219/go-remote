package server

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds server configuration
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Auth      AuthConfig      `yaml:"auth"`
	Heartbeat HeartbeatConfig `yaml:"heartbeat"`
	Log       LogConfig       `yaml:"log"`
}

// ServerConfig holds WebSocket server configuration
type ServerConfig struct {
	Host string      `yaml:"host"`
	Port int         `yaml:"port"`
	TLS  TLSConfig  `yaml:"tls"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled bool   `yaml:"enabled"`
	Cert    string `yaml:"cert"`
	Key     string `yaml:"key"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Key string `yaml:"key"`
}

// HeartbeatConfig holds heartbeat configuration
type HeartbeatConfig struct {
	Interval int `yaml:"interval"` // heartbeat check interval in seconds
	Timeout  int `yaml:"timeout"`  // heartbeat timeout in seconds
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
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Heartbeat.Interval == 0 {
		config.Heartbeat.Interval = 30
	}
	if config.Heartbeat.Timeout == 0 {
		config.Heartbeat.Timeout = 90
	}
	if config.Log.Level == "" {
		config.Log.Level = "info"
	}

	return &config, nil
}
