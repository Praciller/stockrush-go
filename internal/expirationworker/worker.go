package expirationworker

import (
	"context"
	"log/slog"
	"time"

	"stockrush-go/internal/domain"
	metricspkg "stockrush-go/internal/metrics"
	"stockrush-go/internal/store"
)

func Run(ctx context.Context, dataStore *store.Store, batchSize int, pollInterval time.Duration, metrics *metricspkg.Metrics) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			batchID, _ := domain.NewID()
			result, err := dataStore.ExpireBatch(ctx, batchSize, now)
			if err != nil {
				if metrics != nil {
					metrics.WorkerErrors.Inc()
				}
				slog.Error("expiration batch failed", "batch_id", batchID, "error", err)
				continue
			}
			if metrics != nil {
				metrics.Expired.Add(float64(result.Expired))
				metrics.InventoryRestored.Add(float64(result.Restored))
			}
			if result.Processed > 0 {
				slog.Info("expiration batch complete", "batch_id", batchID, "processed", result.Processed, "expired", result.Expired, "restored", result.Restored)
			}
		}
	}
}
