# Observability

Production logs use JSON and include request ID, method, path, status, and duration. Authorization headers, database URLs, idempotency keys, user IDs, and IP addresses are not logged. Metrics avoid identifiers as labels.

## Signals

- HTTP rate/error/latency: `stockrush_http_requests_total`, `stockrush_http_request_duration_seconds`
- Reservations: attempts, successes, sold-out, duplicate replay
- Abuse: `stockrush_rate_limited_requests_total`
- Worker: expired, restored, and worker-error counters when co-located with the API
- Readiness: `/health/ready`
- Correctness: `go run ./cmd/invariant-check`

`/metrics` requires the production admin bearer key. The standalone local worker logs its counters; the free public deployment co-locates the worker so its metrics share the API registry.

## Manual Queries

```promql
sum(rate(stockrush_http_requests_total[5m]))
sum(rate(stockrush_http_requests_total{status=~"5.."}[5m]))
histogram_quantile(0.95, sum by (le, route) (rate(stockrush_http_request_duration_seconds_bucket[5m])))
rate(stockrush_rate_limited_requests_total[5m])
rate(stockrush_worker_processing_errors_total[5m])
```

```sql
SELECT state, count(*) FROM reservations GROUP BY state;
SELECT count(*) FROM reservations WHERE state='pending' AND expires_at < now();
SELECT state, wait_event_type, wait_event, count(*) FROM pg_stat_activity GROUP BY 1,2,3;
SELECT max(version) FROM schema_migrations;
```

Investigate any sustained 5xx rate, readiness failure, worker errors, overdue pending reservations, rate-limit spike, or invariant-check failure. No paid monitoring provider is required; these are manual portfolio-demo diagnostics.
