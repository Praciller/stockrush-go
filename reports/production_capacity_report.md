# Production Capacity Report

Run: 2026-07-13 on local Docker Desktop. These results do not predict free-cloud performance.

## Correctness

The authoritative 1,000-attempt test admits exactly 100 reservations for 100 stock with zero overselling and zero duplicate orders.

## Capacity Profiles

| Concurrency | Throughput attempts/s | p50 ms | p95 ms | p99 ms | Unexpected error rate | Success | Sold out |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 10 | 8.47 | 48.800 | 80.551 | 80.551 | 0% | 10 | 0 |
| 25 | 43.83 | 100.196 | 156.825 | 162.629 | 0% | 25 | 0 |
| 50 | 57.68 | 181.682 | 294.700 | 305.213 | 0% | 50 | 0 |
| 100 | 100.51 | 287.639 | 517.059 | 539.182 | 0% | 100 | 0 |
| 200 | 152.47 | 536.192 | 681.436 | 693.995 | 0% | 100 | 100 |
| 500 | 353.03 | 685.617 | 906.796 | 927.850 | 0% | 100 | 400 |

Throughput includes Go command startup and demo reset overhead; latency comes from the synchronized reservation operations.

## Ten-Minute Soak

- Virtual users: 2
- Request pacing: one iteration per user per second
- Completed iterations: 1,194
- HTTP requests: 1,195
- HTTP failure rate: 0.00%
- p50: 5.05 ms
- p95: 7.48 ms
- Maximum: 19.62 ms
- Checks: 1,195/1,195 passed
- Data received/sent: 720 kB / 311 kB
- Post-soak invariant check: PASS

Post-soak snapshot: API 8.762 MiB, worker 5.434 MiB, PostgreSQL 33.26 MiB; CPU was 0.00%, 0.25%, and 2.56% respectively. Inventory was 20 available, 80 reserved, 0 sold, with 500 historical reservations; expiration recycling explains cumulative successes above the simultaneous allocation without overselling.

Public hosts must not receive this correctness, capacity, or soak load.
