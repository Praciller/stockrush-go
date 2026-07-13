# Security

- Request bodies are capped at 1 MiB and JSON rejects unknown fields.
- HTTP server read, write, header, idle, and shutdown timeouts are explicit.
- Request IDs correlate safe structured logs and responses.
- CORS uses an allowlist from configuration.
- Security headers prevent sniffing, framing, and permissive referrers.
- SQL is parameterized; no request value is concatenated into SQL.
- Demo load values are restricted to 10, 100, 500, or 1,000.
- Per-IP token buckets use forwarded addresses only from configured trusted proxy CIDRs.
- Public reservations fix product, sale, quantity, and synthetic identity server-side and use a PostgreSQL-backed global budget.
- Production operator mutations require a server-side API key whose SHA-256 hash is compared in constant time.
- Development reset/load controls return 404 in production; public mutation has an emergency disable switch.
- `.env`, generated JSON, dependencies, and build outputs are ignored.
- Repository guardrails scan for secret markers and oversized files.

The optional local demo token is not production authentication and is empty by default. It is never built into public frontend assets.
