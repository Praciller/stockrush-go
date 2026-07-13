package store

import "context"

type InvariantViolation struct {
	Code  string `json:"code"`
	Count int    `json:"count"`
}

func (s *Store) CheckInvariants(ctx context.Context) ([]InvariantViolation, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT code, count FROM (
			SELECT 'INVENTORY_RECONCILIATION' code, count(*)::int count FROM inventory WHERE available+reserved+sold <> expected_total
			UNION ALL
			SELECT 'NEGATIVE_INVENTORY', count(*)::int FROM inventory WHERE available < 0 OR reserved < 0 OR sold < 0
			UNION ALL
			SELECT 'NEGATIVE_SALE_STOCK', count(*)::int FROM flash_sales WHERE remaining_stock < 0
			UNION ALL
			SELECT 'ACTIVE_RESERVATION_LIMIT', count(*)::int FROM (
				SELECT r.sale_id,r.user_id FROM reservations r JOIN flash_sales s ON s.id=r.sale_id
				WHERE r.state IN ('pending','confirmed','paid') GROUP BY r.sale_id,r.user_id,s.max_quantity_per_user
				HAVING sum(r.quantity) > s.max_quantity_per_user
			) excessive
			UNION ALL
			SELECT 'ORDER_WITHOUT_RESERVATION', count(*)::int FROM orders o LEFT JOIN reservations r ON r.id=o.reservation_id WHERE r.id IS NULL
			UNION ALL
			SELECT 'ORDER_RESERVATION_STATE_MISMATCH', count(*)::int FROM orders o JOIN reservations r ON r.id=o.reservation_id
			WHERE r.state IN ('paid','cancelled','expired') AND o.state <> r.state
			UNION ALL
			SELECT 'PAID_EXPIRED_RESERVATION', count(*)::int FROM payments p JOIN orders o ON o.id=p.order_id
			JOIN reservations r ON r.id=o.reservation_id WHERE p.status='succeeded' AND r.state='expired'
		) checks WHERE count > 0 ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	violations := make([]InvariantViolation, 0)
	for rows.Next() {
		var violation InvariantViolation
		if err := rows.Scan(&violation.Code, &violation.Count); err != nil {
			return nil, err
		}
		violations = append(violations, violation)
	}
	return violations, rows.Err()
}
