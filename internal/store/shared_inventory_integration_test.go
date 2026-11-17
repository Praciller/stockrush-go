//go:build integration

package store

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"stockrush-go/internal/domain"
)

func TestSharedInventorySoldOutPreservesAllocationAndReplays(t *testing.T) {
	ctx, pool := testDatabase(t)
	s := New(pool, 2*time.Minute)
	status, err := s.ResetDemo(ctx, 1)
	if err != nil {
		t.Fatalf("ResetDemo: %v", err)
	}
	second := activateSharedProductSale(t, ctx, s, status.Product.ID, 1)
	first, err := s.Reserve(ctx, status.Sale.ID, ReservationRequest{
		UserID: "first-buyer", Quantity: 1, IdempotencyKey: "first-reservation",
	})
	if err != nil {
		t.Fatalf("exhaust product stock through first sale: %v", err)
	}

	request := ReservationRequest{UserID: "second-buyer", Quantity: 1, IdempotencyKey: "sold-out-reservation"}
	before := readReservationState(t, ctx, pool)
	result, err := s.Reserve(ctx, second.ID, request)
	want := domain.Error{Code: "INVENTORY_SOLD_OUT", Message: "The requested product is sold out", HTTPStatus: http.StatusConflict}
	var rejection *domain.Error
	if !errors.As(err, &rejection) || *rejection != want || result != (ReservationResult{}) {
		t.Fatalf("Reserve() = %+v, %v, want empty result and %+v", result, err, want)
	}
	sale, err := s.GetSale(ctx, second.ID)
	if err != nil {
		t.Fatalf("GetSale after rejection: %v", err)
	}
	if sale.RemainingStock != second.RemainingStock {
		t.Errorf("rejected sale remaining_stock = %d, want %d", sale.RemainingStock, second.RemainingStock)
	}
	if after := readReservationState(t, ctx, pool); after != before {
		t.Errorf("rejection changed business rows\nbefore: %+v\nafter:  %+v", before, after)
	}

	var persisted domain.Error
	var noResource bool
	err = pool.QueryRow(ctx, `SELECT response_status, response_body->>'errorCode', response_body->>'message', resource_id IS NULL
		FROM idempotency_keys WHERE key=$1`, request.IdempotencyKey).
		Scan(&persisted.HTTPStatus, &persisted.Code, &persisted.Message, &noResource)
	if err != nil || persisted != want || !noResource {
		t.Fatalf("persisted rejection = %+v, no resource = %t, err = %v", persisted, noResource, err)
	}

	// Restore global stock so executing the request again would no longer reject on product availability.
	if _, err := s.TransitionReservation(ctx, first.Reservation.ID, domain.ReservationCancelled); err != nil {
		t.Fatalf("restore product stock: %v", err)
	}
	beforeReplay := readReservationState(t, ctx, pool)
	result, err = s.Reserve(ctx, second.ID, request)
	rejection = nil
	if !errors.As(err, &rejection) || *rejection != want || result != (ReservationResult{}) {
		t.Fatalf("replay after restock = %+v, %v, want empty result and %+v", result, err, want)
	}
	if after := readReservationState(t, ctx, pool); after != beforeReplay {
		t.Errorf("replay changed business rows\nbefore: %+v\nafter:  %+v", beforeReplay, after)
	}
}

func TestConcurrentReservationsAcrossSalesPreserveSharedInventory(t *testing.T) {
	ctx, pool := testDatabase(t)
	s := New(pool, 2*time.Minute)
	const stock, attemptsPerSale = 16, 32
	status, err := s.ResetDemo(ctx, stock)
	if err != nil {
		t.Fatalf("ResetDemo: %v", err)
	}
	second := activateSharedProductSale(t, ctx, s, status.Product.ID, stock)
	sales := []domain.Sale{status.Sale, second}

	type outcome struct {
		saleID string
		result ReservationResult
		err    error
	}
	start := make(chan struct{})
	outcomes := make(chan outcome, len(sales)*attemptsPerSale)
	for _, sale := range sales {
		for i := 0; i < attemptsPerSale; i++ {
			go func(saleID string, buyer int) {
				<-start
				key := fmt.Sprintf("%s-buyer-%02d", saleID, buyer)
				result, err := s.Reserve(ctx, saleID, ReservationRequest{
					UserID: key, Quantity: 1, IdempotencyKey: key,
				})
				outcomes <- outcome{saleID, result, err}
			}(sale.ID, i)
		}
	}
	close(start)
	successes := make(map[string]int)
	soldOut := 0
	for range len(sales) * attemptsPerSale {
		got := <-outcomes
		var rejection *domain.Error
		switch {
		case got.err == nil && !got.result.Duplicate:
			successes[got.saleID]++
		case errors.As(got.err, &rejection) && rejection.Code == "INVENTORY_SOLD_OUT" && rejection.HTTPStatus == http.StatusConflict:
			soldOut++
		default:
			t.Errorf("unexpected reservation result for sale %s: %+v, %v", got.saleID, got.result, got.err)
		}
	}

	for _, sale := range sales {
		got, err := s.GetSale(ctx, sale.ID)
		if err != nil {
			t.Fatalf("GetSale: %v", err)
		}
		if want := sale.AllocatedStock - successes[sale.ID]; got.RemainingStock != want {
			t.Errorf("sale %s remaining_stock = %d, want %d", sale.ID, got.RemainingStock, want)
		}
	}
	if successful := successes[status.Sale.ID] + successes[second.ID]; successful != stock || soldOut != len(sales)*attemptsPerSale-stock {
		t.Errorf("successes = %d, sold out = %d, want %d and %d", successful, soldOut, stock, len(sales)*attemptsPerSale-stock)
	}

	final, err := s.GetProduct(ctx, status.Product.ID)
	if err != nil {
		t.Fatalf("GetProduct: %v", err)
	}
	if final.Available != 0 || final.Reserved != stock || final.Sold != 0 {
		t.Errorf("final inventory = %d/%d/%d, want 0/%d/0", final.Available, final.Reserved, final.Sold, stock)
	}
	var reservations, orders, events int
	err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM reservations), (SELECT count(*) FROM orders),
		(SELECT count(*) FROM domain_events)`).Scan(&reservations, &orders, &events)
	if err != nil || reservations != stock || orders != stock || events != stock {
		t.Errorf("reservation/order/event counts = %d/%d/%d, want %d each, err = %v", reservations, orders, events, stock, err)
	}
}

func activateSharedProductSale(t *testing.T, ctx context.Context, s *Store, productID string, stock int) domain.Sale {
	t.Helper()
	now := s.now()
	sale, err := s.CreateSale(ctx, CreateSaleInput{
		ProductID: productID, StartsAt: now.Add(-time.Minute), EndsAt: now.Add(time.Hour),
		AllocatedStock: stock, MaxQuantityPerUser: 1,
	})
	if err != nil {
		t.Fatalf("CreateSale: %v", err)
	}
	sale, err = s.SetSaleState(ctx, sale.ID, domain.SaleActive)
	if err != nil {
		t.Fatalf("activate sale: %v", err)
	}
	return sale
}

type reservationState struct {
	inventory, sales, reservations, orders, events string
}

func readReservationState(t *testing.T, ctx context.Context, pool *pgxpool.Pool) reservationState {
	t.Helper()
	var state reservationState
	err := pool.QueryRow(ctx, `SELECT
		(SELECT coalesce(jsonb_agg(i ORDER BY product_id), '[]')::text FROM inventory i),
		(SELECT coalesce(jsonb_agg(s ORDER BY id), '[]')::text FROM flash_sales s),
		(SELECT coalesce(jsonb_agg(r ORDER BY id), '[]')::text FROM reservations r),
		(SELECT coalesce(jsonb_agg(o ORDER BY id), '[]')::text FROM orders o),
		(SELECT coalesce(jsonb_agg(e ORDER BY id), '[]')::text FROM domain_events e)`).
		Scan(&state.inventory, &state.sales, &state.reservations, &state.orders, &state.events)
	if err != nil {
		t.Fatalf("snapshot reservation state: %v", err)
	}
	return state
}
