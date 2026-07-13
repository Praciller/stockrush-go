# StockRush Go

Concurrency-safe flash-sale and inventory-reservation platform built with Go and PostgreSQL. It demonstrates atomic inventory updates, idempotent checkout, expiring reservations, synthetic payments, bounded load simulation, and a reproducible zero-oversell proof.

> **1,000 buyers. 100 items. Exactly 100 successful reservations. Zero overselling.**

## Verified result

| Measure | Result |
|---|---:|
| Initial inventory | 100 |
| Concurrent attempts | 1,000 |
| Successful reservations | 100 |
| Sold-out responses | 900 |
| Final available | 0 |
| Negative stock events | 0 |
| Duplicate orders | 0 |
| Zero overselling | **PASS** |

See the generated [local portfolio report](reports/local_portfolio_report.md).

## Screenshot

![StockRush Go mobile reviewer interface](docs/assets/stockrush-mobile.png)

The React interface reports live PostgreSQL state when the API is available. If it is offline, the page explicitly labels and displays deterministic pre-generated evidence.

## Architecture

```mermaid
flowchart LR
    Reviewer["Reviewer / k6"] --> UI["React evidence UI"]
    UI --> API["Go HTTP API"]
    API --> DB[("PostgreSQL")]
    Worker["Expiration worker"] --> DB
    Loadgen["Go demo load generator"] --> DB
    DB --> Report["Generated Markdown report"]
```

This is a modular monolith. PostgreSQL is the correctness boundary. The reservation transaction claims an idempotency key, validates the sale and per-user limit, conditionally decrements available stock, and creates the reservation, order, and event together.

## Quickstart

Requirements: Docker Desktop with Compose. GNU Make is optional.

```bash
docker compose up --build
```

- Frontend: <http://localhost:5173>
- API: <http://localhost:8080>
- OpenAPI: <http://localhost:8080/openapi.yaml>
- Metrics: <http://localhost:8080/metrics>

## One-command demo

With Make:

```bash
make demo
```

Windows PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\tasks.ps1 demo
```

The command starts the local stack, runs 1,000 bounded purchase attempts, validates database invariants, writes `reports/local_portfolio_report.md`, and exits non-zero on failure.

## Local development

```bash
make setup
make migrate
make seed
make api
make worker
make frontend
```

PowerShell equivalents:

```powershell
.\scripts\tasks.ps1 setup
$env:DATABASE_URL='postgres://stockrush:stockrush@localhost:5432/stockrush?sslmode=disable'
.\scripts\tasks.ps1 migrate
.\scripts\tasks.ps1 api
```

Copy `.env.example` to `.env` only for local convenience. Never commit `.env`.

## Testing and verification

```bash
make test
make test-integration
make verify
```

PowerShell:

```powershell
.\scripts\tasks.ps1 verify
```

`verify` checks Go formatting and vet, unit and race tests, PostgreSQL-backed integration tests, frontend type checks/tests/build, migrations, Compose configuration, Git whitespace, and repository guardrails.

## Load testing

```bash
make load-test-small
make load-test
make load-test-demo
```

The k6 scenario accepts `API_BASE_URL`, `SALE_ID`, `VUS`, `DURATION`, and `IDEMPOTENCY_STRATEGY` (`unique`, `user`, or `shared`). See [load testing](docs/load-testing.md).

## Deployment modes

1. **Fully local:** Docker Compose, PostgreSQL, API, worker, and frontend. Primary and authoritative.
2. **Static portfolio:** build `web/` and deploy its deterministic fallback to any static host.
3. **Optional live demo:** static host, container host, and PostgreSQL provider selected by the operator.

> Free-plan limits and availability may change. The local Docker Compose demo remains the authoritative deployment path.

## Important files

- Atomic reservation transaction: [`internal/store/reservation.go`](internal/store/reservation.go)
- Expiration worker transaction: [`internal/store/expiration.go`](internal/store/expiration.go)
- Database constraints: [`db/migrations/001_init.sql`](db/migrations/001_init.sql)
- Concurrency proof: [`internal/store/store_integration_test.go`](internal/store/store_integration_test.go)
- Reviewer UI: [`web/src/App.tsx`](web/src/App.tsx)
- Generated evidence: [`reports/local_portfolio_report.md`](reports/local_portfolio_report.md)

## Documentation

[Architecture](docs/architecture.md) · [Database](docs/database.md) · [API](docs/api.md) · [Concurrency](docs/concurrency.md) · [Idempotency](docs/idempotency.md) · [Worker](docs/reservation-worker.md) · [Local demo](docs/local-demo.md) · [Security](docs/security.md) · [Deployment](docs/deployment.md) · [Portfolio review](docs/portfolio-review.md) · [Limitations](docs/limitations.md) · [Backlog](docs/backlog.md)

## Limitations

The MVP uses a process-local rate limiter, synthetic payments, and a local demo token. Authentication, distributed rate limiting, external payment providers, Kafka, Redis, Kubernetes, and microservices are intentionally excluded. See [limitations](docs/limitations.md).
