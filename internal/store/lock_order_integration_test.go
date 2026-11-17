//go:build integration

package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReserveWaitsForInventoryBeforeLockingSale(t *testing.T) {
	ctx, pool := testDatabase(t)
	s := New(pool, time.Minute)
	setup, err := s.ResetDemo(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer rollback(tx)
	var pid int32
	err = tx.QueryRow(ctx, `SELECT pg_backend_pid() FROM inventory
		WHERE product_id=$1 FOR NO KEY UPDATE`, setup.Product.ID).Scan(&pid)
	if err != nil {
		t.Fatal(err)
	}

	result := make(chan error, 1)
	go func() {
		_, err := s.Reserve(ctx, setup.Sale.ID, ReservationRequest{
			UserID: "contending-buyer", Quantity: 1, IdempotencyKey: "contending-request",
		})
		result <- err
	}()
	waitForBlockedTransaction(t, ctx, pool, pid, result)
	// Use a real row lock without introducing a second mutation into this fixture.
	var saleID string
	err = tx.QueryRow(ctx, `SELECT id FROM flash_sales WHERE id=$1 FOR NO KEY UPDATE`, setup.Sale.ID).Scan(&saleID)
	if err == nil {
		err = tx.Commit(ctx)
	} else {
		rollback(tx)
	}
	checkTransactionResult(t, "lock sale", err)
	checkTransactionResult(t, "reserve", <-result)
}

func TestPaymentWaitsForReservationBeforeLockingOrder(t *testing.T) {
	ctx, pool := testDatabase(t)
	s := New(pool, time.Minute)
	setup, err := s.ResetDemo(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	reserved, err := s.Reserve(ctx, setup.Sale.ID, ReservationRequest{
		UserID: "payment-buyer", Quantity: 1, IdempotencyKey: "payment-reservation",
	})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer rollback(tx)
	var pid int32
	err = tx.QueryRow(ctx, `SELECT pg_backend_pid() FROM reservations
		WHERE id=$1 FOR NO KEY UPDATE`, reserved.Reservation.ID).Scan(&pid)
	if err != nil {
		t.Fatal(err)
	}

	result := make(chan error, 1)
	go func() {
		_, err := s.SimulatePayment(ctx, PaymentRequest{
			OrderID: reserved.Reservation.OrderID, Outcome: "delayed", IdempotencyKey: "contending-payment",
		})
		result <- err
	}()
	waitForBlockedTransaction(t, ctx, pool, pid, result)
	var orderID string
	err = tx.QueryRow(ctx, `SELECT id FROM orders WHERE id=$1 FOR NO KEY UPDATE`, reserved.Reservation.OrderID).Scan(&orderID)
	if err == nil {
		err = tx.Commit(ctx)
	} else {
		rollback(tx)
	}
	checkTransactionResult(t, "lock order", err)
	checkTransactionResult(t, "payment", <-result)
}

func waitForBlockedTransaction(t *testing.T, ctx context.Context, pool *pgxpool.Pool, pid int32, result <-chan error) {
	t.Helper()
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	poll := time.NewTicker(10 * time.Millisecond)
	defer poll.Stop()
	for {
		var blockedPID int32
		var query, wait string
		err := pool.QueryRow(waitCtx, `SELECT coalesce(max(pid),0), coalesce(max(query),''), coalesce(max(wait_event),'')
			FROM pg_stat_activity WHERE $1 = ANY(pg_blocking_pids(pid))`, pid).Scan(&blockedPID, &query, &wait)
		if err != nil {
			t.Fatalf("observe wait on backend %d: %v", pid, err)
		}
		if blockedPID != 0 {
			t.Logf("backend %d waits on backend %d (%s): %s", blockedPID, pid, wait, query)
			return
		}
		select {
		case err := <-result:
			t.Fatalf("operation completed before lock release: %v", err)
		case <-waitCtx.Done():
			t.Fatalf("operation did not wait for backend %d: %v", pid, waitCtx.Err())
		case <-poll.C:
		}
	}

}

func checkTransactionResult(t *testing.T, operation string, err error) {
	t.Helper()
	if err == nil {
		return
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		t.Errorf("%s: SQLSTATE %s: %s", operation, pgErr.Code, pgErr.Detail)
	} else {
		t.Errorf("%s: %v", operation, err)
	}
}
