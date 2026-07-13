# StockRush Go Agent Guide

- Treat `.planning/PROJECT.md`, `.planning/REQUIREMENTS.md`, and `.planning/ROADMAP.md` as the execution contract.
- Keep the system a Go modular monolith backed by PostgreSQL; never replace database correctness with an in-memory mutex.
- Use vertical TDD slices for correctness-critical behavior and run the smallest relevant check after each change.
- Preserve synthetic-data-only, local-first, no-secret, no-paid-service operation.
- Prefer the standard library and existing dependencies; do not add Redis, Kafka, Kubernetes, or speculative abstractions.
- Keep Windows PowerShell equivalents for Make targets.
- Claim completion only with command output or an explicit limitation.
