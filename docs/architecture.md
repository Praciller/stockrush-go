# Architecture

StockRush Go is one deployable Go codebase with separate API, worker, migration, and load-generator commands. Packages are organized by responsibility, but PostgreSQL transactions join operations that must succeed or fail together.

```mermaid
flowchart TB
    Browser["React/Vite reviewer UI"] -->|REST| Router["Go net/http + chi"]
    K6["k6 load scenario"] -->|REST| Router
    Router --> Store["PostgreSQL store"]
    Store --> DB[("PostgreSQL 17")]
    Expirer["Expiration worker"] --> Store
    Demo["Go load generator"] --> Store
    Demo --> Evidence["reports/local_portfolio_report.md"]
```

## Reservation flow

```mermaid
sequenceDiagram
    participant B as Buyer
    participant A as API
    participant P as PostgreSQL
    B->>A: POST reservation + Idempotency-Key
    A->>P: BEGIN
    A->>P: INSERT idempotency key ON CONFLICT DO NOTHING
    alt Original request
        A->>P: Validate sale, product, and user limit
        A->>P: UPDATE inventory WHERE available >= quantity
        A->>P: INSERT reservation, order, and event
        A->>P: COMMIT
        A-->>B: 201 reservation
    else Retry
        A->>P: Read committed original response
        A-->>B: 200 same reservation
    end
```

## Order state machine

```mermaid
stateDiagram-v2
    [*] --> pending
    pending --> confirmed
    pending --> cancelled
    pending --> expired
    confirmed --> paid
    confirmed --> payment_failed
    payment_failed --> paid
    paid --> [*]
    cancelled --> [*]
    expired --> [*]
```

The API never relies on an in-memory mutex for inventory. Multiple processes and worker instances coordinate through PostgreSQL row, advisory, and uniqueness locks.
