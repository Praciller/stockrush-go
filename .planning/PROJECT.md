# StockRush Go

## What This Is

StockRush Go is a portfolio-ready, concurrency-safe flash-sale and inventory-reservation platform built with Go and PostgreSQL. It gives reviewers a reproducible local proof that database-backed atomic reservation, idempotency, and expiration processing prevent overselling under heavy contention.

## Core Value

1,000 buyers compete for 100 items and exactly 100 reservations succeed with zero overselling and no duplicate orders.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Run a complete local demo with Go, PostgreSQL, worker, and reviewer UI.
- [ ] Protect inventory with atomic SQL and database constraints.
- [ ] Make reservation and payment retries idempotent under concurrency.
- [ ] Restore expired inventory exactly once across competing workers.
- [ ] Produce repeatable tests and generated portfolio evidence.
- [ ] Remain understandable through a clearly labelled static fallback when the API is offline.

### Out of Scope

- Authentication — unnecessary for the local synthetic portfolio demo.
- Real payments — synthetic outcomes prove state handling without external risk.
- Redis, Kafka, Kubernetes, and microservices — add operational cost without strengthening the MVP proof.
- Required cloud infrastructure — the local Docker Compose path is authoritative.

## Context

The repository was empty except for Git metadata on 2026-07-13. The supplied project brief defines the API, domain behavior, concurrency tests, demo result, frontend evidence, documentation, security baseline, and verification commands. The host has Go 1.26.2, Docker 29.6.1, Node 24.15.0, and npm 11.14.1; GNU Make and k6 are not installed, so Windows-friendly PowerShell entry points are required alongside Make targets.

## Constraints

- **Stack**: Go, PostgreSQL, React, Vite, and TypeScript — explicitly requested portfolio stack.
- **Correctness**: PostgreSQL is the final concurrency boundary — in-memory locking cannot be the source of truth.
- **Cost**: Only free and open-source software; no trial or paid service dependency.
- **Portability**: Docker Compose is the primary runtime; Windows PowerShell must be practical.
- **Security**: Synthetic data only, no committed secrets, bounded inputs and load generation.
- **Architecture**: Modular monolith — no microservices, Kubernetes, Kafka, or Redis for MVP.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Use PostgreSQL conditional updates and constraints for inventory | Correct under concurrent processes and instances | Validated by integration and k6 tests |
| Keep one Go module with focused internal packages | Smallest architecture that preserves domain boundaries | Validated by full Go test and race suites |
| Use deterministic static evidence only when the API is unreachable | Keeps the portfolio legible without misrepresenting live state | Validated in the reviewer UI |
| Run lifecycle sequentially | GSD agents are not installed and automatic subagent dispatch is unavailable | Completed with recorded verification evidence |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition**:
1. Move verified requirements to Validated.
2. Record scope changes and key decisions.
3. Update the project description if implementation reality changes.

**After each milestone**:
1. Re-check the core value and constraints.
2. Audit out-of-scope boundaries.
3. Record current test and demo evidence.

---
*Last updated: 2026-07-13 after initialization*
