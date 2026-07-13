package config

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv                     string
	PublicBaseURL              string
	HTTPPort                   int
	DatabaseURL                string
	DatabaseMaxConns           int
	DatabaseMinConns           int
	DatabaseMaxConnLifetime    time.Duration
	DatabaseStatementTimeout   time.Duration
	DatabaseTransactionTimeout time.Duration
	ReservationTTL             time.Duration
	WorkerBatchSize            int
	WorkerPollInterval         time.Duration
	RateLimitRequests          int
	RateLimitBurst             int
	LogLevel                   string
	CORSAllowedOrigins         []string
	TrustedProxyCIDRs          []netip.Prefix
	AdminAPIKeyHash            []byte
	DemoMode                   bool
	DemoMaxConcurrency         int
	DemoResetEnabled           bool
	PublicMutationsEnabled     bool
	RunWorker                  bool
	RunMigrations              bool
	PublicDemoSeedEnabled      bool
	ShutdownTimeout            time.Duration
	DemoToken                  string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:             value("APP_ENV", "development"),
		PublicBaseURL:      strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		LogLevel:           value("LOG_LEVEL", "info"),
		CORSAllowedOrigins: split(value("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
		DemoToken:          strings.TrimSpace(os.Getenv("DEMO_TOKEN")),
	}

	var err error
	if cfg.HTTPPort, err = intValue("HTTP_PORT", 8080, 1, 65535); err != nil {
		return Config{}, err
	}
	if cfg.ReservationTTL, err = durationValue("RESERVATION_TTL", 2*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMaxConns, err = intValue("DATABASE_MAX_CONNS", 8, 1, 100); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMinConns, err = intValue("DATABASE_MIN_CONNS", 1, 0, 100); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMinConns > cfg.DatabaseMaxConns {
		return Config{}, errors.New("DATABASE_MIN_CONNS cannot exceed DATABASE_MAX_CONNS")
	}
	if cfg.DatabaseMaxConnLifetime, err = durationValue("DATABASE_MAX_CONN_LIFETIME", 30*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseStatementTimeout, err = durationValue("DATABASE_STATEMENT_TIMEOUT", 5*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseTransactionTimeout, err = durationValue("DATABASE_TRANSACTION_TIMEOUT", 10*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.WorkerBatchSize, err = intValue("WORKER_BATCH_SIZE", 50, 1, 1000); err != nil {
		return Config{}, err
	}
	if cfg.WorkerPollInterval, err = durationValue("WORKER_POLL_INTERVAL", time.Second); err != nil {
		return Config{}, err
	}
	if cfg.RateLimitRequests, err = intValue("RATE_LIMIT_REQUESTS", 20, 1, 100000); err != nil {
		return Config{}, err
	}
	if cfg.RateLimitBurst, err = intValue("RATE_LIMIT_BURST", 40, 1, 100000); err != nil {
		return Config{}, err
	}
	if cfg.DemoMaxConcurrency, err = intValue("DEMO_MAX_CONCURRENCY", 20, 1, 100); err != nil {
		return Config{}, err
	}
	if cfg.ShutdownTimeout, err = durationValue("SHUTDOWN_TIMEOUT", 10*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.DemoMode, err = boolValue("DEMO_MODE", cfg.AppEnv == "development"); err != nil {
		return Config{}, err
	}
	if cfg.DemoResetEnabled, err = boolValue("DEMO_RESET_ENABLED", cfg.AppEnv == "development"); err != nil {
		return Config{}, err
	}
	if cfg.PublicMutationsEnabled, err = boolValue("PUBLIC_MUTATIONS_ENABLED", cfg.AppEnv == "development"); err != nil {
		return Config{}, err
	}
	if cfg.RunWorker, err = boolValue("RUN_WORKER", false); err != nil {
		return Config{}, err
	}
	if cfg.RunMigrations, err = boolValue("RUN_MIGRATIONS", false); err != nil {
		return Config{}, err
	}
	if cfg.PublicDemoSeedEnabled, err = boolValue("PUBLIC_DEMO_SEED_ENABLED", false); err != nil {
		return Config{}, err
	}
	if cfg.TrustedProxyCIDRs, err = prefixes(os.Getenv("TRUSTED_PROXY_CIDRS")); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if len(cfg.CORSAllowedOrigins) == 0 {
		return Config{}, errors.New("CORS_ALLOWED_ORIGINS must contain at least one origin")
	}
	if cfg.Production() {
		if err := cfg.validateProduction(); err != nil {
			return Config{}, err
		}
	}
	return cfg, nil
}

func (c Config) Production() bool { return c.AppEnv == "production" }

func (c *Config) validateProduction() error {
	base, err := url.Parse(c.PublicBaseURL)
	if err != nil || base.Scheme != "https" || base.Host == "" {
		return errors.New("PUBLIC_BASE_URL must be an absolute HTTPS URL in production")
	}
	databaseURL, err := url.Parse(c.DatabaseURL)
	if err != nil || (databaseURL.Query().Get("sslmode") != "require" && databaseURL.Query().Get("sslmode") != "verify-full") {
		return errors.New("DATABASE_URL must require TLS in production")
	}
	for _, origin := range c.CORSAllowedOrigins {
		parsed, parseErr := url.Parse(origin)
		if origin == "*" || parseErr != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return errors.New("CORS_ALLOWED_ORIGINS must contain only absolute HTTPS origins in production")
		}
	}
	if c.LogLevel == "debug" {
		return errors.New("LOG_LEVEL=debug is not allowed in production")
	}
	if c.DemoMode || c.DemoResetEnabled {
		return errors.New("DEMO_MODE and DEMO_RESET_ENABLED must be false in production")
	}
	rawHash := strings.TrimSpace(os.Getenv("ADMIN_API_KEY_HASH"))
	decoded, err := hex.DecodeString(rawHash)
	if err != nil || len(decoded) != sha256.Size {
		return errors.New("ADMIN_API_KEY_HASH must be a SHA-256 hex digest in production")
	}
	for _, weak := range []string{"admin", "password", "stockrush-local-demo", "change-me"} {
		sum := sha256.Sum256([]byte(weak))
		if string(decoded) == string(sum[:]) {
			return errors.New("ADMIN_API_KEY_HASH must not use a weak or default key")
		}
	}
	c.AdminAPIKeyHash = decoded
	return nil
}

func value(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}

func intValue(name string, fallback, min, max int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < min || v > max {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", name, min, max)
	}
	return v, nil
}

func durationValue(name string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	v, err := time.ParseDuration(raw)
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return v, nil
}

func boolValue(name string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", name)
	}
	return v, nil
}

func prefixes(raw string) ([]netip.Prefix, error) {
	values := split(raw)
	out := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return nil, fmt.Errorf("TRUSTED_PROXY_CIDRS contains invalid CIDR %q", value)
		}
		if prefix.Bits() == 0 {
			return nil, errors.New("TRUSTED_PROXY_CIDRS cannot trust the entire internet")
		}
		out = append(out, prefix)
	}
	return out, nil
}

func split(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
