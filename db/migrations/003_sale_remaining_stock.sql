ALTER TABLE flash_sales
    ADD COLUMN IF NOT EXISTS remaining_stock integer NOT NULL DEFAULT 0 CHECK (remaining_stock >= 0);

UPDATE flash_sales s
SET remaining_stock = GREATEST(
    s.allocated_stock - COALESCE((
        SELECT sum(r.quantity)
        FROM reservations r
        WHERE r.sale_id = s.id AND r.state IN ('pending', 'confirmed', 'paid')
    ), 0),
    0
);
