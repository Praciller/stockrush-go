CREATE TABLE IF NOT EXISTS public_demo_actions (
    idempotency_key text PRIMARY KEY CHECK (char_length(idempotency_key) BETWEEN 1 AND 200),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS public_demo_actions_created_idx ON public_demo_actions(created_at);
