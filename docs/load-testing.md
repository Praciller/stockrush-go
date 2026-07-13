# Load testing

`loadtest/flash-sale.js` runs with k6 in Docker, so a host k6 install is optional.

```bash
make load-test-small
make load-test
```

Variables:

| Variable | Default | Purpose |
|---|---|---|
| `API_BASE_URL` | `http://host.docker.internal:8080` | API origin |
| `SALE_ID` | fetched from demo status | Target sale |
| `VUS` | `100` | Virtual users |
| `DURATION` | `10s` | Test duration |
| `IDEMPOTENCY_STRATEGY` | `unique` | `unique`, `user`, or `shared` |

For the deterministic portfolio proof, use `make load-test-demo`. The Go load generator uses exactly 1,000 synchronized attempts and writes the Markdown report from actual results.
