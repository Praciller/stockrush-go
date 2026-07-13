param(
    [Parameter(Position = 0)]
    [ValidateSet('setup', 'migrate', 'seed', 'api', 'worker', 'frontend', 'test', 'test-integration', 'load-test', 'demo', 'verify', 'clean')]
    [string]$Task = 'verify'
)

$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

switch ($Task) {
    'setup' { docker compose up -d postgres }
    'migrate' { go run ./cmd/migrate }
    'seed' { go run ./cmd/loadgen -mode reset }
    'api' { go run ./cmd/api }
    'worker' { go run ./cmd/worker }
    'frontend' { npm --prefix web run dev }
    'test' { go test ./... }
    'test-integration' { go test -tags=integration ./... }
    'load-test' { docker run --rm --network host -i grafana/k6:latest run - < loadtest/flash-sale.js }
    'demo' { docker compose up -d --build; go run ./cmd/loadgen -mode demo -attempts 1000 }
    'verify' {
        $files = Get-ChildItem -Path cmd,internal -Recurse -Filter '*.go' | Select-Object -ExpandProperty FullName
        gofmt -w $files
        go vet ./...
        go test ./...
        go test -race ./internal/...
        npm --prefix web ci
        npm --prefix web run test
        npm --prefix web run build
        docker compose config --quiet
        go run ./cmd/guardrails
    }
    'clean' { docker compose down -v --remove-orphans; go clean -testcache }
}
