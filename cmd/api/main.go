package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"stockrush-go/internal/config"
	"stockrush-go/internal/database"
	"stockrush-go/internal/expirationworker"
	"stockrush-go/internal/httpserver"
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
	if cfg.RunMigrations {
		if err := database.Migrate(ctx, pool, "db/migrations"); err != nil {
			slog.Error("migration failed", "error", err)
			os.Exit(1)
		}
	}
	if cfg.PublicDemoSeedEnabled {
		if _, err := dataStore.EnsurePublicDemo(ctx, 100); err != nil {
			slog.Error("public demo seed failed", "error", err)
			os.Exit(1)
		}
	}

	registry := prometheus.NewRegistry()
	handler, metrics := httpserver.NewWithMetrics(cfg, pool, dataStore, registry)
	if cfg.RunWorker {
		go expirationworker.Run(ctx, dataStore, cfg.WorkerBatchSize, cfg.WorkerPollInterval, metrics)
	}
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	go func() {
		slog.Info("API listening", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("API stopped unexpectedly", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}
