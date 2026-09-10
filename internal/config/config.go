package config

import (
	"fmt"
	"strconv"
	"strings"

	koanfenv "github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/v2"
)

const (
	environmentPrefix = "CONSUMEL_"
	defaultHTTPPort   = 8080
)

type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentTest        Environment = "test"
	EnvironmentStaging     Environment = "staging"
	EnvironmentProduction  Environment = "production"
)

type Config struct {
	Environment Environment
	HTTP        HTTP
}

type HTTP struct {
	Port int
}

func Load() (Config, error) {
	k := koanf.New(".")
	provider := koanfenv.Provider(".", koanfenv.Opt{
		Prefix: environmentPrefix,
		TransformFunc: func(key, value string) (string, any) {
			key = strings.ToLower(strings.TrimPrefix(key, environmentPrefix))
			group, setting, grouped := strings.Cut(key, "_")
			if grouped {
				key = group + "." + setting
			}
			return key, value
		},
	})
	if err := k.Load(provider, nil); err != nil {
		return Config{}, fmt.Errorf("load environment: %w", err)
	}

	cfg := Config{
		Environment: EnvironmentDevelopment,
		HTTP: HTTP{
			Port: defaultHTTPPort,
		},
	}

	if k.Exists("environment") {
		cfg.Environment = Environment(k.String("environment"))
	}
	if k.Exists("http.port") {
		port, err := strconv.Atoi(k.String("http.port"))
		if err != nil {
			return Config{}, fmt.Errorf("CONSUMEL_HTTP_PORT must be an integer: %w", err)
		}
		cfg.HTTP.Port = port
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (cfg Config) Validate() error {
	switch cfg.Environment {
	case EnvironmentDevelopment, EnvironmentTest, EnvironmentStaging, EnvironmentProduction:
	default:
		return fmt.Errorf("CONSUMEL_ENVIRONMENT must be development, test, staging, or production")
	}

	if cfg.HTTP.Port < 1 || cfg.HTTP.Port > 65535 {
		return fmt.Errorf("CONSUMEL_HTTP_PORT must be between 1 and 65535")
	}

	return nil
}

func (cfg HTTP) Address() string {
	return ":" + strconv.Itoa(cfg.Port)
}
