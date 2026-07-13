# Limitations

- The per-IP token-bucket limiter is process-local. The anonymous public reservation also has a PostgreSQL-backed global budget, but multiple replicas still have independent per-IP buckets.
- Operator access is one hashed API key, not user identity or role-based access control.
- Payment outcomes are synthetic and do not model settlement, refunds, or chargebacks.
- PostgreSQL service tests run sequentially because they reset shared synthetic tables.
- The OpenAPI document covers the main review path rather than every schema field.
- Static evidence is a bundled snapshot, not live status.
- The evaluated free host sleeps, co-locates API and worker, has no SLA, and uses a public TLS database endpoint with tight storage/egress/restore limits.
- Metrics do not yet instrument every individual database query failure; HTTP errors, readiness, worker errors, provider metrics, and manual SQL are used together.
