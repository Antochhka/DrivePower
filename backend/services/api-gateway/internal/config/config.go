package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	libconfig "drivepower/backend/libs/config"
)

// Config defines gateway configuration.
type Config struct {
	HTTP struct {
		Port string `yaml:"port" env:"API_GATEWAY_HTTP_PORT"`
	} `yaml:"http"`
	JWT struct {
		Secret string `yaml:"secret" env:"API_GATEWAY_JWT_SECRET"`
	} `yaml:"jwt"`
	Services struct {
		AuthURL     string `yaml:"authUrl" env:"AUTH_SERVICE_URL"`
		SessionsURL string `yaml:"sessionsUrl" env:"SESSIONS_SERVICE_URL"`
		BillingURL  string `yaml:"billingUrl" env:"BILLING_SERVICE_URL"`
		StationsURL string `yaml:"stationsUrl" env:"STATIONS_SERVICE_URL"`
	} `yaml:"services"`
	HTTPClient struct {
		TimeoutSeconds int `yaml:"timeoutSeconds" env:"API_GATEWAY_HTTP_TIMEOUT"`
	} `yaml:"httpClient"`
}

// Load configuration via shared helper.
func Load() (*Config, error) {
	cfg := &Config{
		HTTP: struct {
			Port string `yaml:"port" env:"API_GATEWAY_HTTP_PORT"`
		}{
			Port: "8080",
		},
		HTTPClient: struct {
			TimeoutSeconds int `yaml:"timeoutSeconds" env:"API_GATEWAY_HTTP_TIMEOUT"`
		}{
			TimeoutSeconds: 5,
		},
	}

	if err := libconfig.LoadConfig(cfg); err != nil {
		return nil, err
	}

	if strings.TrimSpace(cfg.JWT.Secret) == "" {
		return nil, errors.New("config: jwt secret required")
	}
	return cfg, nil
}

// HTTPAddress returns :port style.
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

// HTTPTimeout returns http client timeout.
func (c *Config) HTTPTimeout() time.Duration {
	if c.HTTPClient.TimeoutSeconds <= 0 {
		return 5 * time.Second
	}
	return time.Duration(c.HTTPClient.TimeoutSeconds) * time.Second
}

