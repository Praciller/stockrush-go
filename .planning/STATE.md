# Project State

## Project Reference

See: `.planning/PROJECT.md` (updated 2026-07-13)

**Core value:** Exactly 100 of 1,000 buyers reserve 100 items with zero overselling and no duplicate orders.  
**Current focus:** Phase 3 — Core Domain

## Progress

| Phase | Status | Evidence |
|-------|--------|----------|
| 1 | Complete | Repository audit and planning artifacts validated |
| 2 | Complete | Config and health tests pass; PostgreSQL migration applied locally |
| 3 | In progress | Core domain and API tracer bullets next |
| 4-10 | Pending | — |

## Environment

- Go 1.26.2
- Docker client/server 29.6.1
- Node 24.15.0
- npm 11.14.1
- GNU Make missing
- k6 missing

## Risks and Mitigations

- Docker-backed integration tests depend on the local daemon; keep unit checks runnable without Docker.
- GNU Make is absent on the host; provide equivalent PowerShell commands.
- k6 is absent on the host; validate the script structurally and run it in Docker.
- PostgreSQL concurrency must be proved by database-backed tests, not an in-memory substitute.

## Next Action

Implement the reservation state machine and PostgreSQL store through public behavior tests.
