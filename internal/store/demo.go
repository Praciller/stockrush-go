package store

import (
	"context"
	"fmt"
	"time"

	"stockrush-go/internal/domain"
)

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
	if _, err := tx.Exec(ctx, `TRUNCATE domain_events,payments,idempotency_keys,orders,reservations,flash_sales,inventory,products`); err != nil {
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
	if _, err := tx.Exec(ctx, `INSERT INTO inventory(product_id,available) VALUES ($1,$2)`, productID, inventory); err != nil {
		return DemoStatus{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO flash_sales(id,product_id,starts_at,ends_at,allocated_stock,remaining_stock,max_quantity_per_user,state)
		VALUES ($1,$2,$3,$4,$5,$5,1,'active')`, saleID, productID, now.Add(-time.Minute), now.Add(time.Hour), inventory); err != nil {
		return DemoStatus{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DemoStatus{}, err
	}
	return s.DemoStatus(ctx)
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
