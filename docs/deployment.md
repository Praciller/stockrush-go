# Deployment

## Mode 1: Fully local

Docker Compose runs PostgreSQL, migrations, API, worker, and static frontend. This is the primary mode and the source of truth.

## Mode 2: Static portfolio

Run `npm --prefix web ci && npm --prefix web run build`, then publish `web/dist/` to any static host. The bundled fallback evidence keeps the project understandable without an API and is clearly labelled pre-generated.

## Mode 3: Optional live demo

Choose any static host, container host, and PostgreSQL provider that support the required free/open setup. Supply configuration through environment variables, disable or protect demo mutations, and run migrations before API and worker startup.

> Free-plan limits and availability may change. The local Docker Compose demo remains the authoritative deployment path.

No required path needs a cloud account, payment card, trial, Kubernetes, Kafka, or Redis.
