# Failure Recovery Report

Run: 2026-07-13, isolated local Docker Compose environment.

| Failure | Expected | Observed | Result |
|---|---|---|---|
| Stop PostgreSQL | Liveness 200, readiness 503 | 200 / 503 | PASS |
| Restart PostgreSQL | Pool reconnects and readiness recovers | Recovered within bounded polling window | PASS |
| Stop worker with pending reservation | No expiration while stopped | Reservation remained pending | PASS |
| Restart worker after expiry | Restore once | State expired, available returned to 100, one expiration event | PASS |
| Repeat worker cycles | No duplicate restoration | One `reservation.expired` event; inventory remained reconciled | PASS |
| SIGTERM API | Graceful exit | Container exit code 0 | PASS |
| Restart API | Readiness recovers | Ready within bounded polling window | PASS |

Concurrent reservations, duplicate callbacks, cancellation/expiration races, competing workers, and sale closure remain covered by PostgreSQL integration tests. No failure was injected into shared or public infrastructure.
