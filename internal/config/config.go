package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"net/mail"
	"net/url"
	"strconv"
	"strings"

	"github.com/knadh/koanf/parsers/dotenv"
	koanfenv "github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const (
	environmentPrefix = "CONSUMEL_"
	defaultHTTPPort   = 8080
)

type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentProduction  Environment = "production"
)

type Config struct {
	Environment Environment
	HTTP        HTTP
	Database    Database
	Redis       Redis
	Auth        Auth
	Resend      Resend
}

type HTTP struct {
	Port int
}

type Database struct {
	URL string
}

type Redis struct {
	URL string
}

type Auth struct {
	BaseURL        string
	TrustedOrigins []string
	TrustedProxies []string
	OTPHMACSecret  []byte
}

type Resend struct {
	APIKey      string
	NoReplyFrom string
	HelloFrom   string
}

func Load() (Config, error) {
	k := koanf.New(".")
	transform := func(key, value string) (string, any) {
		key = strings.ToLower(strings.TrimPrefix(key, environmentPrefix))
		group, setting, grouped := strings.Cut(key, "_")
		if grouped {
			key = group + "." + setting
		}
		return key, value
	}

	if err := k.Load(
		file.Provider(".env"),
		dotenv.ParserEnvWithValue(environmentPrefix, ".", transform),
	); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	provider := koanfenv.Provider(".", koanfenv.Opt{
		Prefix:        environmentPrefix,
		TransformFunc: transform,
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
	cfg.Database.URL = k.String("database.url")
	cfg.Redis.URL = k.String("redis.url")
	cfg.Auth.BaseURL = k.String("auth.base_url")
	cfg.Auth.TrustedOrigins = commaSeparated(k.String("auth.trusted_origins"))
	cfg.Auth.TrustedProxies = commaSeparated(k.String("auth.trusted_proxies"))
	cfg.Resend.APIKey = k.String("resend.api_key")
	cfg.Resend.NoReplyFrom = k.String("resend.noreply_from")
	cfg.Resend.HelloFrom = k.String("resend.hello_from")
	if encodedSecret := strings.TrimSpace(k.String("auth.otp_hmac_secret")); encodedSecret != "" {
		secret, err := base64.StdEncoding.DecodeString(encodedSecret)
		if err != nil {
			return Config{}, fmt.Errorf("CONSUMEL_AUTH_OTP_HMAC_SECRET must be base64 encoded: %w", err)
		}
		cfg.Auth.OTPHMACSecret = secret
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (cfg Config) Validate() error {
	switch cfg.Environment {
	case EnvironmentDevelopment, EnvironmentProduction:
	default:
		return fmt.Errorf("CONSUMEL_ENVIRONMENT must be development or production")
	}

	if cfg.HTTP.Port < 1 || cfg.HTTP.Port > 65535 {
		return fmt.Errorf("CONSUMEL_HTTP_PORT must be between 1 and 65535")
	}
	if strings.TrimSpace(cfg.Database.URL) == "" {
		return fmt.Errorf("CONSUMEL_DATABASE_URL is required")
	}
	if strings.TrimSpace(cfg.Redis.URL) == "" {
		return fmt.Errorf("CONSUMEL_REDIS_URL is required")
	}
	if _, err := url.ParseRequestURI(cfg.Redis.URL); err != nil {
		return fmt.Errorf("CONSUMEL_REDIS_URL must be a valid URL: %w", err)
	}
	baseURL, err := url.Parse(strings.TrimSpace(cfg.Auth.BaseURL))
	if err != nil || baseURL.Host == "" ||
		(baseURL.Scheme != "http" && baseURL.Scheme != "https") ||
		baseURL.User != nil || baseURL.Path != "" || baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return fmt.Errorf("CONSUMEL_AUTH_BASE_URL must be an HTTP or HTTPS origin without a path")
	}
	if len(cfg.Auth.OTPHMACSecret) < 32 {
		return fmt.Errorf("CONSUMEL_AUTH_OTP_HMAC_SECRET must decode to at least 32 bytes")
	}
	if strings.TrimSpace(cfg.Resend.APIKey) == "" {
		return fmt.Errorf("CONSUMEL_RESEND_API_KEY is required")
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(cfg.Resend.NoReplyFrom)); err != nil {
		return fmt.Errorf("CONSUMEL_RESEND_NOREPLY_FROM must be a valid email sender: %w", err)
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(cfg.Resend.HelloFrom)); err != nil {
		return fmt.Errorf("CONSUMEL_RESEND_HELLO_FROM must be a valid email sender: %w", err)
	}

	return nil
}

func (cfg HTTP) Address() string {
	return ":" + strconv.Itoa(cfg.Port)
}

func commaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
