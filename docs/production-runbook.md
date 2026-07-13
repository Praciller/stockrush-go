# Production Runbook

## Deploy

1. Re-check `docs/hosting-evaluation.md`.
2. Build from a reviewed commit and pass `scripts/tasks.ps1 verify`.
3. Set production variables through provider secrets; never frontend assets.
4. Run migrations once under the advisory lock.
5. Seed only synthetic demo data with `cmd/seed`.
6. Start API with `RUN_WORKER=true` on the single free container.
7. Verify liveness, readiness, version, CORS, unauthorized routes, public reservation, expiration, and invariants.

Required variables are documented in `.env.example`. Production additionally requires HTTPS `PUBLIC_BASE_URL`, TLS `DATABASE_URL`, HTTPS CORS origins, and a 64-character SHA-256 `ADMIN_API_KEY_HASH`.

## Rollback

Redeploy the previous known-good commit/image. Database changes are roll-forward: restore a pre-change logical backup into a fresh database when compatibility cannot be preserved. Never force-push deployment history.

## Common Failures

- **Database connections exhausted:** lower `DATABASE_MAX_CONNS`, stop duplicate instances, inspect `pg_stat_activity`, then restart one instance.
- **Worker not processing:** check worker error logs, PostgreSQL readiness, `WORKER_POLL_INTERVAL`, and overdue pending query.
- **Reservation backlog:** disable public mutation, run the invariant command, then restart the worker.
- **Invariant failure:** immediately set `PUBLIC_MUTATIONS_ENABLED=false`, preserve logs and a logical dump, and investigate before re-enabling.
- **Cold start:** keep static fallback visible; Hugging Face free CPU and Neon compute both sleep.

## Emergency Controls

- Set `PUBLIC_MUTATIONS_ENABLED=false` and restart to return 503 for anonymous reservations while reads remain available.
- Disable the deployment or pause the Space to shut down safely.
- Rotate an admin key by generating a new strong value, storing only its SHA-256 hash server-side, redeploying, and invalidating the old operator copy.
- Never expose database ports, raw keys, or migration credentials in frontend code.

## Backup and Restore

Follow `docs/database-operations.md`. Record timestamp, size, migration version, invariant result, and restored API readiness for every drill.
