package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	libconfig "drivepower/backend/libs/config"
)

// Config represents service configuration loaded from YAML/env.
type Config struct {
	HTTP struct {
		Port string `yaml:"port" env:"AUTH_HTTP_PORT"`
	} `yaml:"http"`
	Database struct {
		DSN string `yaml:"dsn" env:"AUTH_POSTGRES_DSN"`
	} `yaml:"database"`
	JWT struct {
		Secret           string `yaml:"secret" env:"AUTH_JWT_SECRET"`
		ExpiresInMinutes int    `yaml:"expiresInMinutes" env:"AUTH_JWT_EXPIRES_MINUTES"`
	} `yaml:"jwt"`
}

// Load reads configuration using the shared config loader.
func Load() (*Config, error) {
	cfg := &Config{
		HTTP: struct {
			Port string `yaml:"port" env:"AUTH_HTTP_PORT"`
		}{
			Port: "8080",
		},
		JWT: struct {
			Secret           string `yaml:"secret" env:"AUTH_JWT_SECRET"`
			ExpiresInMinutes int    `yaml:"expiresInMinutes" env:"AUTH_JWT_EXPIRES_MINUTES"`
		}{
			ExpiresInMinutes: 60,
		},
	}

	if err := libconfig.LoadConfig(cfg); err != nil {
		return nil, err
	}

	if cfg.Database.DSN == "" {
		return nil, errors.New("config: database DSN is required")
	}
	if cfg.JWT.Secret == "" {
		return nil, errors.New("config: jwt secret is required")
	}
	if cfg.JWT.ExpiresInMinutes <= 0 {
		cfg.JWT.ExpiresInMinutes = 60
	}

	return cfg, nil
}

// HTTPAddress ensures we always return host:port formatted string.
func (c *Config) HTTPAddress() string {
	port := strings.TrimSpace(c.HTTP.Port)
	if port == "" {
		port = "8080"
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return fmt.Sprintf(":%s", port)
}

// JWTExpiration converts configured expiry to duration.
func (c *Config) JWTExpiration() time.Duration {
	if c.JWT.ExpiresInMinutes <= 0 {
		return time.Hour
	}
	return time.Duration(c.JWT.ExpiresInMinutes) * time.Minute
}

