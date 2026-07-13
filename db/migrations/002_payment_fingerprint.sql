ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS request_fingerprint text NOT NULL DEFAULT '';
