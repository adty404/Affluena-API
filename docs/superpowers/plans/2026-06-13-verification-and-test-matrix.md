# Verification and Test Matrix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expand Affluena-API's test suite around financial invariants, user isolation, and cross-module data integrity, then add a single `make verify` gate for pre-commit and pre-push checks.

**Architecture:** Add focused unit tests for pure domain/usecase behavior, integration tests for multi-user API and database ownership boundaries, and a shell verification script that composes formatting, unit tests, optional integration tests, vet, build, Docker, health, and Postman JSON validation. Keep production code changes minimal and only where tests reveal a real gap.

**Tech Stack:** Go testing package, PostgreSQL via Docker Compose, `go test`, `go vet`, `go build`, POSIX shell, Makefile.

---

### Task 1: Unit Test Financial Domain Edge Cases

**Files:**
- Modify: `internal/transaction/service_test.go`
- Modify: `internal/budget/service_test.go`
- Modify: `internal/debt/service_test.go`
- Modify: `internal/tracker/service_test.go`
- Modify: `internal/recurring/service_test.go`
- Modify: `internal/caldate/month_test.go`

- [ ] Add table-driven tests for transaction balance deltas, budget usage, debt payment lifecycle, tracker lifecycle, recurring schedules, and month parsing.
- [ ] Run package tests after each module: `go test ./internal/<module> -count=1`.

### Task 2: Unit Test Usecase Validation and Delegation

**Files:**
- Modify: `internal/auth/service_test.go`
- Modify: `internal/wallet/usecase_test.go`
- Modify: `internal/category/usecase_test.go`
- Modify: `internal/budget/usecase_test.go`
- Modify: `internal/quickentry/usecase_test.go`
- Modify: `internal/recurring/usecase_test.go`
- Modify: `internal/tracker/usecase_test.go`
- Modify: `internal/debt/usecase_test.go`
- Modify: `internal/transaction/usecase_test.go`

- [ ] Add tests for invalid types, missing required values, defaulting behavior, error propagation, and request-to-repository delegation.
- [ ] Run `go test ./internal/... -count=1`.

### Task 3: Integration Test User Isolation Matrix

**Files:**
- Modify: `internal/server/isolation_integration_test.go`
- Modify: `internal/debt/repository_integration_test.go`
- Modify: `internal/db/migration_integration_test.go`

- [ ] Cover two-user isolation for wallet, category, transaction, quick entry, budget, debt, installment, subscription, and recurring endpoints.
- [ ] Cover cross-owner `wallet_id`, `to_wallet_id`, `category_id`, debt, and recurring references.
- [ ] Verify database composite ownership foreign keys exist.
- [ ] Run `AFFLUENA_API_TEST_DATABASE_URL=postgres://affluena_api:affluena_api@localhost:5432/affluena_api?sslmode=disable go test ./internal/db ./internal/debt ./internal/server -count=1`.

### Task 4: Add Verify Gate

**Files:**
- Create: `scripts/verify.sh`
- Create: `Makefile`

- [ ] Add a strict shell script that checks formatting, runs unit tests, runs integration tests when database URL is available, runs vet, builds API, validates Postman JSON, optionally rebuilds Docker, and checks `/healthz`.
- [ ] Add `make verify` as the single command for the script.
- [ ] Run `make verify`.

### Task 5: Final Verification and Git Hygiene

**Files:**
- All modified test, migration, script, and plan files.

- [ ] Run `go test ./... -cover`.
- [ ] Run `make verify`.
- [ ] Rebuild Docker and check health.
- [ ] Review `git diff`.
- [ ] Commit and push only after all gates pass.
