param(
    [Parameter(Position = 0)]
    [ValidateSet('setup', 'migrate', 'seed', 'api', 'worker', 'frontend', 'test', 'test-integration', 'load-test', 'demo', 'verify', 'clean')]
    [string]$Task = 'verify'
)

$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
if (-not $env:DATABASE_URL) { $env:DATABASE_URL = 'postgres://stockrush:stockrush@localhost:5432/stockrush?sslmode=disable' }

function Assert-NativeSuccess([string]$Name) {
    if ($LASTEXITCODE -ne 0) { throw "$Name failed with exit code $LASTEXITCODE" }
}

switch ($Task) {
    'setup' { docker compose up -d postgres }
    'migrate' { go run ./cmd/migrate }
    'seed' { go run ./cmd/loadgen -mode reset }
    'api' { go run ./cmd/api }
    'worker' { go run ./cmd/worker }
    'frontend' { npm --prefix web run dev }
    'test' { go test ./... }
    'test-integration' { go test -tags=integration -p 1 ./... }
    'load-test' { docker run --rm --network stockrush-go_default -e API_BASE_URL=http://api:8080 -v "${Root}/loadtest:/scripts:ro" grafana/k6:latest run /scripts/flash-sale.js }
    'demo' { docker compose up -d --build; go run ./cmd/loadgen -mode demo -attempts 1000 }
    'verify' {
        $files = Get-ChildItem -Path cmd,internal -Recurse -Filter '*.go' | Select-Object -ExpandProperty FullName
        $unformatted = gofmt -l $files
        if ($unformatted) { throw "Unformatted Go files:`n$($unformatted -join "`n")" }
        go vet ./...
        Assert-NativeSuccess 'go vet'
        go test ./...
        Assert-NativeSuccess 'Go unit tests'
        docker run --rm -v "${Root}:/src" -w /src golang:1.26.5 go test -race ./internal/...
        Assert-NativeSuccess 'Go race tests'
        docker compose up -d --wait postgres
        Assert-NativeSuccess 'PostgreSQL startup'
        $env:DATABASE_URL = 'postgres://stockrush:stockrush@localhost:5432/stockrush?sslmode=disable'
        $env:TEST_DATABASE_URL = $env:DATABASE_URL
        go run ./cmd/migrate
        Assert-NativeSuccess 'Migration validation'
        go test -tags=integration -p 1 ./...
        Assert-NativeSuccess 'PostgreSQL integration tests'
        docker build --target build -t stockrush-web-verify ./web
        Assert-NativeSuccess 'Frontend lint, tests, and build'
        docker compose config --quiet
        Assert-NativeSuccess 'Docker Compose validation'
        go run ./cmd/guardrails
        Assert-NativeSuccess 'Repository guardrails'
        git diff --check
        Assert-NativeSuccess 'Git whitespace check'
    }
    'clean' { docker compose down -v --remove-orphans; go clean -testcache }
}
