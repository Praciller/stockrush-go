# StockRush Go Roadmap

## Phase 1: Repository Audit and Plan
**Goal:** Establish the greenfield execution contract before code changes.  
**Mode:** mvp

**Success Criteria:**
1. Every existing file and tool dependency is inventoried.
2. Project context, testable requirements, roadmap, and state are recorded.
3. Existing work is preserved and risks are visible.

## Phase 2: Foundation
**Goal:** Boot a secure Go API and PostgreSQL schema locally.  
**Mode:** mvp

**Requirements:** FND-01..04

**Success Criteria:**
1. Migrations apply to PostgreSQL.
2. `/health/live` and `/health/ready` behave correctly.
3. Docker Compose configuration validates.

## Phase 3: Core Domain
**Goal:** Expose products, inventory, sales, reservations, orders, and explicit transitions.  
**Mode:** mvp

**Requirements:** CAT-01..02, SALE-01..02

**Success Criteria:**
1. Product and sale APIs persist validated data.
2. State transitions are explicit and auditable.
3. Money uses integer minor units.

## Phase 4: Correctness
**Goal:** Make overselling and duplicate order creation impossible at the database boundary.  
**Mode:** mvp

**Requirements:** RES-01..04, IDEM-01..03, EXP-01..02

**Success Criteria:**
1. 1,000 unique attempts against 100 units yield exactly 100 successes.
2. Concurrent duplicates return one original result without duplicate orders.
3. Competing workers restore expired stock exactly once.

## Phase 5: Demo and API
**Goal:** Complete the reviewer-facing API, synthetic payment, metrics, seed, and demo controls.  
**Mode:** mvp

**Requirements:** API-01..02, OPS-01..04, PAY-01..02, DEMO-01

**Success Criteria:**
1. Demo reset/status/load/report endpoints are bounded and useful.
2. Payment callbacks are idempotent.
3. OpenAPI, metrics, request IDs, and safe middleware are present.

## Phase 6: Frontend
**Goal:** Make the concurrency proof understandable live or offline.  
**Mode:** mvp

**Requirements:** UI-01..03

**Success Criteria:**
1. Reviewer can inspect the sale and run only bounded simulations.
2. Inventory and invariant results are visually clear.
3. API failure switches to honestly labelled static evidence.

## Phase 7: Testing
**Goal:** Leave repeatable automated proof across domain, database, API, and frontend.  
**Mode:** mvp

**Requirements:** TEST-01..03

**Success Criteria:**
1. Unit, race, integration, and concurrency checks pass repeatedly.
2. k6 has safe configurable scenarios.
3. Tests avoid flaky sleep-based coordination.

## Phase 8: Evidence
**Goal:** Generate portfolio evidence and explain the implementation to a reviewer.  
**Mode:** mvp

**Requirements:** DEMO-02..03, DOC-01

**Success Criteria:**
1. The local report is generated from actual run output.
2. Architecture and correctness documentation contains the required Mermaid diagrams.
3. The ten-minute portfolio review path identifies the critical files.

## Phase 9: CI and Deployment Documentation
**Goal:** Automate verification and document provider-neutral deployment modes.  
**Mode:** mvp

**Requirements:** CI-01, SEC-01

**Success Criteria:**
1. CI covers Go, PostgreSQL, frontend, migrations, and Docker.
2. Local, static, and optional live deployment paths are documented.
3. Repository guardrails pass without secrets or real data.

## Phase 10: Final Verification
**Goal:** Run the real finish-line commands and fix every in-scope failure.  
**Mode:** mvp

**Requirements:** All v1 requirements

**Success Criteria:**
1. `make verify` or its PowerShell equivalent passes.
2. `make demo` or its PowerShell equivalent produces the requested invariant report.
3. Remaining limitations are explicit and do not contradict the core proof.
