package httpserver

import "net/http"

func (s *Server) openapi(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write([]byte(openAPISpec))
}

const openAPISpec = `openapi: 3.1.0
info:
  title: StockRush Go API
  version: 1.0.0
  description: Concurrency-safe flash-sale reservation API.
servers:
  - url: http://localhost:8080
paths:
  /health/live:
    get:
      summary: Process liveness
      responses: {"200": {description: Alive}}
  /health/ready:
    get:
      summary: PostgreSQL readiness
      responses: {"200": {description: Ready}, "503": {description: Database unavailable}}
  /version:
    get:
      summary: Safe build metadata
      responses: {"200": {description: Version, commit, and build time}}
  /api/v1/products:
    get:
      summary: List products
      responses: {"200": {description: Products}}
    post:
      summary: Create product
      responses: {"201": {description: Product created}}
  /api/v1/sales/{id}/reservations:
    post:
      summary: Atomically reserve inventory
      parameters:
        - {in: path, name: id, required: true, schema: {type: string, format: uuid}}
        - {in: header, name: Idempotency-Key, required: true, schema: {type: string}}
      responses:
        "201": {description: Reservation created}
        "200": {description: Original result replayed}
        "409": {description: Sold out, inactive sale, limit, or idempotency conflict}
        "429": {description: Rate limited}
  /api/v1/payments/simulate:
    post:
      summary: Simulate an idempotent payment callback
      responses: {"201": {description: Payment recorded}, "200": {description: Original result replayed}}
  /api/v1/public-demo/reservations:
    post:
      summary: Create one bounded anonymous synthetic reservation
      parameters:
        - {in: header, name: Idempotency-Key, required: true, schema: {type: string}}
      responses: {"201": {description: Reservation created}, "200": {description: Original result replayed}, "429": {description: Per-IP or global budget exhausted}, "503": {description: Public mutations disabled}}
  /api/v1/demo/load-test:
    post:
      summary: Run a bounded local demonstration
      responses: {"200": {description: Load report}, "403": {description: Local demo token required}}
  /metrics:
    get:
      summary: Prometheus metrics
      responses: {"200": {description: Metrics}}
`
