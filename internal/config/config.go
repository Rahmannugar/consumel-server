package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"net"
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
	environmentPrefix   = "CONSUMEL_"
	defaultHTTPPort     = 8080
	defaultPostgresPort = 5432
	localSessionCookie  = "consumel_session"
	secureSessionCookie = "__Host-consumel_session"
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
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

type Redis struct {
	URL string
}

type Auth struct {
	BaseURL            string
	ClientBaseURL      string
	TrustedOrigins     []string
	TrustedProxies     []string
	OTPHMACSecret      []byte
	GoogleClientID     string
	GoogleClientSecret string
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
		Database: Database{
			Port:    defaultPostgresPort,
			SSLMode: "disable",
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
	cfg.Database.Host = strings.TrimSpace(k.String("database.host"))
	if k.Exists("database.port") {
		port, err := strconv.Atoi(k.String("database.port"))
		if err != nil {
			return Config{}, fmt.Errorf("CONSUMEL_DATABASE_PORT must be an integer: %w", err)
		}
		cfg.Database.Port = port
	}
	cfg.Database.Name = strings.TrimSpace(k.String("database.name"))
	cfg.Database.User = strings.TrimSpace(k.String("database.user"))
	cfg.Database.Password = k.String("database.password")
	if k.Exists("database.ssl_mode") {
		cfg.Database.SSLMode = strings.TrimSpace(k.String("database.ssl_mode"))
	}
	cfg.Redis.URL = k.String("redis.url")
	cfg.Auth.BaseURL = k.String("auth.base_url")
	cfg.Auth.ClientBaseURL = k.String("auth.client_base_url")
	cfg.Auth.TrustedOrigins = commaSeparated(k.String("auth.trusted_origins"))
	cfg.Auth.TrustedProxies = commaSeparated(k.String("auth.trusted_proxies"))
	cfg.Auth.GoogleClientID = strings.TrimSpace(k.String("auth.google_client_id"))
	cfg.Auth.GoogleClientSecret = strings.TrimSpace(k.String("auth.google_client_secret"))
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
	if cfg.Database.Host == "" {
		return fmt.Errorf("CONSUMEL_DATABASE_HOST is required")
	}
	if cfg.Database.Port < 1 || cfg.Database.Port > 65535 {
		return fmt.Errorf("CONSUMEL_DATABASE_PORT must be between 1 and 65535")
	}
	if cfg.Database.Name == "" {
		return fmt.Errorf("CONSUMEL_DATABASE_NAME is required")
	}
	if cfg.Database.User == "" {
		return fmt.Errorf("CONSUMEL_DATABASE_USER is required")
	}
	if cfg.Database.Password == "" {
		return fmt.Errorf("CONSUMEL_DATABASE_PASSWORD is required")
	}
	switch cfg.Database.SSLMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("CONSUMEL_DATABASE_SSL_MODE must be a supported PostgreSQL SSL mode")
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
	if cfg.Environment == EnvironmentProduction && baseURL.Scheme != "https" {
		return fmt.Errorf("CONSUMEL_AUTH_BASE_URL must use HTTPS in production")
	}
	clientBaseURL, err := url.Parse(strings.TrimSpace(cfg.Auth.ClientBaseURL))
	if err != nil || clientBaseURL.Host == "" ||
		(clientBaseURL.Scheme != "http" && clientBaseURL.Scheme != "https") ||
		clientBaseURL.User != nil || clientBaseURL.Path != "" || clientBaseURL.RawQuery != "" || clientBaseURL.Fragment != "" {
		return fmt.Errorf("CONSUMEL_AUTH_CLIENT_BASE_URL must be an HTTP or HTTPS origin without a path")
	}
	if cfg.Environment == EnvironmentProduction && clientBaseURL.Scheme != "https" {
		return fmt.Errorf("CONSUMEL_AUTH_CLIENT_BASE_URL must use HTTPS in production")
	}
	if len(cfg.Auth.OTPHMACSecret) < 32 {
		return fmt.Errorf("CONSUMEL_AUTH_OTP_HMAC_SECRET must decode to at least 32 bytes")
	}
	if (cfg.Auth.GoogleClientID == "") != (cfg.Auth.GoogleClientSecret == "") {
		return fmt.Errorf("CONSUMEL_AUTH_GOOGLE_CLIENT_ID and CONSUMEL_AUTH_GOOGLE_CLIENT_SECRET must be configured together")
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

func (cfg Auth) PasswordResetURL() string {
	return strings.TrimRight(cfg.ClientBaseURL, "/") + "/reset-password"
}

func (cfg Auth) GoogleSuccessURL() string {
	return strings.TrimRight(cfg.ClientBaseURL, "/") + "/auth/complete"
}

func (cfg Auth) GoogleEnabled() bool {
	return cfg.GoogleClientID != "" && cfg.GoogleClientSecret != ""
}

// SessionCookieName uses the host-only prefix in production. Browsers require
// __Host- cookies to be Secure, use Path=/, and omit the Domain attribute.
func (cfg Config) SessionCookieName() string {
	if cfg.Environment == EnvironmentProduction {
		return secureSessionCookie
	}
	return localSessionCookie
}

func (cfg HTTP) Address() string {
	return ":" + strconv.Itoa(cfg.Port)
}

func (cfg Database) ConnectionString() string {
	connection := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Path:   cfg.Name,
	}
	query := connection.Query()
	query.Set("sslmode", cfg.SSLMode)
	connection.RawQuery = query.Encode()
	return connection.String()
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
