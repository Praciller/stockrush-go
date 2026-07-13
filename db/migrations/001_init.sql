CREATE TABLE IF NOT EXISTS schema_migrations (
    version bigint PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS products (
    id uuid PRIMARY KEY,
    sku text NOT NULL UNIQUE,
    name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    description text NOT NULL DEFAULT '',
    price_minor bigint NOT NULL CHECK (price_minor >= 0),
    currency char(3) NOT NULL CHECK (currency = upper(currency)),
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS inventory (
    product_id uuid PRIMARY KEY REFERENCES products(id) ON DELETE CASCADE,
    available integer NOT NULL DEFAULT 0 CHECK (available >= 0),
    reserved integer NOT NULL DEFAULT 0 CHECK (reserved >= 0),
    sold integer NOT NULL DEFAULT 0 CHECK (sold >= 0),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS flash_sales (
    id uuid PRIMARY KEY,
    product_id uuid NOT NULL REFERENCES products(id),
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    allocated_stock integer NOT NULL CHECK (allocated_stock > 0),
    max_quantity_per_user integer NOT NULL DEFAULT 1 CHECK (max_quantity_per_user > 0),
    state text NOT NULL CHECK (state IN ('draft', 'scheduled', 'active', 'ended', 'cancelled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at)
);
CREATE INDEX IF NOT EXISTS flash_sales_product_state_idx ON flash_sales(product_id, state);

CREATE TABLE IF NOT EXISTS reservations (
    id uuid PRIMARY KEY,
    sale_id uuid NOT NULL REFERENCES flash_sales(id),
    product_id uuid NOT NULL REFERENCES products(id),
    user_id text NOT NULL CHECK (char_length(user_id) BETWEEN 1 AND 128),
    quantity integer NOT NULL CHECK (quantity > 0),
    state text NOT NULL CHECK (state IN ('pending', 'confirmed', 'paid', 'cancelled', 'expired')),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS reservations_expiration_idx ON reservations(state, expires_at);
CREATE INDEX IF NOT EXISTS reservations_user_sale_idx ON reservations(user_id, sale_id);

CREATE TABLE IF NOT EXISTS orders (
    id uuid PRIMARY KEY,
    reservation_id uuid NOT NULL UNIQUE REFERENCES reservations(id),
    sale_id uuid NOT NULL REFERENCES flash_sales(id),
    product_id uuid NOT NULL REFERENCES products(id),
    user_id text NOT NULL,
    quantity integer NOT NULL CHECK (quantity > 0),
    amount_minor bigint NOT NULL CHECK (amount_minor >= 0),
    currency char(3) NOT NULL,
    state text NOT NULL CHECK (state IN ('pending', 'confirmed', 'paid', 'cancelled', 'expired', 'payment_failed')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS orders_created_idx ON orders(created_at DESC);

CREATE TABLE IF NOT EXISTS payments (
    id uuid PRIMARY KEY,
    order_id uuid NOT NULL REFERENCES orders(id),
    idempotency_key text NOT NULL UNIQUE,
    outcome text NOT NULL CHECK (outcome IN ('successful', 'failed', 'delayed')),
    status text NOT NULL CHECK (status IN ('succeeded', 'failed', 'pending')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key text PRIMARY KEY,
    scope text NOT NULL,
    user_id text NOT NULL,
    request_fingerprint text NOT NULL,
    resource_id uuid,
    response_status integer,
    response_body jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS idempotency_expiration_idx ON idempotency_keys(expires_at);

CREATE TABLE IF NOT EXISTS domain_events (
    id uuid PRIMARY KEY,
    aggregate_type text NOT NULL,
    aggregate_id uuid NOT NULL,
    event_type text NOT NULL,
    previous_state text,
    new_state text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS domain_events_aggregate_idx ON domain_events(aggregate_type, aggregate_id, created_at);
