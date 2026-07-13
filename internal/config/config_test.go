package config

import (
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
