# Security

- Request bodies are capped at 1 MiB and JSON rejects unknown fields.
- HTTP server read, write, header, idle, and shutdown timeouts are explicit.
- Request IDs correlate safe structured logs and responses.
- CORS uses an allowlist from configuration.
- Security headers prevent sniffing, framing, and permissive referrers.
- SQL is parameterized; no request value is concatenated into SQL.
- Demo load values are restricted to 10, 100, 500, or 1,000.
- Per-client token buckets bound abuse in one API process.
- Demo mutations require a local token and `APP_ENV=development`.
- `.env`, generated JSON, dependencies, and build outputs are ignored.
- Repository guardrails scan for secret markers and oversized files.

The local demo token is not production authentication. A live public deployment must disable demo mutations or add real authentication and authorization.
