//go:build integration

package store

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"stockrush-go/internal/database"
	"stockrush-go/internal/domain"
)

func TestConcurrentReservationsNeverOversell(t *testing.T) {
	ctx, pool := testDatabase(t)
	s := New(pool, 2*time.Minute)
	status, err := s.ResetDemo(ctx, 100)
	if err != nil {
		t.Fatalf("ResetDemo: %v", err)
	}

	var success atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, err := s.Reserve(ctx, status.Sale.ID, ReservationRequest{
				UserID:         fmt.Sprintf("buyer-%04d", i),
				Quantity:       1,
				IdempotencyKey: fmt.Sprintf("checkout-%04d", i),
			})
			if err == nil {
				success.Add(1)
			}
		}(i)
	}
	close(start)
	wg.Wait()

	final, err := s.DemoStatus(ctx)
	if err != nil {
		t.Fatalf("DemoStatus: %v", err)
	}
	if success.Load() != 100 || final.Product.Available != 0 || final.Product.Reserved+final.Product.Sold != 100 || final.DuplicateOrders != 0 {
		t.Fatalf("success=%d available=%d reserved=%d sold=%d duplicates=%d", success.Load(), final.Product.Available, final.Product.Reserved, final.Product.Sold, final.DuplicateOrders)
	}
}

func TestConcurrentReservationsRespectSaleAllocation(t *testing.T) {
	ctx, pool := testDatabase(t)
	s := New(pool, 2*time.Minute)
	status, err := s.ResetDemo(ctx, 100)
	if err != nil {
		t.Fatalf("ResetDemo: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE flash_sales SET allocated_stock=10,remaining_stock=10 WHERE id=$1`, status.Sale.ID); err != nil {
		t.Fatalf("set sale allocation: %v", err)
	}

	var success atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, err := s.Reserve(ctx, status.Sale.ID, ReservationRequest{
				UserID:         fmt.Sprintf("allocated-buyer-%03d", i),
				Quantity:       1,
				IdempotencyKey: fmt.Sprintf("allocated-checkout-%03d", i),
			})
			if err == nil {
				success.Add(1)
			}
		}(i)
	}
	close(start)
	wg.Wait()

	product, err := s.GetProduct(ctx, status.Product.ID)
	if err != nil {
		t.Fatalf("GetProduct: %v", err)
	}
	sale, err := s.GetSale(ctx, status.Sale.ID)
	if err != nil {
		t.Fatalf("GetSale: %v", err)
	}
	if success.Load() != 10 || product.Available != 90 || product.Reserved != 10 || sale.RemainingStock != 0 {
		t.Fatalf("success=%d available=%d reserved=%d remaining=%d", success.Load(), product.Available, product.Reserved, sale.RemainingStock)
	}
}

func TestConcurrentDuplicateReservationsReturnOneResource(t *testing.T) {
	ctx, pool := testDatabase(t)
	s := New(pool, 2*time.Minute)
	status, err := s.ResetDemo(ctx, 100)
	if err != nil {
		t.Fatalf("ResetDemo: %v", err)
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	ids := make(chan string, 1000)
	errs := make(chan error, 1000)
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := s.Reserve(ctx, status.Sale.ID, ReservationRequest{UserID: "same-buyer", Quantity: 1, IdempotencyKey: "same-key"})
			if err != nil {
				errs <- err
				return
			}
			ids <- result.Reservation.ID
		}()
	}
	close(start)
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Fatalf("Reserve: %v", err)
	}
	var original string
	for id := range ids {
		if original == "" {
			original = id
		}
		if id != original {
			t.Fatalf("duplicate returned reservation %s, want %s", id, original)
		}
	}
	final, err := s.DemoStatus(ctx)
	if err != nil {
		t.Fatalf("DemoStatus: %v", err)
	}
	if final.Reservations != 1 || final.Orders != 1 || final.Product.Available != 99 {
		t.Fatalf("reservations=%d orders=%d available=%d", final.Reservations, final.Orders, final.Product.Available)
	}
}

func TestCompetingExpirationWorkersRestoreInventoryOnce(t *testing.T) {
	ctx, pool := testDatabase(t)
	fixed := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	s := New(pool, time.Minute)
	s.now = func() time.Time { return fixed }
	status, err := s.ResetDemo(ctx, 100)
	if err != nil {
		t.Fatalf("ResetDemo: %v", err)
	}
	if _, err := s.Reserve(ctx, status.Sale.ID, ReservationRequest{UserID: "expiring-buyer", Quantity: 1, IdempotencyKey: "expiring-key"}); err != nil {
		t.Fatalf("Reserve: %v", err)
	}

	var restored atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := s.ExpireBatch(ctx, 50, fixed.Add(2*time.Minute))
			if err != nil {
				t.Errorf("ExpireBatch: %v", err)
				return
			}
			restored.Add(int64(result.Restored))
		}()
	}
	close(start)
	wg.Wait()
	final, err := s.DemoStatus(ctx)
	if err != nil {
		t.Fatalf("DemoStatus: %v", err)
	}
	if restored.Load() != 1 || final.Product.Available != 100 || final.Product.Reserved != 0 {
		t.Fatalf("restored=%d available=%d reserved=%d", restored.Load(), final.Product.Available, final.Product.Reserved)
	}
}

func TestDuplicatePaymentCallbacksApplyInventoryOnce(t *testing.T) {
	ctx, pool := testDatabase(t)
	s := New(pool, 2*time.Minute)
	status, err := s.ResetDemo(ctx, 100)
	if err != nil {
		t.Fatalf("ResetDemo: %v", err)
	}
	reserved, err := s.Reserve(ctx, status.Sale.ID, ReservationRequest{UserID: "paying-buyer", Quantity: 1, IdempotencyKey: "payment-reservation"})
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if _, err := s.TransitionReservation(ctx, reserved.Reservation.ID, domain.ReservationConfirmed); err != nil {
		t.Fatalf("confirm reservation: %v", err)
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := s.SimulatePayment(ctx, PaymentRequest{OrderID: reserved.Reservation.OrderID, Outcome: "successful", IdempotencyKey: "same-payment-callback"})
			if err != nil {
				errs <- err
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("SimulatePayment: %v", err)
	}
	final, err := s.DemoStatus(ctx)
	if err != nil {
		t.Fatalf("DemoStatus: %v", err)
	}
	if final.Product.Available != 99 || final.Product.Reserved != 0 || final.Product.Sold != 1 {
		t.Fatalf("available=%d reserved=%d sold=%d", final.Product.Available, final.Product.Reserved, final.Product.Sold)
	}
}

func TestCancellationAndExpirationRaceRestoresOnce(t *testing.T) {
	ctx, pool := testDatabase(t)
	fixed := time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC)
	s := New(pool, time.Minute)
	s.now = func() time.Time { return fixed }
	status, err := s.ResetDemo(ctx, 10)
	if err != nil {
		t.Fatalf("ResetDemo: %v", err)
	}
	result, err := s.Reserve(ctx, status.Sale.ID, ReservationRequest{UserID: "racing-buyer", Quantity: 1, IdempotencyKey: "racing-key"})
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, _ = s.TransitionReservation(ctx, result.Reservation.ID, domain.ReservationCancelled)
	}()
	go func() {
		defer wg.Done()
		<-start
		_, _ = s.ExpireBatch(ctx, 50, fixed.Add(2*time.Minute))
	}()
	close(start)
	wg.Wait()

	final, err := s.DemoStatus(ctx)
	if err != nil {
		t.Fatalf("DemoStatus: %v", err)
	}
	reservation, err := s.GetReservation(ctx, result.Reservation.ID)
	if err != nil {
		t.Fatalf("GetReservation: %v", err)
	}
	if final.Product.Available != 10 || final.Product.Reserved != 0 || (reservation.State != domain.ReservationCancelled && reservation.State != domain.ReservationExpired) {
		t.Fatalf("available=%d reserved=%d state=%s", final.Product.Available, final.Product.Reserved, reservation.State)
	}
}

func TestIdempotencyKeyRejectsDifferentPayload(t *testing.T) {
	ctx, pool := testDatabase(t)
	s := New(pool, 2*time.Minute)
	status, err := s.ResetDemo(ctx, 10)
	if err != nil {
		t.Fatalf("ResetDemo: %v", err)
	}
	if _, err := s.Reserve(ctx, status.Sale.ID, ReservationRequest{UserID: "buyer-a", Quantity: 1, IdempotencyKey: "reused-key"}); err != nil {
		t.Fatalf("first Reserve: %v", err)
	}
	if _, err := s.Reserve(ctx, status.Sale.ID, ReservationRequest{UserID: "buyer-b", Quantity: 1, IdempotencyKey: "reused-key"}); !domain.IsCode(err, "IDEMPOTENCY_CONFLICT") {
		t.Fatalf("second Reserve error = %v, want IDEMPOTENCY_CONFLICT", err)
	}
}

func TestSaleEndingRejectsNewReservations(t *testing.T) {
	ctx, pool := testDatabase(t)
	s := New(pool, 2*time.Minute)
	status, err := s.ResetDemo(ctx, 100)
	if err != nil {
		t.Fatalf("ResetDemo: %v", err)
	}
	if _, err := s.SetSaleState(ctx, status.Sale.ID, domain.SaleEnded); err != nil {
		t.Fatalf("SetSaleState: %v", err)
	}
	if _, err := s.Reserve(ctx, status.Sale.ID, ReservationRequest{UserID: "late-buyer", Quantity: 1, IdempotencyKey: "late-key"}); !domain.IsCode(err, "SALE_NOT_ACTIVE") {
		t.Fatalf("Reserve error = %v, want SALE_NOT_ACTIVE", err)
	}
}

func testDatabase(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://stockrush:stockrush@localhost:5432/stockrush?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Skipf("PostgreSQL unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool, "../../db/migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return ctx, pool
}
