# Requirements: StockRush Go

**Defined:** 2026-07-13  
**Core Value:** 1,000 buyers compete for 100 items and exactly 100 reservations succeed with zero overselling and no duplicate orders.

## v1 Requirements

### Foundation

- [x] **FND-01**: A developer can start PostgreSQL, API, worker, and frontend locally with Docker Compose.
- [x] **FND-02**: The API validates configuration and shuts down gracefully with propagated contexts and timeouts.
- [x] **FND-03**: Liveness reports process health and readiness verifies PostgreSQL connectivity.
- [x] **FND-04**: SQL migrations create normalized tables, constraints, foreign keys, and indexes.

### Catalog and Sales

- [x] **CAT-01**: A reviewer can create, list, view, update, activate, and deactivate products using integer minor-unit prices.
- [x] **CAT-02**: A reviewer can set inventory while non-negative inventory constraints remain enforced.
- [x] **SALE-01**: A reviewer can create, view, activate, and end a flash sale with schedule, stock allocation, and per-user limits.
- [x] **SALE-02**: Reservation attempts outside an active sale window fail with a typed response.

### Reservation Correctness

- [x] **RES-01**: A buyer can atomically reserve stock without inventory becoming negative.
- [x] **RES-02**: One reservation creates exactly one pending order and one auditable domain event.
- [x] **RES-03**: A buyer cannot exceed the sale quantity limit.
- [x] **RES-04**: Reservation transitions reject invalid state changes.
- [x] **IDEM-01**: A retry with the same user, sale, key, and payload returns the original result.
- [x] **IDEM-02**: Concurrent duplicate reservation attempts create no duplicate reservation or order.
- [x] **IDEM-03**: Reusing an idempotency key with a different payload is rejected.

### Expiration and Payment

- [x] **EXP-01**: Competing workers lock expired reservations safely and restore inventory exactly once.
- [x] **EXP-02**: Cancellation and expiration races leave inventory and reservation state consistent.
- [x] **PAY-01**: Synthetic successful, failed, delayed, and duplicate payment callbacks are idempotent.
- [x] **PAY-02**: Paid, cancelled, and expired reservations cannot enter invalid later states.

### API and Operations

- [x] **API-01**: Versioned REST endpoints return consistent success and error envelopes with request IDs.
- [x] **API-02**: Request bodies are bounded, validated, reject unknown fields, and return safe errors.
- [x] **OPS-01**: Per-user in-process token-bucket limiting returns clear HTTP 429 responses.
- [x] **OPS-02**: Prometheus metrics cover HTTP, reservations, duplicates, sold-out, rate-limit, expiration, restoration, and worker errors without unbounded labels.
- [x] **OPS-03**: Structured request and worker logs contain useful non-sensitive correlation fields.
- [x] **OPS-04**: OpenAPI documentation describes the public API.

### Demo and Frontend

- [x] **DEMO-01**: Demo reset seeds one active product, 100 units, and an active sale using synthetic data.
- [x] **DEMO-02**: A bounded 1,000-attempt load run produces exactly 100 reservations and proves zero overselling.
- [x] **DEMO-03**: A generated Markdown report derives values from actual API or database output.
- [x] **UI-01**: The reviewer UI explains the goal, sale, inventory proof, orders, architecture, and bounded load simulator.
- [x] **UI-02**: When the API is unavailable, the UI clearly labels and displays pre-generated deterministic evidence.
- [x] **UI-03**: The UI never permits an unbounded load request.

### Verification and Delivery

- [x] **TEST-01**: Unit tests cover state transitions, rate limiting, validation, and deterministic helpers.
- [x] **TEST-02**: PostgreSQL-backed tests repeatedly prove the 100-of-1,000 invariant, duplicate safety, worker safety, payment safety, and race handling.
- [x] **TEST-03**: A k6 scenario supports configurable base URL, sale ID, users, duration, and idempotency strategy.
- [x] **CI-01**: GitHub Actions run Go, database, frontend, migration, and Docker verification on free hosted runners.
- [x] **DOC-01**: Documentation explains architecture, database, API, concurrency, idempotency, worker, load testing, local demo, deployment, security, limitations, backlog, and a ten-minute review path.
- [x] **SEC-01**: Repository guardrails detect secrets, real user data, unsafe generated artifacts, and malformed configuration.

## v2 Requirements

### Optional Extensions

- **EXT-01**: Distributed rate limiting can use a shared store when multiple API instances require a global budget.
- **EXT-02**: Authentication and role-based administration can protect non-local deployments.
- **EXT-03**: A real payment adapter can be introduced behind the proven idempotent payment boundary.

## Out of Scope

| Feature | Reason |
|---------|--------|
| Kubernetes | Operational complexity is unnecessary for a local portfolio MVP. |
| Kafka | PostgreSQL transactions and events are sufficient for the required proof. |
| Redis | The database and in-process rate limiter cover MVP correctness and abuse prevention. |
| Real payment provider | Synthetic payment behavior avoids secrets, money, and vendor dependencies. |
| Required cloud deployment | Free plans change; the local demo remains authoritative. |

## Traceability

| Requirement group | Phase | Status |
|-------------------|-------|--------|
| FND-01..04 | Phase 2 | Complete |
| CAT-01..02, SALE-01..02 | Phase 3 | Complete |
| RES-01..04, IDEM-01..03, EXP-01..02 | Phase 4 | Complete |
| API-01..02, OPS-01..04, PAY-01..02, DEMO-01 | Phase 5 | Complete |
| UI-01..03 | Phase 6 | Complete |
| TEST-01..03 | Phase 7 | Complete |
| DEMO-02..03, DOC-01 | Phase 8 | Complete |
| CI-01, SEC-01 | Phase 9 | Complete |
| All v1 requirements | Phase 10 | Complete |

**Coverage:**
- v1 requirements: 37
- Mapped to phases: 37
- Unmapped: 0

---
*Requirements defined: 2026-07-13*
*Last updated: 2026-07-13 after full verification*
