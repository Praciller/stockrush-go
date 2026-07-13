# Deployment

## Mode 1: Fully local

Docker Compose runs PostgreSQL, migrations, API, worker, and static frontend. This is the primary mode and the source of truth.

## Mode 2: Static portfolio

The `Pages` workflow tests and builds only `web/`, publishes `web/dist/` under `/stockrush-go/`, and deploys it through GitHub Pages. `VITE_STATIC_DEMO=true` disables API discovery so the page immediately and honestly labels its bundled deterministic evidence.

Equivalent local build:

```powershell
$env:VITE_STATIC_DEMO='true'
npm --prefix web run build -- --base=/stockrush-go/
```

## Mode 3: Optional live demo

Choose any static host, container host, and PostgreSQL provider that support the required free/open setup. Supply configuration through environment variables, disable or protect demo mutations, and run migrations before API and worker startup.

> Free-plan limits and availability may change. The local Docker Compose demo remains the authoritative deployment path.

No required path needs a cloud account, payment card, trial, Kubernetes, Kafka, or Redis.
