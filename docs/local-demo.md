# Local demo

## Docker Compose

```bash
docker compose up --build
```

## Deterministic proof

```bash
make demo
```

PowerShell:

```powershell
.\scripts\tasks.ps1 demo
```

The demo migrates PostgreSQL, verifies 100 concurrent retries resolve to one reservation, expires that reservation and restores one unit, resets inventory to 100, runs 1,000 unique attempts, checks reconciliation, and generates `reports/local_portfolio_report.md`.

The local token is intentionally documented for development. It is not authentication and demo mutation endpoints refuse non-development environments.
