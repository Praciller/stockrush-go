package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"stockrush-go/internal/config"
	"stockrush-go/internal/database"
	"stockrush-go/internal/domain"
	"stockrush-go/internal/store"
)

type report struct {
	Timestamp           time.Time        `json:"timestamp"`
	GoVersion           string           `json:"goVersion"`
	InitialInventory    int              `json:"initialInventory"`
	Attempts            int              `json:"attempts"`
	Successful          int              `json:"successful"`
	SoldOut             int              `json:"soldOut"`
	Duplicate           int              `json:"duplicate"`
	RateLimited         int              `json:"rateLimited"`
	Failed              int              `json:"failed"`
	P50Millis           float64          `json:"p50Millis"`
	P95Millis           float64          `json:"p95Millis"`
	P99Millis           float64          `json:"p99Millis"`
	IdempotentResults   int              `json:"idempotentResults"`
	ExpiredReservations int              `json:"expiredReservations"`
	RestoredInventory   int              `json:"restoredInventory"`
	Final               store.DemoStatus `json:"final"`
	NegativeStockEvents int              `json:"negativeStockEvents"`
	DatabaseInvariantOK bool             `json:"databaseInvariantOk"`
	ZeroOverselling     bool             `json:"zeroOverselling"`
}

func main() {
	mode := flag.String("mode", "demo", "reset or demo")
	attempts := flag.Int("attempts", 1000, "bounded purchase attempts")
	flag.Parse()
	if *attempts < 1 || *attempts > 1000 {
		exit(errors.New("attempts must be between 1 and 1000"))
	}
	cfg, err := config.Load()
	if err != nil {
		exit(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		exit(err)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool, "db/migrations"); err != nil {
		exit(err)
	}
	dataStore := store.New(pool, cfg.ReservationTTL)
	if *mode == "reset" {
		_, err := dataStore.ResetDemo(ctx, 100)
		if err != nil {
			exit(err)
		}
		fmt.Println("demo reset: 100 units")
		return
	}
	if *mode != "demo" {
		exit(errors.New("mode must be reset or demo"))
	}
	r, err := run(ctx, dataStore, *attempts)
	if err != nil {
		exit(err)
	}
	if err := writeReport(r); err != nil {
		exit(err)
	}
	fmt.Printf("successful=%d sold_out=%d available=%d reserved=%d sold=%d zero_overselling=%t\n", r.Successful, r.SoldOut, r.Final.Product.Available, r.Final.Product.Reserved, r.Final.Product.Sold, r.ZeroOverselling)
	fmt.Println("frontend=http://localhost:5173 api=http://localhost:8080 report=reports/local_portfolio_report.md")
	if !r.ZeroOverselling {
		os.Exit(1)
	}
}

func run(ctx context.Context, dataStore *store.Store, attempts int) (report, error) {
	r := report{Timestamp: time.Now().UTC(), GoVersion: runtime.Version(), InitialInventory: 100, Attempts: attempts}
	proof, err := dataStore.ResetDemo(ctx, 1)
	if err != nil {
		return report{}, err
	}
	var wg sync.WaitGroup
	ids := make(chan string, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := dataStore.Reserve(ctx, proof.Sale.ID, store.ReservationRequest{UserID: "idempotency-proof", Quantity: 1, IdempotencyKey: "idempotency-proof"})
			if err == nil {
				ids <- result.Reservation.ID
			}
		}()
	}
	wg.Wait()
	close(ids)
	var first string
	for id := range ids {
		if first == "" {
			first = id
		}
		if id == first {
			r.IdempotentResults++
		}
	}
	reservation, err := dataStore.GetReservation(ctx, first)
	if err != nil {
		return report{}, err
	}
	expired, err := dataStore.ExpireBatch(ctx, 50, reservation.ExpiresAt.Add(time.Second))
	if err != nil {
		return report{}, err
	}
	r.ExpiredReservations = expired.Expired
	r.RestoredInventory = expired.Restored

	initial, err := dataStore.ResetDemo(ctx, r.InitialInventory)
	if err != nil {
		return report{}, err
	}
	latencies := make([]time.Duration, 0, attempts)
	var mu sync.Mutex
	start := make(chan struct{})
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			started := time.Now()
			result, reserveErr := dataStore.Reserve(ctx, initial.Sale.ID, store.ReservationRequest{UserID: fmt.Sprintf("buyer-%04d", i), Quantity: 1, IdempotencyKey: fmt.Sprintf("demo-%d-%04d", r.Timestamp.UnixNano(), i)})
			elapsed := time.Since(started)
			mu.Lock()
			defer mu.Unlock()
			latencies = append(latencies, elapsed)
			switch {
			case reserveErr == nil && result.Duplicate:
				r.Duplicate++
			case reserveErr == nil:
				r.Successful++
			case domain.IsCode(reserveErr, "INVENTORY_SOLD_OUT"):
				r.SoldOut++
			case domain.IsCode(reserveErr, "RATE_LIMITED"):
				r.RateLimited++
			default:
				r.Failed++
			}
		}(i)
	}
	close(start)
	wg.Wait()
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	r.P50Millis, r.P95Millis, r.P99Millis = percentile(latencies, .50), percentile(latencies, .95), percentile(latencies, .99)
	r.Final, err = dataStore.DemoStatus(ctx)
	if err != nil {
		return report{}, err
	}
	r.DatabaseInvariantOK = r.Final.InvariantPass
	r.ZeroOverselling = r.DatabaseInvariantOK && r.Successful <= r.InitialInventory && r.Final.Product.Available >= 0 && r.Final.DuplicateOrders == 0 && r.IdempotentResults == 100 && r.RestoredInventory == 1
	return r, nil
}

func percentile(values []time.Duration, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	return float64(values[int(float64(len(values)-1)*p)].Microseconds()) / 1000
}

func writeReport(r report) error {
	if err := os.MkdirAll("reports", 0o755); err != nil {
		return err
	}
	verdict := "FAIL"
	if r.ZeroOverselling {
		verdict = "PASS"
	}
	markdown := fmt.Sprintf(`# StockRush Go Portfolio Evidence

## Environment

- Timestamp: %s
- Go: %s
- Storage: PostgreSQL

## Demo Parameters

- Initial inventory: %d
- Concurrent purchase attempts: %d
- Quantity per request: 1

## Concurrency Test Results

| Result | Count |
|---|---:|
| Successful reservations | %d |
| Sold out | %d |
| Duplicate | %d |
| Rate limited | %d |
| Failed | %d |

## Latency Summary

- p50: %.2f ms
- p95: %.2f ms
- p99: %.2f ms

## Reservation Expiration Results

- Expired reservations: %d
- Restored inventory: %d

## Idempotency Results

- Concurrent retries returning one reservation: %d/100
- Duplicate orders: %d

## Database Invariant Checks

- Available inventory is non-negative: %t
- Reserved inventory is non-negative: %t
- Sold inventory is non-negative: %t
- Reservation/order reconciliation: %t

## Final Inventory Reconciliation

- Available: %d
- Reserved: %d
- Sold: %d
- Total: %d

## Zero-Oversell Verdict

**Zero overselling: %s**

## Known Limitations

- The MVP rate limiter is process-local.
- Payment is synthetic.
- The local Docker Compose demo is authoritative.
`, r.Timestamp.Format(time.RFC3339), r.GoVersion, r.InitialInventory, r.Attempts, r.Successful, r.SoldOut, r.Duplicate, r.RateLimited, r.Failed, r.P50Millis, r.P95Millis, r.P99Millis, r.ExpiredReservations, r.RestoredInventory, r.IdempotentResults, r.Final.DuplicateOrders, r.Final.Product.Available >= 0, r.Final.Product.Reserved >= 0, r.Final.Product.Sold >= 0, r.Final.Reservations == r.Final.Orders, r.Final.Product.Available, r.Final.Product.Reserved, r.Final.Product.Sold, r.Final.Product.Available+r.Final.Product.Reserved+r.Final.Product.Sold, verdict)
	if err := os.WriteFile(filepath.Join("reports", "local_portfolio_report.md"), []byte(strings.ReplaceAll(markdown, "\r\n", "\n")), 0o644); err != nil {
		return err
	}
	jsonBody, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join("reports", "local_portfolio_report.json"), append(jsonBody, '\n'), 0o644)
}

func exit(err error) {
	slog.Error("load generator failed", "error", err)
	os.Exit(1)
}
