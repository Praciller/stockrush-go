# Limitations

- The token-bucket limiter is process-local. Multiple API replicas need a shared limiter if a global quota matters.
- Demo endpoints use a documented local token, not user authentication.
- Payment outcomes are synthetic and do not model settlement, refunds, or chargebacks.
- PostgreSQL service tests run sequentially because they reset shared synthetic tables.
- The OpenAPI document covers the main review path rather than every schema field.
- Static evidence is a bundled snapshot, not live status.
- Free-host deployment is optional and provider-neutral; availability is not guaranteed.
