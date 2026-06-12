# Debt Loan Manager Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Debt & Loan Manager endpoints for receivables and payables with atomic wallet balance updates.

**Architecture:** Add a new `internal/debt` module following the existing handler/repository/service pattern. Debt creation and repayment reuse `transaction.Repository.CreateInTx` so wallet balance changes and debt state changes commit or roll back together.

**Tech Stack:** Go 1.26, Gin, pgxpool, PostgreSQL migrations, Docker Compose.

---

### Task 1: Domain Tests

**Files:**
- Create: `internal/debt/service_test.go`
- Create: `internal/debt/service.go`

- [ ] Write tests for debt payment status transitions.
- [ ] Write tests for overpayment rejection.
- [ ] Write tests for transaction type mapping by debt type and action.
- [ ] Run `go test ./internal/debt` and verify failure before implementation.
- [ ] Implement service functions and rerun `go test ./internal/debt`.

### Task 2: Schema

**Files:**
- Create: `migrations/000004_debts.sql`

- [ ] Add `debts` table with user, wallet, categories, origination transaction, principal, paid amount, due date, status, note, timestamps.
- [ ] Add `debt_payments` table with user, debt, generated transaction, amount, paid timestamp, note, created timestamp.
- [ ] Add indexes for user status/due date and payment history.

### Task 3: Repository

**Files:**
- Create: `internal/debt/repository.go`

- [ ] Implement create with origination transaction and debt insert in one DB transaction.
- [ ] Implement list/get.
- [ ] Implement metadata update with status validation.
- [ ] Implement soft delete as status `cancelled`.
- [ ] Implement payment with row lock, generated repayment transaction, payment insert, paid amount update, and automatic status transition.

### Task 4: Handler And Routes

**Files:**
- Create: `internal/debt/handler.go`
- Modify: `internal/server/router.go`

- [ ] Bind and validate create, update, and payment request bodies.
- [ ] Add protected debt routes under `/api/v1/debts`.
- [ ] Map domain and repository errors to 400 or 404 responses.

### Task 5: Docs And Verification

**Files:**
- Modify: `README.md`

- [ ] Document debt endpoints and example flows.
- [ ] Run `gofmt -w internal/debt internal/server/router.go`.
- [ ] Run `go test ./...`.
- [ ] Run `go vet ./...`.
- [ ] Run `go build -o /tmp/affluena-api ./cmd/api`.
- [ ] Run Docker smoke test for receivable and payable create/pay flows.
