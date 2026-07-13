# Production Readiness

This document preserves two explicit repository snapshots:

- **Pre-hardening audit baseline:** 2026-07-13 at commit `e8749e4bc6eb366beeee8f160588f8c6e7b680c2`.
- **Final hardened repository state:** 2026-07-13 at commit `b6d6770d83ee3022b561637b985b452d84fdc300`.

## Current Status at `b6d6770`

| Surface / level | Status |
|---|---|
| Static portfolio | Verified |
| Local full-stack | Verified |
| Public Go API | Not deployed |
| Public PostgreSQL | Not deployed |
| Public worker | Not deployed |
| Level B | Implementation-ready locally but not publicly verified |
| Commercial production | Not claimed |

## Readiness Levels

- **Level A — Static Portfolio Ready:** verified. GitHub Pages serves deterministic evidence and clearly states that the API is offline.
- **Level B — Public Production-Like Demo Ready:** implementation-ready and verified locally, but not publicly verified because the API, database, and worker are not deployed.
- **Level C — Commercial Production Ready:** not claimed. Real identity, payments, legal/privacy controls, automated backups, incident response, support, and reliable paid infrastructure are intentionally outside this portfolio demo.

## Post-Hardening Control Summary at `b6d6770`

| Control | Final state | Repository evidence |
|---|---|---|
| Production configuration | `APP_ENV=production` fails closed on insecure URLs, TLS, CORS, logging, demo controls, and weak admin credentials | `internal/config/config.go`, `internal/config/config_test.go`, `.env.example` |
| Authentication and route policy | Operator mutations use a hashed bearer key; development controls return 404 in production | `internal/httpserver/router.go`, `internal/httpserver/server_test.go`, `docs/security.md` |
| Bounded public mutation | One synthetic quantity-one reservation path uses server-generated identity, idempotency, per-client limiting, and a PostgreSQL global budget | `internal/httpserver/demo.go`, `internal/store/demo.go`, `db/migrations/004_public_demo_actions.sql` |
| Proxy and HTTP hardening | Forwarded client IPs are accepted only from configured proxy CIDRs; JSON, body, header, CORS, list, timeout, health, and version boundaries are explicit | `internal/httpserver/router.go`, `cmd/api/main.go`, `internal/config/config.go` |
| Database safety | TLS validation, explicit pool/session timeouts, advisory-locked migrations, reconciliation state, and an operator invariant command are implemented | `internal/database/database.go`, `db/migrations/005_inventory_expected_total.sql`, `cmd/invariant-check/main.go`, [Database operations](database-operations.md) |
| Worker reliability | The reusable expiration loop updates shared metrics and was verified through worker/database/API recovery drills | `internal/expirationworker/worker.go`, `cmd/worker/main.go`, [Failure recovery report](../reports/failure_recovery_report.md) |
| Container and dependency security | Go 1.26.5 builds into a non-root distroless runtime with a compiled healthcheck; CI runs Go/npm vulnerability checks and Trivy | `Dockerfile`, `.github/workflows/ci.yml` |
| Operational evidence | Restore, failure, correctness, capacity, and soak checks completed locally with synthetic data | [Restore drill](../reports/restore_drill_report.md), [Failure recovery](../reports/failure_recovery_report.md), [Capacity and soak](../reports/production_capacity_report.md), [Local correctness](../reports/local_portfolio_report.md) |
| Operations guidance | Deployment, rollback, backup/restore, diagnostics, and emergency read-only procedures are documented | [Production runbook](production-runbook.md), [Database operations](database-operations.md), [Observability](observability.md) |
| Hosting decision | A permanent-free candidate was documented, but public deployment stopped when database provisioning could not be completed | [Hosting evaluation](hosting-evaluation.md) |

Level B remains **not publicly verified** until compliant database and API resources are provisioned, production secrets are configured, GitHub Pages is pointed at the deployed API, and public smoke, expiration, restart, and availability checks pass.

## Pre-Hardening Audit Matrix at `e8749e4`

Every row below describes the original audit baseline, not the final state at `b6d6770`. The post-hardening control summary above is the current repository status.

| Area | Current state | Evidence | Risk | Required change | Verification method | Status |
|---|---|---|---|---|---|---|
| Authentication | Local demo token works only in development; all other writes are anonymous | `internal/httpserver/router.go`, `docs/security.md` | Public callers can create products, sales, reservations, transitions, and synthetic payments | Require a production API key for privileged mutations; expose only one bounded anonymous reservation path | Production router tests for 401/404 and constant-time key verification | Gap |
| Authorization | No roles or endpoint policy | `internal/httpserver/router.go` | Any caller can invoke every non-demo mutation | Split public reads, bounded public demo mutation, and authenticated operator routes | Route table and integration tests | Gap |
| Demo endpoint exposure | Reset/load require development token, but status/report are public and privileged non-demo writes remain exposed | `internal/httpserver/demo.go` | Destructive controls are hidden only by environment; broad writes remain | Return 404 for development-only controls in production; disable public mutation by switch | Production-mode endpoint tests | Partial |
| Rate limiting | Bounded in-process token buckets; identity trusts `X-User-ID` | `internal/httpserver/router.go`, `internal/ratelimit/` | Header spoofing bypasses limits; replicas have independent budgets | Derive client IP from trusted proxies only; add a database-backed global public-demo bound | Proxy tests, multi-instance integration test | Gap |
| Database security | Parameterized SQL and local credentials; Compose publishes PostgreSQL to the host | `docker-compose.yml`, `internal/store/` | No least-privilege production role or remote TLS enforcement | Validate TLS in production, document app/migration roles, keep cloud database private | Production config tests and role grants review | Gap |
| Secret management | `.env` ignored and basic secret markers scanned; local demo token is embedded in frontend default | `.gitignore`, `cmd/guardrails`, `web/src/api.ts` | A production build could accidentally contain a privileged token | Remove privileged frontend token path; validate placeholders/defaults in production | Built-asset scan and config tests | Gap |
| Migration safety | Ordered transactional SQL with version table | `internal/database/database.go`, `db/migrations/` | Concurrent migrators can race; no checksum or advisory lock | Add a PostgreSQL advisory migration lock and operational procedure | Concurrent migration integration test | Gap |
| Backup and restore | No backup procedure or restore evidence | Existing docs contain no runbook | Data loss cannot be recovered or demonstrated | Add `pg_dump`/`pg_restore` commands and complete a synthetic restore drill | `reports/restore_drill_report.md` | Gap |
| Worker reliability | Batched `FOR UPDATE SKIP LOCKED`; errors are logged and retried next tick | `cmd/worker/main.go`, `internal/store/expiration.go` | No backoff/readiness signal; recovery after DB restart is unproven | Bound each batch, expose useful metrics, verify outage/restart recovery | Failure-injection report | Partial |
| Multi-instance correctness | Reservation, idempotency, payment, cancellation, and expiration use PostgreSQL locks/constraints | `internal/store/store_integration_test.go` | Tests use one pool/store and omit API restart/reconnect cases | Add independent-pool/process-equivalent tests and database-backed public-demo budget | Integration and restart tests | Partial |
| HTTP proxy behavior | Uses `RemoteAddr`; ignores forwarded headers; `X-User-ID` can override identity | `internal/httpserver/router.go` | Incorrect rate-limit identity behind proxy and spoofable identity | Configure trusted proxy CIDRs and trust forwarded headers only from them | Unit tests for trusted/untrusted proxies | Gap |
| TLS termination | No application redirect or trusted forwarded scheme handling | `cmd/api/main.go` | Incorrect scheme assumptions behind TLS terminator | Document TLS termination and validate trusted `X-Forwarded-Proto` where needed | Proxy integration tests and deployment check | Gap |
| CORS | Explicit origin allowlist and no credentials | `internal/httpserver/router.go` | Production wildcard/placeholder is not rejected; disallowed preflight still returns 204 | Fail closed in production and reject disallowed preflight | CORS tests | Partial |
| CSRF relevance | No cookie authentication; API keys would use headers | Current API contract | Low while credentials are never ambient | Keep cookie auth out; document why CSRF is not applicable | Security review | Complete |
| Request validation | 1 MiB body cap, unknown fields rejected, one JSON value required | `decodeJSON` in `internal/httpserver/router.go` | Content-Type is not validated; path/query and list bounds are incomplete | Require JSON media type for bodies and bound list responses/query inputs | HTTP tests | Partial |
| Logging | `slog` request logs include request ID and duration | `cmd/api/main.go`, `internal/httpserver/router.go` | Production logs use default text handler; build version/error code absent | Configure JSON logs in production and include safe build/error fields | Captured log tests/manual check | Gap |
| Metrics | HTTP, reservation, sold-out, duplicate, and rate-limit metrics exist without user labels | `internal/metrics/metrics.go` | Worker metrics are registered only in API and never updated; no DB pool/query/lag/reconciliation metrics; metrics are public | Share/emit worker metrics, add bounded operational metrics, protect or disable public metrics | Prometheus scrape and metric assertions | Gap |
| Alerting | None; paid monitoring is not required | No alerting configuration | Failures depend on manual observation | Document local/manual thresholds and diagnostic queries | `docs/observability.md` review | Gap |
| Dependency security | Locked Go/npm dependencies and CI tests; no vulnerability jobs | `go.sum`, `web/package-lock.json`, `.github/workflows/ci.yml` | Reachable vulnerabilities may go unnoticed | Run `govulncheck`, `npm audit`, and an available free container scanner; document unavailable databases | CI/local scan evidence | Gap |
| Container security | Multi-stage, non-root Alpine image, no compiler in runtime | `Dockerfile` | No healthcheck, read-only proof, pinned digest, SBOM, or scan report | Add healthcheck/read-only compatibility and generate scan/SBOM evidence where tools permit | Image smoke test and scan report | Partial |
| Capacity | Deterministic 1,000-attempt correctness proof only | `reports/local_portfolio_report.md` | No throughput envelope or resource measurements | Run bounded concurrency profiles and 10–15 minute local soak | `reports/production_capacity_report.md` | Gap |
| Abuse resistance | Demo load values bounded; broad mutation surface and list endpoints are unbounded | `internal/httpserver/demo.go`, `internal/httpserver/router.go` | Public state mutation, enumeration, and resource exhaustion | Restrict production routes, paginate lists, cap public actions globally, add emergency switch | Abuse-focused HTTP/integration tests | Gap |
| Operational recovery | Graceful API shutdown exists; no outage/restart drill | `cmd/api/main.go` | Recovery behavior and data integrity after failures are unproven | Run isolated PostgreSQL/API/worker failure injection | `reports/failure_recovery_report.md` | Gap |
| Data retention | Synthetic rows and idempotency expiry timestamp exist; no cleanup job/policy | `db/migrations/001_init.sql` | Tables grow indefinitely | Document retention; add only the minimal safe cleanup required for a public demo | Retention query/runbook check | Gap |
| Privacy | Synthetic data only by design; user IDs/IP handling not formally documented | `.planning/PROJECT.md`, `docs/security.md` | Public callers could submit personal data in arbitrary `userId` | Generate public-demo identity server-side and exclude IP/user identifiers from logs/metrics | API response/log review | Partial |
| Deployment rollback | Provider-neutral deployment notes only | `docs/deployment.md` | No versioned rollback or migration compatibility procedure | Add image/commit rollback and roll-forward database guidance | `docs/production-runbook.md` review | Gap |
| Health/version | Liveness and database readiness exist | `internal/httpserver/router.go` | No `/version`; readiness checks only `Ping` | Add safe build metadata endpoint and retain distinct liveness/readiness semantics | HTTP tests and runtime curl | Partial |
| HTTP server limits | Read/header/write/idle timeouts and graceful shutdown are explicit | `cmd/api/main.go` | Shutdown timeout and max headers are not configurable; `MaxHeaderBytes` unset | Add bounded production configuration | Config and server tests | Partial |
| Database timeouts/pool | Contexts flow to queries; pgx defaults are used | `internal/database/database.go`, store methods | Pool size/lifetime and statement/transaction timeouts are uncontrolled | Configure pool bounds and session timeouts | Pool config tests and database inspection | Gap |
| Database invariants | Non-negative checks, foreign keys, unique reservation/order mapping, and reconciliation logic exist | `db/migrations/001_init.sql`, `internal/store/demo.go` | Invariant checker is tied to first demo rows and is not an operator command | Add a read-only all-row invariant command that exits non-zero | `go run ./cmd/invariant-check` | Partial |

## Baseline Controls Already Present at `e8749e4`

- PostgreSQL is the concurrency boundary; conditional updates and constraints prevent negative inventory.
- Reservation and payment idempotency are database-backed.
- Competing expiration workers use `FOR UPDATE SKIP LOCKED`.
- Request IDs, panic recovery, safe error envelopes, body limits, security headers, and explicit server timeouts exist.
- The container is multi-stage and runs as a non-root user.
- The public static frontend is verified and clearly labels deterministic fallback evidence.

## Original Level B Gaps at `e8749e4`

1. Fail-closed production configuration and route policy.
2. Authenticated operator mutations plus one bounded anonymous synthetic reservation path.
3. Trusted-proxy client identity and database-backed global public-demo limits.
4. Production database pool, TLS, migration lock, least-privilege, backup, restore, and invariant procedures.
5. Production logging/metrics, worker recovery, failure injection, capacity, and soak evidence.
6. Current official hosting evaluation satisfying the permanent-free, no-card, no-trial, no-auto-billing constraint.
7. Real public end-to-end verification if and only if item 6 succeeds.

At `b6d6770`, the locally actionable implementation and evidence work in items 1–6 is complete. Item 7 remains open because the public API, PostgreSQL database, and worker were not deployed.

## Deferred Level C Controls

Real users and identity, real payment processing, legal/privacy compliance, automated off-site backups, staffed monitoring/incident response, customer support, and reliable paid infrastructure remain deferred and are required before any commercial-production claim.

## Evidence Links

- [Architecture](architecture.md)
- [Security baseline](security.md)
- [Database design](database.md)
- [Concurrency proof](concurrency.md)
- [Production capacity report](../reports/production_capacity_report.md)
- [Failure recovery report](../reports/failure_recovery_report.md)
- [Restore drill report](../reports/restore_drill_report.md)
- [Local portfolio report](../reports/local_portfolio_report.md)
- [Hosting evaluation](hosting-evaluation.md)
- [Production runbook](production-runbook.md)
- [Database operations](database-operations.md)
- [Observability](observability.md)
- [Static portfolio](https://praciller.github.io/stockrush-go/)
