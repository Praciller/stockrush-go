package database

import (
	"testing"
	"time"
)

func TestPoolConfigAppliesProductionBounds(t *testing.T) {
	cfg, err := PoolConfig("postgres://app:secret@example.com/stockrush?sslmode=require", Options{
		MaxConns: 8, MinConns: 1, MaxConnLifetime: 30 * time.Minute,
		StatementTimeout: 5 * time.Second, TransactionTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("PoolConfig() error = %v", err)
	}
	if cfg.MaxConns != 8 || cfg.MinConns != 1 || cfg.MaxConnLifetime != 30*time.Minute {
		t.Fatalf("pool bounds = max:%d min:%d lifetime:%s", cfg.MaxConns, cfg.MinConns, cfg.MaxConnLifetime)
	}
	if cfg.ConnConfig.RuntimeParams["statement_timeout"] != "5000" || cfg.ConnConfig.RuntimeParams["transaction_timeout"] != "10000" {
		t.Fatalf("runtime params = %#v", cfg.ConnConfig.RuntimeParams)
	}
}
