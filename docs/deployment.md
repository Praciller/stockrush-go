# Deployment

## Mode 1: Fully local

Docker Compose runs PostgreSQL, migrations, API, worker, and static frontend. This is the primary mode and the source of truth.

## Mode 2: Static portfolio

The `Pages` workflow tests and builds only `web/`, publishes `web/dist/` under `/stockrush-go/`, and deploys it through GitHub Pages. `VITE_STATIC_DEMO=true` disables API discovery so the page immediately and honestly labels its bundled deterministic evidence.

Verified static URL: <https://praciller.github.io/stockrush-go/>

Equivalent local build:

```powershell
$env:VITE_STATIC_DEMO='true'
npm --prefix web run build -- --base=/stockrush-go/
```

## Mode 3: Evaluated live demo

The current candidate is GitHub Pages plus one Hugging Face CPU Basic Docker Space and Neon Free PostgreSQL. The API co-locates the expiration worker because the free compute boundary is one container. Production configuration fails closed, migrations use an advisory lock, and the public reservation is globally bounded in PostgreSQL.

See `hosting-evaluation.md` before deployment. Free-plan limits and terms must be rechecked, and cloud secrets require explicit approval.

> Free-plan limits and availability may change. The local Docker Compose demo remains the authoritative deployment path.

No required path needs a cloud account, payment card, trial, Kubernetes, Kafka, or Redis.
