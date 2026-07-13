package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"stockrush-go/internal/config"
	"stockrush-go/internal/database"
	"stockrush-go/internal/expirationworker"
	"stockrush-go/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	if cfg.Production() {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := database.OpenWithOptions(ctx, cfg.DatabaseURL, database.Options{
		MaxConns: cfg.DatabaseMaxConns, MinConns: cfg.DatabaseMinConns, MaxConnLifetime: cfg.DatabaseMaxConnLifetime,
		StatementTimeout: cfg.DatabaseStatementTimeout, TransactionTimeout: cfg.DatabaseTransactionTimeout,
	})
	if err != nil {
		slog.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	dataStore := store.New(pool, cfg.ReservationTTL)
	expirationworker.Run(ctx, dataStore, cfg.WorkerBatchSize, cfg.WorkerPollInterval, nil)
}
