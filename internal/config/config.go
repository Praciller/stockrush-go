package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv             string
	HTTPPort           int
	DatabaseURL        string
	ReservationTTL     time.Duration
	WorkerBatchSize    int
	WorkerPollInterval time.Duration
	RateLimitRequests  int
	RateLimitBurst     int
	LogLevel           string
	CORSAllowedOrigins []string
	DemoToken          string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:             value("APP_ENV", "development"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		LogLevel:           value("LOG_LEVEL", "info"),
		CORSAllowedOrigins: split(value("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
		DemoToken:          value("DEMO_TOKEN", "stockrush-local-demo"),
	}

	var err error
	if cfg.HTTPPort, err = intValue("HTTP_PORT", 8080, 1, 65535); err != nil {
		return Config{}, err
	}
	if cfg.ReservationTTL, err = durationValue("RESERVATION_TTL", 2*time.Minute); err != nil {
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
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if len(cfg.CORSAllowedOrigins) == 0 {
		return Config{}, errors.New("CORS_ALLOWED_ORIGINS must contain at least one origin")
	}
	return cfg, nil
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
