package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	libconfig "drivepower/backend/libs/config"
)

// Config defines OCPP server configuration.
type Config struct {
	HTTP struct {
		Port string `yaml:"port" env:"OCPP_HTTP_PORT"`
	} `yaml:"http"`
	Database struct {
		DSN string `yaml:"dsn" env:"OCPP_POSTGRES_DSN"`
	} `yaml:"database"`
	Services struct {
		SessionsURL string `yaml:"sessionsUrl" env:"SESSIONS_SERVICE_URL"`
		BillingURL  string `yaml:"billingUrl" env:"BILLING_SERVICE_URL"`
	} `yaml:"services"`
	WebSocket struct {
		PingIntervalSeconds int `yaml:"pingIntervalSeconds" env:"OCPP_PING_INTERVAL"`
		WriteTimeoutSeconds int `yaml:"writeTimeoutSeconds" env:"OCPP_WRITE_TIMEOUT"`
	} `yaml:"websocket"`
}

// Load uses shared config loader and validates required fields.
func Load() (*Config, error) {
	cfg := &Config{
		HTTP: struct {
			Port string `yaml:"port" env:"OCPP_HTTP_PORT"`
		}{
			Port: "8081",
		},
		WebSocket: struct {
			PingIntervalSeconds int `yaml:"pingIntervalSeconds" env:"OCPP_PING_INTERVAL"`
			WriteTimeoutSeconds int `yaml:"writeTimeoutSeconds" env:"OCPP_WRITE_TIMEOUT"`
		}{
			PingIntervalSeconds: 30,
			WriteTimeoutSeconds: 15,
		},
	}

	if err := libconfig.LoadConfig(cfg); err != nil {
		return nil, err
	}

	if strings.TrimSpace(cfg.Database.DSN) == "" {
		return nil, errors.New("config: database DSN is required")
	}

	return cfg, nil
}

// HTTPAddress returns :port style address.
func (c *Config) HTTPAddress() string {
	port := strings.TrimSpace(c.HTTP.Port)
	if port == "" {
		port = "8081"
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return fmt.Sprintf(":%s", port)
}

// PingInterval returns websocket ping interval.
func (c *Config) PingInterval() time.Duration {
	if c.WebSocket.PingIntervalSeconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.WebSocket.PingIntervalSeconds) * time.Second
}

// WriteTimeout returns websocket write timeout.
func (c *Config) WriteTimeout() time.Duration {
	if c.WebSocket.WriteTimeoutSeconds <= 0 {
		return 15 * time.Second
	}
	return time.Duration(c.WebSocket.WriteTimeoutSeconds) * time.Second
}

