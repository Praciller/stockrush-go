package store

import (
	"context"

	"stockrush-go/internal/domain"
)

func (s *Store) ListOrders(ctx context.Context) ([]domain.Order, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,reservation_id,sale_id,product_id,user_id,quantity,amount_minor,currency,state,created_at,updated_at FROM orders ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orders := make([]domain.Order, 0)
	for rows.Next() {
		var order domain.Order
		if err := rows.Scan(&order.ID, &order.ReservationID, &order.SaleID, &order.ProductID, &order.UserID, &order.Quantity, &order.AmountMinor, &order.Currency, &order.State, &order.CreatedAt, &order.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (s *Store) GetOrder(ctx context.Context, id string) (domain.Order, error) {
	var order domain.Order
	err := s.pool.QueryRow(ctx, `SELECT id,reservation_id,sale_id,product_id,user_id,quantity,amount_minor,currency,state,created_at,updated_at FROM orders WHERE id=$1`, id).
		Scan(&order.ID, &order.ReservationID, &order.SaleID, &order.ProductID, &order.UserID, &order.Quantity, &order.AmountMinor, &order.Currency, &order.State, &order.CreatedAt, &order.UpdatedAt)
	return order, notFound(err)
}
