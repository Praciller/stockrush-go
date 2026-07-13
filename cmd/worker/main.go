package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stockrush-go/internal/config"
	"stockrush-go/internal/database"
	"stockrush-go/internal/domain"
	"stockrush-go/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	dataStore := store.New(pool, cfg.ReservationTTL)
	ticker := time.NewTicker(cfg.WorkerPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			batchID, _ := domain.NewID()
			result, err := dataStore.ExpireBatch(ctx, cfg.WorkerBatchSize, now)
			if err != nil {
				slog.Error("expiration batch failed", "batch_id", batchID, "error", err)
				continue
			}
			if result.Processed > 0 {
				slog.Info("expiration batch complete", "batch_id", batchID, "processed", result.Processed, "expired", result.Expired, "restored", result.Restored)
			}
		}
	}
}
