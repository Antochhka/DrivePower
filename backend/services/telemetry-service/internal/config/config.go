package config

import (
	"errors"
	"fmt"
	"strings"

	libconfig "drivepower/backend/libs/config"
)

// Config defines telemetry service configuration.
type Config struct {
	HTTP struct {
		Port string `yaml:"port" env:"TELEMETRY_HTTP_PORT"`
	} `yaml:"http"`
	Database struct {
		DSN string `yaml:"dsn" env:"TELEMETRY_POSTGRES_DSN"`
	} `yaml:"database"`
}

// Load configuration using shared helper.
func Load() (*Config, error) {
	cfg := &Config{
		HTTP: struct {
			Port string `yaml:"port" env:"TELEMETRY_HTTP_PORT"`
		}{
			Port: "8084",
		},
	}

	if err := libconfig.LoadConfig(cfg); err != nil {
		return nil, err
	}

	if strings.TrimSpace(cfg.Database.DSN) == "" {
		return nil, errors.New("config: database dsn required")
	}
	return cfg, nil
}

// HTTPAddress returns :port style.
func (c *Config) HTTPAddress() string {
	port := strings.TrimSpace(c.HTTP.Port)
	if port == "" {
		port = "8084"
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return fmt.Sprintf(":%s", port)
}

