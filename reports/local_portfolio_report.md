# StockRush Go Portfolio Evidence

## Environment

- Timestamp: 2026-07-13T06:42:12Z
- Go: go1.26.2
- Storage: PostgreSQL

## Demo Parameters

- Initial inventory: 100
- Concurrent purchase attempts: 1000
- Quantity per request: 1

## Concurrency Test Results

| Result | Count |
|---|---:|
| Successful reservations | 100 |
| Sold out | 900 |
| Duplicate | 0 |
| Rate limited | 0 |
| Failed | 0 |

## Latency Summary

- p50: 1610.58 ms
- p95: 2075.70 ms
- p99: 2108.90 ms

## Reservation Expiration Results

- Expired reservations: 1
- Restored inventory: 1

## Idempotency Results

- Concurrent retries returning one reservation: 100/100
- Duplicate orders: 0

## Database Invariant Checks

- Available inventory is non-negative: true
- Reserved inventory is non-negative: true
- Sold inventory is non-negative: true
- Reservation/order reconciliation: true

## Final Inventory Reconciliation

- Available: 0
- Reserved: 100
- Sold: 0
- Total: 100

## Zero-Oversell Verdict

**Zero overselling: PASS**

## Known Limitations

- The MVP rate limiter is process-local.
- Payment is synthetic.
- The local Docker Compose demo is authoritative.
