# Database design

All money uses integer minor units. Inventory counters have non-negative check constraints. State values are constrained in SQL, and foreign keys preserve aggregate relationships.

```mermaid
erDiagram
    PRODUCTS ||--|| INVENTORY : has
    PRODUCTS ||--o{ FLASH_SALES : offered_in
    FLASH_SALES ||--o{ RESERVATIONS : receives
    RESERVATIONS ||--|| ORDERS : creates
    ORDERS ||--o{ PAYMENTS : receives
    RESERVATIONS ||--o{ DOMAIN_EVENTS : records
    PRODUCTS {
      uuid id PK
      text sku UK
      bigint price_minor
      char currency
      boolean active
    }
    INVENTORY {
      uuid product_id PK,FK
      int available
      int reserved
      int sold
    }
    FLASH_SALES {
      uuid id PK
      uuid product_id FK
      text state
      int allocated_stock
      int remaining_stock
    }
    RESERVATIONS {
      uuid id PK
      uuid sale_id FK
      text user_id
      int quantity
      text state
      timestamptz expires_at
    }
    ORDERS {
      uuid id PK
      uuid reservation_id UK,FK
      bigint amount_minor
      text state
    }
    PAYMENTS {
      uuid id PK
      uuid order_id FK
      text idempotency_key UK
      text request_fingerprint
    }
    DOMAIN_EVENTS {
      uuid id PK
      uuid aggregate_id
      text event_type
      jsonb metadata
    }
```

`idempotency_keys.key` is unique. `orders.reservation_id` is unique. These constraints turn concurrent duplicate requests into deterministic replay instead of duplicate effects. Reservations atomically decrement both `inventory.available` and `flash_sales.remaining_stock`; cancellation and expiration restore both counters in the same transaction.

Migrations live in `db/migrations/` and run transactionally through `cmd/migrate`.
