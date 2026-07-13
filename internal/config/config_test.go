package config

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

func TestLoadRejectsInvalidReservationTTL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://stockrush:stockrush@localhost:5432/stockrush?sslmode=disable")
	t.Setenv("RESERVATION_TTL", "not-a-duration")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid RESERVATION_TTL error")
	}
}

func TestLoadUsesSafeDevelopmentDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://stockrush:stockrush@localhost:5432/stockrush?sslmode=disable")
	t.Setenv("RESERVATION_TTL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPPort != 8080 || cfg.ReservationTTL != 2*time.Minute {
		t.Fatalf("Load() = port %d ttl %s, want port 8080 ttl 2m", cfg.HTTPPort, cfg.ReservationTTL)
	}
}

func TestProductionFailsClosedWithoutRequiredConfiguration(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://app:secret@example.com/stockrush?sslmode=require")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "PUBLIC_BASE_URL") {
		t.Fatalf("Load() error = %v, want missing PUBLIC_BASE_URL", err)
	}
}

func TestProductionAcceptsExplicitSecureConfiguration(t *testing.T) {
	hash := sha256.Sum256([]byte("correct-horse-battery-staple-public-demo-admin-key"))
	for name, value := range map[string]string{
		"APP_ENV":                      "production",
		"PUBLIC_BASE_URL":              "https://example.hf.space",
		"DATABASE_URL":                 "postgres://app:secret@example.com/stockrush?sslmode=require",
		"DATABASE_MAX_CONNS":           "8",
		"DATABASE_MIN_CONNS":           "1",
		"DATABASE_MAX_CONN_LIFETIME":   "30m",
		"DATABASE_STATEMENT_TIMEOUT":   "5s",
		"DATABASE_TRANSACTION_TIMEOUT": "10s",
		"CORS_ALLOWED_ORIGINS":         "https://praciller.github.io",
		"TRUSTED_PROXY_CIDRS":          "10.0.0.0/8,172.16.0.0/12",
		"ADMIN_API_KEY_HASH":           hex.EncodeToString(hash[:]),
		"DEMO_MODE":                    "false",
		"DEMO_MAX_CONCURRENCY":         "20",
		"DEMO_RESET_ENABLED":           "false",
		"PUBLIC_MUTATIONS_ENABLED":     "true",
		"RESERVATION_TTL":              "2m",
		"WORKER_BATCH_SIZE":            "50",
		"WORKER_POLL_INTERVAL":         "1s",
		"RATE_LIMIT_REQUESTS":          "10",
		"RATE_LIMIT_BURST":             "20",
		"LOG_LEVEL":                    "info",
		"SHUTDOWN_TIMEOUT":             "10s",
	} {
		t.Setenv(name, value)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Production() || cfg.DatabaseMaxConns != 8 || !cfg.PublicMutationsEnabled {
		t.Fatalf("Load() = %+v, want secure production configuration", cfg)
	}
}

func TestProductionRejectsInsecureValues(t *testing.T) {
	hash := sha256.Sum256([]byte("correct-horse-battery-staple-public-demo-admin-key"))
	base := map[string]string{
		"APP_ENV": "production", "PUBLIC_BASE_URL": "https://example.hf.space",
		"DATABASE_URL":         "postgres://app:secret@example.com/stockrush?sslmode=require",
		"CORS_ALLOWED_ORIGINS": "https://praciller.github.io", "ADMIN_API_KEY_HASH": hex.EncodeToString(hash[:]),
		"DEMO_MODE": "false", "DEMO_RESET_ENABLED": "false", "LOG_LEVEL": "info",
	}
	for name, value := range base {
		t.Setenv(name, value)
	}

	for _, test := range []struct{ name, value string }{
		{"DATABASE_URL", "postgres://app:secret@example.com/stockrush?sslmode=disable"},
		{"CORS_ALLOWED_ORIGINS", "*"},
		{"LOG_LEVEL", "debug"},
		{"DEMO_MODE", "true"},
		{"DEMO_RESET_ENABLED", "true"},
		{"ADMIN_API_KEY_HASH", "not-a-sha256-hash"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(test.name, test.value)
			if _, err := Load(); err == nil {
				t.Fatalf("Load() accepted %s=%q", test.name, test.value)
			}
		})
	}
}
