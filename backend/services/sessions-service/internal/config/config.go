package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	libconfig "drivepower/backend/libs/config"
)

// Config defines sessions service configuration.
type Config struct {
	HTTP struct {
		Port string `yaml:"port" env:"SESSIONS_HTTP_PORT"`
	} `yaml:"http"`
	Database struct {
		DSN string `yaml:"dsn" env:"SESSIONS_POSTGRES_DSN"`
	} `yaml:"database"`
	Redis struct {
		Addr     string `yaml:"addr" env:"SESSIONS_REDIS_ADDR"`
		Password string `yaml:"password" env:"SESSIONS_REDIS_PASSWORD"`
		DB       int    `yaml:"db" env:"SESSIONS_REDIS_DB"`
		TTL      int    `yaml:"ttlSeconds" env:"SESSIONS_REDIS_TTL"`
	} `yaml:"redis"`
}

// Load reads configuration via shared helper.
func Load() (*Config, error) {
	cfg := &Config{
		HTTP: struct {
			Port string `yaml:"port" env:"SESSIONS_HTTP_PORT"`
		}{
			Port: "8082",
		},
		Redis: struct {
			Addr     string `yaml:"addr" env:"SESSIONS_REDIS_ADDR"`
			Password string `yaml:"password" env:"SESSIONS_REDIS_PASSWORD"`
			DB       int    `yaml:"db" env:"SESSIONS_REDIS_DB"`
			TTL      int    `yaml:"ttlSeconds" env:"SESSIONS_REDIS_TTL"`
		}{
			Addr: "localhost:6379",
			TTL:  86400,
		},
	}

	if err := libconfig.LoadConfig(cfg); err != nil {
		return nil, err
	}

	if strings.TrimSpace(cfg.Database.DSN) == "" {
		return nil, errors.New("config: database dsn required")
	}
	if strings.TrimSpace(cfg.Redis.Addr) == "" {
		return nil, errors.New("config: redis addr required")
	}
	return cfg, nil
}

// HTTPAddress returns :port style.
func (c *Config) HTTPAddress() string {
	port := strings.TrimSpace(c.HTTP.Port)
	if port == "" {
		port = "8082"
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return fmt.Sprintf(":%s", port)
}

// ActiveSessionTTL returns ttl as duration.
func (c *Config) ActiveSessionTTL() time.Duration {
	if c.Redis.TTL <= 0 {
		return 24 * time.Hour
	}
	return time.Duration(c.Redis.TTL) * time.Second
}

