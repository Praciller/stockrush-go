# API

The API is versioned under `/api/v1`. Successful responses use `{ "data": ..., "meta": { "requestId": "..." } }`. Errors use `{ "error": { "code": "...", "message": "...", "requestId": "..." } }`.

The live OpenAPI document is served at `/openapi.yaml`.

## Endpoints

- Products: `GET/POST /products`, `GET/PATCH /products/{id}`
- Sales: `GET/POST /sales`, `GET /sales/{id}`, `POST /sales/{id}/activate|end`
- Reservations: `POST /sales/{id}/reservations`, `GET /reservations/{id}`, `POST /reservations/{id}/confirm|cancel`
- Payments: `POST /payments/simulate`
- Orders: `GET /orders`, `GET /orders/{id}`
- Demo: `GET /demo/status|report`, `POST /demo/reset|load-test`
- Operations: `/health/live`, `/health/ready`, `/metrics`

Reservation and payment writes require `Idempotency-Key`. Demo mutations require `X-Demo-Token` and work only when `APP_ENV=development`.
