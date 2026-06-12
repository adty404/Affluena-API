# Budget Installment Subscription Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add category budgeting plus installment and subscription trackers with manual payment triggers that create expense transactions.

**Architecture:** Extend the existing Gin + pgx API with three new user-scoped modules. Budget summaries read existing expense transactions; installment and subscription payment endpoints reuse transaction balance logic inside the same PostgreSQL transaction as tracker updates.

**Tech Stack:** Go 1.26, Gin, pgxpool, PostgreSQL SQL migrations.

---

### Task 1: Domain Tests

**Files:**
- Create: `internal/budget/service_test.go`
- Create: `internal/tracker/service_test.go`

- [ ] Test budget month parsing and usage math.
- [ ] Test installment payment state transitions.
- [ ] Test subscription due date advancement.
- [ ] Run `go test ./internal/budget ./internal/tracker` and verify failure because functions do not exist.

### Task 2: Schema

**Files:**
- Create: `migrations/000002_budget_installment_subscription.sql`

- [ ] Add `category_budgets`.
- [ ] Add `installments`.
- [ ] Add `subscriptions`.

### Task 3: Budget API

**Files:**
- Create: `internal/budget/service.go`
- Create: `internal/budget/repository.go`
- Create: `internal/budget/handler.go`
- Modify: `internal/server/router.go`

- [ ] Implement budget CRUD.
- [ ] Implement monthly summary endpoint with spent, remaining, and usage percentage.
- [ ] Run `go test ./...`.

### Task 4: Tracker API

**Files:**
- Create: `internal/tracker/service.go`
- Create: `internal/tracker/installment_repository.go`
- Create: `internal/tracker/subscription_repository.go`
- Create: `internal/tracker/handler.go`
- Modify: `internal/transaction/repository.go`
- Modify: `internal/server/router.go`

- [ ] Implement installment CRUD and `POST /installments/:id/pay`.
- [ ] Implement subscription CRUD and `POST /subscriptions/:id/pay`.
- [ ] Ensure payment endpoints create transactions and update tracker state atomically.
- [ ] Run `go test ./...`.

### Task 5: Docs And Smoke

**Files:**
- Modify: `README.md`

- [ ] Document new endpoints and payment behavior.
- [ ] Run `gofmt -w`.
- [ ] Run `go test ./...`.
- [ ] Run `go build -o /tmp/affluena-api ./cmd/api`.
- [ ] Run Docker smoke flow for budget, installment pay, and subscription pay.
