package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"stockrush-go/internal/domain"
)

type CreateProductInput struct {
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceMinor  int64  `json:"priceMinor"`
	Currency    string `json:"currency"`
	Inventory   int    `json:"inventory"`
	Active      bool   `json:"active"`
}

type UpdateProductInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	PriceMinor  *int64  `json:"priceMinor"`
	Active      *bool   `json:"active"`
	Inventory   *int    `json:"inventory"`
}

func (s *Store) CreateProduct(ctx context.Context, in CreateProductInput) (domain.Product, error) {
	in.SKU = strings.TrimSpace(in.SKU)
	in.Name = strings.TrimSpace(in.Name)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	if in.SKU == "" || in.Name == "" || len(in.Currency) != 3 || in.PriceMinor < 0 || in.Inventory < 0 {
		return domain.Product{}, &domain.Error{Code: "VALIDATION_ERROR", Message: "SKU, name, currency, price, and inventory are invalid", HTTPStatus: 400}
	}
	id, err := domain.NewID()
	if err != nil {
		return domain.Product{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Product{}, err
	}
	defer rollback(tx)
	var product domain.Product
	err = tx.QueryRow(ctx, `INSERT INTO products (id,sku,name,description,price_minor,currency,active)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id,sku,name,description,price_minor,currency,active,created_at,updated_at`,
		id, in.SKU, in.Name, in.Description, in.PriceMinor, in.Currency, in.Active).
		Scan(&product.ID, &product.SKU, &product.Name, &product.Description, &product.PriceMinor, &product.Currency, &product.Active, &product.CreatedAt, &product.UpdatedAt)
	if err != nil {
		return domain.Product{}, fmt.Errorf("insert product: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO inventory(product_id,available) VALUES ($1,$2)`, id, in.Inventory); err != nil {
		return domain.Product{}, fmt.Errorf("insert inventory: %w", err)
	}
	product.Available = in.Inventory
	if err := tx.Commit(ctx); err != nil {
		return domain.Product{}, err
	}
	return product, nil
}

func (s *Store) ListProducts(ctx context.Context) ([]domain.Product, error) {
	rows, err := s.pool.Query(ctx, `SELECT p.id,p.sku,p.name,p.description,p.price_minor,p.currency,p.active,
		i.available,i.reserved,i.sold,p.created_at,p.updated_at
		FROM products p JOIN inventory i ON i.product_id=p.id ORDER BY p.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	products := make([]domain.Product, 0)
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.PriceMinor, &p.Currency, &p.Active, &p.Available, &p.Reserved, &p.Sold, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (s *Store) GetProduct(ctx context.Context, id string) (domain.Product, error) {
	var p domain.Product
	err := s.pool.QueryRow(ctx, `SELECT p.id,p.sku,p.name,p.description,p.price_minor,p.currency,p.active,
		i.available,i.reserved,i.sold,p.created_at,p.updated_at
		FROM products p JOIN inventory i ON i.product_id=p.id WHERE p.id=$1`, id).
		Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.PriceMinor, &p.Currency, &p.Active, &p.Available, &p.Reserved, &p.Sold, &p.CreatedAt, &p.UpdatedAt)
	return p, notFound(err)
}

func (s *Store) UpdateProduct(ctx context.Context, id string, in UpdateProductInput) (domain.Product, error) {
	if in.PriceMinor != nil && *in.PriceMinor < 0 || in.Inventory != nil && *in.Inventory < 0 {
		return domain.Product{}, &domain.Error{Code: "VALIDATION_ERROR", Message: "price and inventory cannot be negative", HTTPStatus: 400}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Product{}, err
	}
	defer rollback(tx)
	result, err := tx.Exec(ctx, `UPDATE products SET
		name=COALESCE($2,name), description=COALESCE($3,description), price_minor=COALESCE($4,price_minor),
		active=COALESCE($5,active), updated_at=now() WHERE id=$1`, id, in.Name, in.Description, in.PriceMinor, in.Active)
	if err != nil || result.RowsAffected() == 0 {
		if err == nil {
			err = ErrNotFound
		}
		return domain.Product{}, err
	}
	if in.Inventory != nil {
		result, err = tx.Exec(ctx, `UPDATE inventory SET available=$2,updated_at=now() WHERE product_id=$1`, id, *in.Inventory)
		if err != nil || result.RowsAffected() == 0 {
			return domain.Product{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Product{}, err
	}
	return s.GetProduct(ctx, id)
}

type CreateSaleInput struct {
	ProductID          string    `json:"productId"`
	StartsAt           time.Time `json:"startsAt"`
	EndsAt             time.Time `json:"endsAt"`
	AllocatedStock     int       `json:"allocatedStock"`
	MaxQuantityPerUser int       `json:"maxQuantityPerUser"`
}

func (s *Store) CreateSale(ctx context.Context, in CreateSaleInput) (domain.Sale, error) {
	if in.ProductID == "" || !in.EndsAt.After(in.StartsAt) || in.AllocatedStock <= 0 || in.MaxQuantityPerUser <= 0 {
		return domain.Sale{}, &domain.Error{Code: "VALIDATION_ERROR", Message: "sale schedule and quantities are invalid", HTTPStatus: 400}
	}
	id, err := domain.NewID()
	if err != nil {
		return domain.Sale{}, err
	}
	state := domain.SaleDraft
	if in.StartsAt.After(s.now()) {
		state = domain.SaleScheduled
	}
	var sale domain.Sale
	err = s.pool.QueryRow(ctx, `INSERT INTO flash_sales
		(id,product_id,starts_at,ends_at,allocated_stock,remaining_stock,max_quantity_per_user,state)
		VALUES ($1,$2,$3,$4,$5,$5,$6,$7)
		RETURNING id,product_id,starts_at,ends_at,allocated_stock,remaining_stock,max_quantity_per_user,state,created_at,updated_at`,
		id, in.ProductID, in.StartsAt, in.EndsAt, in.AllocatedStock, in.MaxQuantityPerUser, state).
		Scan(&sale.ID, &sale.ProductID, &sale.StartsAt, &sale.EndsAt, &sale.AllocatedStock, &sale.RemainingStock, &sale.MaxQuantityPerUser, &sale.State, &sale.CreatedAt, &sale.UpdatedAt)
	return sale, err
}

func (s *Store) ListSales(ctx context.Context) ([]domain.Sale, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,product_id,starts_at,ends_at,allocated_stock,remaining_stock,max_quantity_per_user,state,created_at,updated_at FROM flash_sales ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sales := make([]domain.Sale, 0)
	for rows.Next() {
		var sale domain.Sale
		if err := rows.Scan(&sale.ID, &sale.ProductID, &sale.StartsAt, &sale.EndsAt, &sale.AllocatedStock, &sale.RemainingStock, &sale.MaxQuantityPerUser, &sale.State, &sale.CreatedAt, &sale.UpdatedAt); err != nil {
			return nil, err
		}
		sales = append(sales, sale)
	}
	return sales, rows.Err()
}

func (s *Store) GetSale(ctx context.Context, id string) (domain.Sale, error) {
	var sale domain.Sale
	err := s.pool.QueryRow(ctx, `SELECT id,product_id,starts_at,ends_at,allocated_stock,remaining_stock,max_quantity_per_user,state,created_at,updated_at FROM flash_sales WHERE id=$1`, id).
		Scan(&sale.ID, &sale.ProductID, &sale.StartsAt, &sale.EndsAt, &sale.AllocatedStock, &sale.RemainingStock, &sale.MaxQuantityPerUser, &sale.State, &sale.CreatedAt, &sale.UpdatedAt)
	return sale, notFound(err)
}

func (s *Store) SetSaleState(ctx context.Context, id string, state domain.SaleState) (domain.Sale, error) {
	if state != domain.SaleActive && state != domain.SaleEnded && state != domain.SaleCancelled {
		return domain.Sale{}, &domain.Error{Code: "VALIDATION_ERROR", Message: "unsupported sale state", HTTPStatus: 400}
	}
	result, err := s.pool.Exec(ctx, `UPDATE flash_sales SET state=$2,updated_at=now() WHERE id=$1`, id, state)
	if err != nil {
		return domain.Sale{}, err
	}
	if result.RowsAffected() == 0 {
		return domain.Sale{}, ErrNotFound
	}
	return s.GetSale(ctx, id)
}
