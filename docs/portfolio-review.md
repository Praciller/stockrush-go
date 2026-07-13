# Ten-minute portfolio review

1. Read the headline and verified result in `README.md`.
2. Open <http://localhost:5173> after `docker compose up --build`.
3. Run `make demo` or `.\scripts\tasks.ps1 demo`.
4. Inspect `reports/local_portfolio_report.md`.
5. Read `internal/store/store_integration_test.go` for the synchronized 1,000-attempt proof.
6. Read `internal/store/reservation.go` for the conditional inventory update and idempotency transaction.
7. Read `db/migrations/001_init.sql` for constraints and uniqueness.
8. Read `internal/store/expiration.go` for `FOR UPDATE SKIP LOCKED`.
9. Open `/metrics` and `/openapi.yaml`.
10. Run `make verify` or `.\scripts\tasks.ps1 verify`.

The fastest evidence chain is: test inputs, transaction SQL, database constraints, generated report, and reviewer UI.
