.PHONY: setup migrate seed api worker frontend test test-integration load-test load-test-small load-test-demo demo verify clean

DATABASE_URL ?= postgres://stockrush:stockrush@localhost:5432/stockrush?sslmode=disable
export DATABASE_URL

setup:
	docker compose up -d postgres

migrate:
	go run ./cmd/migrate

seed:
	go run ./cmd/loadgen -mode reset

api:
	go run ./cmd/api

worker:
	go run ./cmd/worker

frontend:
	npm --prefix web run dev

test:
	go test ./...

test-integration:
	go test -tags=integration -p 1 ./...

load-test:
	docker run --rm --network stockrush-go_default -e API_BASE_URL=http://api:8080 -v "$(CURDIR)/loadtest:/scripts:ro" grafana/k6:latest run /scripts/flash-sale.js

load-test-small:
	docker run --rm --network stockrush-go_default -e API_BASE_URL=http://api:8080 -e VUS=10 -e DURATION=5s -v "$(CURDIR)/loadtest:/scripts:ro" grafana/k6:latest run /scripts/flash-sale.js

load-test-demo:
	go run ./cmd/loadgen -mode demo -attempts 1000

demo:
	powershell -ExecutionPolicy Bypass -File scripts/tasks.ps1 demo

verify:
	powershell -ExecutionPolicy Bypass -File scripts/tasks.ps1 verify

clean:
	docker compose down -v --remove-orphans
	go clean -testcache
