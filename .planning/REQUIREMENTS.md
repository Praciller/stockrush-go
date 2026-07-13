# Requirements: StockRush Go

**Defined:** 2026-07-13  
**Core Value:** 1,000 buyers compete for 100 items and exactly 100 reservations succeed with zero overselling and no duplicate orders.

## v1 Requirements

### Foundation

- [ ] **FND-01**: A developer can start PostgreSQL, API, worker, and frontend locally with Docker Compose.
- [ ] **FND-02**: The API validates configuration and shuts down gracefully with propagated contexts and timeouts.
- [ ] **FND-03**: Liveness reports process health and readiness verifies PostgreSQL connectivity.
- [ ] **FND-04**: SQL migrations create normalized tables, constraints, foreign keys, and indexes.

### Catalog and Sales

- [ ] **CAT-01**: A reviewer can create, list, view, update, activate, and deactivate products using integer minor-unit prices.
- [ ] **CAT-02**: A reviewer can set inventory while non-negative inventory constraints remain enforced.
- [ ] **SALE-01**: A reviewer can create, view, activate, and end a flash sale with schedule, stock allocation, and per-user limits.
- [ ] **SALE-02**: Reservation attempts outside an active sale window fail with a typed response.

### Reservation Correctness

- [ ] **RES-01**: A buyer can atomically reserve stock without inventory becoming negative.
- [ ] **RES-02**: One reservation creates exactly one pending order and one auditable domain event.
- [ ] **RES-03**: A buyer cannot exceed the sale quantity limit.
- [ ] **RES-04**: Reservation transitions reject invalid state changes.
- [ ] **IDEM-01**: A retry with the same user, sale, key, and payload returns the original result.
- [ ] **IDEM-02**: Concurrent duplicate reservation attempts create no duplicate reservation or order.
- [ ] **IDEM-03**: Reusing an idempotency key with a different payload is rejected.

### Expiration and Payment

- [ ] **EXP-01**: Competing workers lock expired reservations safely and restore inventory exactly once.
- [ ] **EXP-02**: Cancellation and expiration races leave inventory and reservation state consistent.
- [ ] **PAY-01**: Synthetic successful, failed, delayed, and duplicate payment callbacks are idempotent.
- [ ] **PAY-02**: Paid, cancelled, and expired reservations cannot enter invalid later states.

### API and Operations

- [ ] **API-01**: Versioned REST endpoints return consistent success and error envelopes with request IDs.
- [ ] **API-02**: Request bodies are bounded, validated, reject unknown fields, and return safe errors.
- [ ] **OPS-01**: Per-user in-process token-bucket limiting returns clear HTTP 429 responses.
- [ ] **OPS-02**: Prometheus metrics cover HTTP, reservations, duplicates, sold-out, rate-limit, expiration, restoration, and worker errors without unbounded labels.
- [ ] **OPS-03**: Structured request and worker logs contain useful non-sensitive correlation fields.
- [ ] **OPS-04**: OpenAPI documentation describes the public API.

### Demo and Frontend

- [ ] **DEMO-01**: Demo reset seeds one active product, 100 units, and an active sale using synthetic data.
- [ ] **DEMO-02**: A bounded 1,000-attempt load run produces exactly 100 reservations and proves zero overselling.
- [ ] **DEMO-03**: A generated Markdown report derives values from actual API or database output.
- [ ] **UI-01**: The reviewer UI explains the goal, sale, inventory proof, orders, architecture, and bounded load simulator.
- [ ] **UI-02**: When the API is unavailable, the UI clearly labels and displays pre-generated deterministic evidence.
- [ ] **UI-03**: The UI never permits an unbounded load request.

### Verification and Delivery

- [ ] **TEST-01**: Unit tests cover state transitions, rate limiting, validation, and deterministic helpers.
- [ ] **TEST-02**: PostgreSQL-backed tests repeatedly prove the 100-of-1,000 invariant, duplicate safety, worker safety, payment safety, and race handling.
- [ ] **TEST-03**: A k6 scenario supports configurable base URL, sale ID, users, duration, and idempotency strategy.
- [ ] **CI-01**: GitHub Actions run Go, database, frontend, migration, and Docker verification on free hosted runners.
- [ ] **DOC-01**: Documentation explains architecture, database, API, concurrency, idempotency, worker, load testing, local demo, deployment, security, limitations, backlog, and a ten-minute review path.
- [ ] **SEC-01**: Repository guardrails detect secrets, real user data, unsafe generated artifacts, and malformed configuration.

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
| FND-01..04 | Phase 2 | Pending |
| CAT-01..02, SALE-01..02 | Phase 3 | Pending |
| RES-01..04, IDEM-01..03, EXP-01..02 | Phase 4 | Pending |
| API-01..02, OPS-01..04, PAY-01..02, DEMO-01 | Phase 5 | Pending |
| UI-01..03 | Phase 6 | Pending |
| TEST-01..03 | Phase 7 | Pending |
| DEMO-02..03, DOC-01 | Phase 8 | Pending |
| CI-01, SEC-01 | Phase 9 | Pending |
| All v1 requirements | Phase 10 | Pending |

**Coverage:**
- v1 requirements: 37
- Mapped to phases: 37
- Unmapped: 0

---
*Requirements defined: 2026-07-13*
*Last updated: 2026-07-13 after initial definition*
