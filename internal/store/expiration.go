package store

import (
	"context"
	"fmt"
	"time"
)

type ExpirationResult struct {
	Processed int `json:"processed"`
	Expired   int `json:"expired"`
	Restored  int `json:"restored"`
}

func (s *Store) ExpireBatch(ctx context.Context, batchSize int, at time.Time) (ExpirationResult, error) {
	if batchSize <= 0 || batchSize > 1000 {
		return ExpirationResult{}, fmt.Errorf("batch size must be between 1 and 1000")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ExpirationResult{}, err
	}
	defer rollback(tx)
	rows, err := tx.Query(ctx, `WITH candidates AS (
		SELECT id FROM reservations
		WHERE state='pending' AND expires_at <= $2
		ORDER BY expires_at
		FOR UPDATE SKIP LOCKED
		LIMIT $1
	)
	UPDATE reservations r SET state='expired',updated_at=$2
	FROM candidates c WHERE r.id=c.id AND r.state='pending'
	RETURNING r.id,r.product_id,r.sale_id,r.quantity`, batchSize, at.UTC())
	if err != nil {
		return ExpirationResult{}, err
	}
	type expiredReservation struct {
		id        string
		productID string
		saleID    string
		quantity  int
	}
	expired := make([]expiredReservation, 0, batchSize)
	for rows.Next() {
		var reservation expiredReservation
		if err := rows.Scan(&reservation.id, &reservation.productID, &reservation.saleID, &reservation.quantity); err != nil {
			rows.Close()
			return ExpirationResult{}, err
		}
		expired = append(expired, reservation)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return ExpirationResult{}, err
	}
	rows.Close()

	for _, reservation := range expired {
		result, err := tx.Exec(ctx, `UPDATE inventory SET available=available+$2,reserved=reserved-$2,updated_at=$3
			WHERE product_id=$1 AND reserved >= $2`, reservation.productID, reservation.quantity, at.UTC())
		if err != nil {
			return ExpirationResult{}, err
		}
		if result.RowsAffected() != 1 {
			return ExpirationResult{}, fmt.Errorf("inventory invariant prevented restoration for reservation %s", reservation.id)
		}
		if _, err := tx.Exec(ctx, `UPDATE flash_sales SET remaining_stock=remaining_stock+$2,updated_at=$3 WHERE id=$1`, reservation.saleID, reservation.quantity, at.UTC()); err != nil {
			return ExpirationResult{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE orders SET state='expired',updated_at=$2 WHERE reservation_id=$1`, reservation.id, at.UTC()); err != nil {
			return ExpirationResult{}, err
		}
		if err := event(ctx, tx, "reservation", reservation.id, "reservation.expired", "pending", "expired", map[string]any{"restored": reservation.quantity}); err != nil {
			return ExpirationResult{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ExpirationResult{}, err
	}
	count := len(expired)
	return ExpirationResult{Processed: count, Expired: count, Restored: count}, nil
}
