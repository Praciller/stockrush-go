//go:build integration

package store

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"stockrush-go/internal/domain"
)

func TestMixedReservationRestoration(t *testing.T) {
	t.Run("shared product", func(t *testing.T) { testMixedReservationRestoration(t, true) })
	t.Run("multiple products", func(t *testing.T) { testMixedReservationRestoration(t, false) })
}

func testMixedReservationRestoration(t *testing.T, sharedProduct bool) {
	t.Helper()
	ctx, pool := testDatabase(t)
	s := New(pool, time.Minute)
	now := time.Now().UTC()
	seedTime := now.Add(-2 * time.Minute)
	s.now = func() time.Time { return seedTime }

	const stock = 48
	first, err := s.ResetDemo(ctx, stock)
	if err != nil {
		t.Fatal(err)
	}
	productID := first.Product.ID
	if !sharedProduct {
		product, err := s.CreateProduct(ctx, CreateProductInput{SKU: "OTHER-STOCK", Name: "Other stock", Currency: "THB", Inventory: stock, Active: true})
		if err != nil {
			t.Fatal(err)
		}
		productID = product.ID
	}
	second, err := s.CreateSale(ctx, CreateSaleInput{
		ProductID: productID, StartsAt: seedTime.Add(-time.Minute), EndsAt: now.Add(time.Hour),
		AllocatedStock: stock, MaxQuantityPerUser: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err = s.SetSaleState(ctx, second.ID, domain.SaleActive)
	if err != nil {
		t.Fatal(err)
	}
	sales := []string{first.Sale.ID, second.ID}
	seed := func(group string, count int) []string {
		t.Helper()
		ids := make([]string, 0, count)
		for i := range count {
			key := fmt.Sprintf("%s-%d", group, i)
			r, err := s.Reserve(ctx, sales[i%len(sales)], ReservationRequest{UserID: key, Quantity: 1, IdempotencyKey: key})
			if err != nil {
				t.Fatalf("seed %s: %v", key, err)
			}
			ids = append(ids, r.Reservation.ID)
		}
		return ids
	}
	expireIDs := seed("expires", 16)
	s.now = func() time.Time { return now }
	cancelIDs := seed("cancel", 16)
	type outcome struct {
		operation string
		restored  int
		err       error
	}
	const requests, workers = 64, 2
	start := make(chan struct{})
	results := make(chan outcome, requests+len(cancelIDs)+workers)
	for i := range requests {
		go func() {
			<-start
			key := fmt.Sprintf("purchase-%d", i)
			_, err := s.Reserve(ctx, sales[i%len(sales)], ReservationRequest{UserID: key, Quantity: 1, IdempotencyKey: key})
			results <- outcome{operation: "reserve", err: err}
		}()
	}
	for _, id := range cancelIDs {
		go func() {
			<-start
			_, err := s.TransitionReservation(ctx, id, domain.ReservationCancelled)
			results <- outcome{operation: "cancel", err: err}
		}()
	}
	for range workers {
		go func() {
			<-start
			r, err := s.ExpireBatch(ctx, 8, now)
			results <- outcome{operation: "expire", restored: r.Restored, err: err}
		}()
	}
	started := time.Now()
	close(start)
	counts := make(map[string]int)
	var rawErrors []string
	for range requests + len(cancelIDs) + workers {
		got := <-results
		var pgError *pgconn.PgError
		switch {
		case got.err == nil:
			counts[got.operation+".ok"]++
			counts["units_restored_by_expiry"] += got.restored
		case got.operation == "reserve" && domain.IsCode(got.err, "INVENTORY_SOLD_OUT"):
			counts["reserve.sold_out"]++
		case errors.As(got.err, &pgError):
			counts[got.operation+"."+pgError.Code]++
			rawErrors = append(rawErrors, got.operation+" "+got.err.Error())
		default:
			counts[got.operation+".other_error"]++
			rawErrors = append(rawErrors, got.operation+" "+got.err.Error())
		}
	}
	t.Logf("MIXED elapsed_ms=%d outcomes=%v", time.Since(started).Milliseconds(), counts)
	if counts["cancel.ok"] != len(cancelIDs) || counts["expire.ok"] != workers || counts["units_restored_by_expiry"] != len(expireIDs) {
		t.Errorf("restoration counts = %v", counts)
	}
	for _, id := range sales {
		var allocated, remaining, held int
		err := pool.QueryRow(ctx, `SELECT allocated_stock, remaining_stock,
			(SELECT coalesce(sum(quantity),0) FROM reservations WHERE sale_id=$1 AND state IN ('pending','confirmed','paid'))
			FROM flash_sales WHERE id=$1`, id).Scan(&allocated, &remaining, &held)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("SALE allocation=%d remaining=%d held=%d", allocated, remaining, held)
		if remaining+held != allocated {
			t.Errorf("sale delta=%d", allocated-remaining-held)
		}
	}
	var badInventory, reservations, orders, creationEvents, restorationEvents int
	err = pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM inventory i WHERE available < 0 OR sold <> 0 OR available+reserved+sold <> expected_total
			OR reserved <> (SELECT coalesce(sum(quantity),0) FROM reservations r
				WHERE r.product_id=i.product_id AND r.state IN ('pending','confirmed'))),
		(SELECT count(*) FROM reservations),
		(SELECT count(*) FROM orders o JOIN reservations r ON o.reservation_id=r.id WHERE o.state=r.state),
		(SELECT count(DISTINCT aggregate_id) FROM domain_events WHERE event_type='reservation.created'),
		(SELECT count(DISTINCT aggregate_id) FROM domain_events WHERE event_type IN ('reservation.cancelled','reservation.expired'))`).
		Scan(&badInventory, &reservations, &orders, &creationEvents, &restorationEvents)
	if err != nil {
		t.Fatal(err)
	}
	wantReservations := len(cancelIDs) + len(expireIDs) + counts["reserve.ok"]
	if badInventory != 0 || reservations != wantReservations || orders != wantReservations ||
		creationEvents != wantReservations || restorationEvents != len(cancelIDs)+len(expireIDs) {
		t.Errorf("bad inventory=%d reservations=%d orders=%d created=%d restored=%d",
			badInventory, reservations, orders, creationEvents, restorationEvents)
	}
	var duplicateEvents int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM (
		SELECT aggregate_id,event_type FROM domain_events GROUP BY aggregate_id,event_type HAVING count(*) > 1
	) duplicates`).Scan(&duplicateEvents)
	if err != nil || duplicateEvents != 0 {
		t.Errorf("duplicate events=%d err=%v", duplicateEvents, err)
	}
	if len(rawErrors) > 0 {
		t.Errorf("unexpected operations (%d): %s", len(rawErrors), strings.Join(rawErrors, "; "))
	}
}
