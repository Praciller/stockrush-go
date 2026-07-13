package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"stockrush-go/internal/config"
	"stockrush-go/internal/database"
	"stockrush-go/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fail(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := database.OpenWithOptions(ctx, cfg.DatabaseURL, database.Options{
		MaxConns: cfg.DatabaseMaxConns, MinConns: cfg.DatabaseMinConns, MaxConnLifetime: cfg.DatabaseMaxConnLifetime,
		StatementTimeout: cfg.DatabaseStatementTimeout, TransactionTimeout: cfg.DatabaseTransactionTimeout,
	})
	if err != nil {
		fail(err)
	}
	defer pool.Close()
	violations, err := store.New(pool, cfg.ReservationTTL).CheckInvariants(ctx)
	if err != nil {
		fail(err)
	}
	status := "pass"
	if len(violations) > 0 {
		status = "fail"
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": status, "violations": violations})
	if len(violations) > 0 {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
