package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"stockrush-go/internal/domain"
)

func (s *Store) ReservePublicDemo(ctx context.Context, key string, maxPerMinute int) (ReservationResult, error) {
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 200 {
		return ReservationResult{}, &domain.Error{Code: "VALIDATION_ERROR", Message: "Idempotency-Key is required", HTTPStatus: http.StatusBadRequest}
	}
	if maxPerMinute < 1 || maxPerMinute > 100 {
		return ReservationResult{}, fmt.Errorf("public demo limit must be between 1 and 100")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ReservationResult{}, err
	}
	defer rollback(tx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('stockrush-public-demo',0))`); err != nil {
		return ReservationResult{}, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM public_demo_actions WHERE created_at < now()-interval '24 hours'; DELETE FROM idempotency_keys WHERE expires_at < now()`); err != nil {
		return ReservationResult{}, err
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM public_demo_actions WHERE idempotency_key=$1)`, key).Scan(&exists); err != nil {
		return ReservationResult{}, err
	}
	if !exists {
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM public_demo_actions WHERE created_at > now()-interval '1 minute'`).Scan(&count); err != nil {
			return ReservationResult{}, err
		}
		if count >= maxPerMinute {
			return ReservationResult{}, &domain.Error{Code: "PUBLIC_DEMO_LIMIT_REACHED", Message: "The public demo action budget is exhausted; retry later", HTTPStatus: http.StatusTooManyRequests}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO public_demo_actions(idempotency_key) VALUES ($1)`, key); err != nil {
			return ReservationResult{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ReservationResult{}, err
	}

	var saleID string
	if err := s.pool.QueryRow(ctx, `SELECT s.id FROM flash_sales s JOIN products p ON p.id=s.product_id
		WHERE p.sku='FLASH-100' AND s.state='active' AND now() >= s.starts_at AND now() < s.ends_at
		ORDER BY s.created_at DESC LIMIT 1`).Scan(&saleID); err != nil {
		return ReservationResult{}, domainError(notFound(err), "DEMO_UNAVAILABLE", "The synthetic public demo sale is unavailable", http.StatusServiceUnavailable)
	}
	sum := sha256.Sum256([]byte(key))
	digest := hex.EncodeToString(sum[:])
	return s.Reserve(ctx, saleID, ReservationRequest{UserID: "public-" + digest[:24], Quantity: 1, IdempotencyKey: "public-" + digest})
}

type DemoStatus struct {
	Product         domain.Product `json:"product"`
	Sale            domain.Sale    `json:"sale"`
	Reservations    int            `json:"reservations"`
	Orders          int            `json:"orders"`
	DuplicateOrders int            `json:"duplicateOrders"`
	InvariantPass   bool           `json:"invariantPass"`
}

func (s *Store) ResetDemo(ctx context.Context, inventory int) (DemoStatus, error) {
	if inventory <= 0 || inventory > 100000 {
		return DemoStatus{}, &domain.Error{Code: "VALIDATION_ERROR", Message: "inventory must be between 1 and 100000", HTTPStatus: 400}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return DemoStatus{}, err
	}
	defer rollback(tx)
	if _, err := tx.Exec(ctx, `TRUNCATE public_demo_actions,domain_events,payments,idempotency_keys,orders,reservations,flash_sales,inventory,products`); err != nil {
		return DemoStatus{}, fmt.Errorf("reset demo: %w", err)
	}
	productID, err := domain.NewID()
	if err != nil {
		return DemoStatus{}, err
	}
	saleID, err := domain.NewID()
	if err != nil {
		return DemoStatus{}, err
	}
	now := s.now().UTC()
	if _, err := tx.Exec(ctx, `INSERT INTO products(id,sku,name,description,price_minor,currency,active)
		VALUES ($1,'FLASH-100','StockRush Limited Drop','Synthetic flash-sale inventory proof',9900,'THB',true)`, productID); err != nil {
		return DemoStatus{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO inventory(product_id,available,expected_total) VALUES ($1,$2,$2)`, productID, inventory); err != nil {
		return DemoStatus{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO flash_sales(id,product_id,starts_at,ends_at,allocated_stock,remaining_stock,max_quantity_per_user,state)
		VALUES ($1,$2,$3,$4,$5,$5,1,'active')`, saleID, productID, now.Add(-time.Minute), now.Add(365*24*time.Hour), inventory); err != nil {
		return DemoStatus{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DemoStatus{}, err
	}
	return s.DemoStatus(ctx)
}

func (s *Store) EnsurePublicDemo(ctx context.Context, inventory int) (DemoStatus, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM products WHERE sku='FLASH-100')`).Scan(&exists); err != nil {
		return DemoStatus{}, err
	}
	if exists {
		return s.DemoStatus(ctx)
	}
	return s.ResetDemo(ctx, inventory)
}

func (s *Store) DemoStatus(ctx context.Context) (DemoStatus, error) {
	var status DemoStatus
	err := s.pool.QueryRow(ctx, `SELECT p.id,p.sku,p.name,p.description,p.price_minor,p.currency,p.active,
		i.available,i.reserved,i.sold,p.created_at,p.updated_at
		FROM products p JOIN inventory i ON i.product_id=p.id ORDER BY p.created_at LIMIT 1`).
		Scan(&status.Product.ID, &status.Product.SKU, &status.Product.Name, &status.Product.Description, &status.Product.PriceMinor, &status.Product.Currency, &status.Product.Active, &status.Product.Available, &status.Product.Reserved, &status.Product.Sold, &status.Product.CreatedAt, &status.Product.UpdatedAt)
	if err != nil {
		return DemoStatus{}, notFound(err)
	}
	err = s.pool.QueryRow(ctx, `SELECT id,product_id,starts_at,ends_at,allocated_stock,remaining_stock,max_quantity_per_user,state,created_at,updated_at
		FROM flash_sales ORDER BY created_at LIMIT 1`).Scan(&status.Sale.ID, &status.Sale.ProductID, &status.Sale.StartsAt, &status.Sale.EndsAt, &status.Sale.AllocatedStock, &status.Sale.RemainingStock, &status.Sale.MaxQuantityPerUser, &status.Sale.State, &status.Sale.CreatedAt, &status.Sale.UpdatedAt)
	if err != nil {
		return DemoStatus{}, notFound(err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM reservations`).Scan(&status.Reservations); err != nil {
		return DemoStatus{}, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM orders`).Scan(&status.Orders); err != nil {
		return DemoStatus{}, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM (SELECT reservation_id FROM orders GROUP BY reservation_id HAVING count(*) > 1) d`).Scan(&status.DuplicateOrders); err != nil {
		return DemoStatus{}, err
	}
	status.InvariantPass = status.Product.Available >= 0 && status.Product.Reserved >= 0 && status.Product.Sold >= 0 && status.Sale.RemainingStock >= 0 && status.Reservations == status.Orders && status.DuplicateOrders == 0 && status.Product.Available+status.Product.Reserved+status.Product.Sold == status.Sale.AllocatedStock && status.Sale.RemainingStock+status.Product.Reserved+status.Product.Sold == status.Sale.AllocatedStock
	return status, nil
}
