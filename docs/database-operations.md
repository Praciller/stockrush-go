# Database Operations

## Roles and TLS

Production connections must use `sslmode=require` or `verify-full`. Use a non-superuser application role with DML access to application tables and sequences. A separate migration owner is preferred; the free single-container demo may use one non-superuser owner role and records that limitation.

Keep PostgreSQL private where the provider supports it. Neon Free exposes a TLS endpoint rather than private networking, so credentials, TLS, a small pool, and strict application boundaries are mandatory.

## Pool Sizing

`DATABASE_MIN_CONNS=1`, `DATABASE_MAX_CONNS=8`, and `DATABASE_MAX_CONN_LIFETIME=30m` are safe starting values for one small API/worker container. `DATABASE_STATEMENT_TIMEOUT=5s` and `DATABASE_TRANSACTION_TIMEOUT=10s` bound database work. Increase only from measured pool saturation.

## Migrations

```powershell
$env:DATABASE_URL='<migration-role TLS URL>'
go run ./cmd/migrate
```

Migrations are numeric, transactional, roll-forward only, and serialized with a PostgreSQL advisory lock. Back up before a destructive migration. Do not run two schema versions concurrently unless the change is explicitly backward compatible.

## Logical Backup

```powershell
docker compose exec -T postgres pg_dump -U stockrush -d stockrush -Fc -f /tmp/stockrush.dump
docker cp stockrush-go-postgres-1:/tmp/stockrush.dump .\tmp\stockrush.dump
```

For Neon, run `pg_dump --format=custom --file=<ignored-path> "$env:DATABASE_URL"` from an operator machine. Dump files contain credentials-derived data, remain outside Git, and use synthetic data only.

## Restore and Verification

```powershell
createdb stockrush_restore
pg_restore --dbname=stockrush_restore .\tmp\stockrush.dump
$env:DATABASE_URL='<fresh restore database URL>'
go run ./cmd/migrate
go run ./cmd/invariant-check
go run ./cmd/api
```

Verify `/health/ready`, then delete the temporary restore database and dump when no longer needed. SQL migrations have no automatic down path; recovery is restore plus roll-forward.

## Database Unavailability

- `/health/live` remains 200 while the process can serve.
- `/health/ready` returns 503 while PostgreSQL is unavailable.
- Request contexts and database timeouts bound failures.
- pgx reconnects when PostgreSQL returns; the worker retries on its next poll.
- If readiness does not recover, inspect connection count, TLS mode, credentials, Neon quota/suspension, and provider status.
