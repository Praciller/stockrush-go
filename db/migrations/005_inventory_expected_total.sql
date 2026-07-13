ALTER TABLE inventory ADD COLUMN IF NOT EXISTS expected_total integer;

UPDATE inventory
SET expected_total = available + reserved + sold
WHERE expected_total IS NULL;

ALTER TABLE inventory ALTER COLUMN expected_total SET NOT NULL;
ALTER TABLE inventory ALTER COLUMN expected_total SET DEFAULT 0;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'inventory_expected_total_nonnegative'
    ) THEN
        ALTER TABLE inventory ADD CONSTRAINT inventory_expected_total_nonnegative CHECK (expected_total >= 0);
    END IF;
END $$;
