# Recurring Transactions Implementation Plan

> **Current status (2026-06-20):** Historical implementation plan. Recurring rules, manual execution, scheduler idempotency, and shared-wallet recurring support are implemented. Use `../../API_CONTRACT.md`, `../../system_map.md`, and `internal/recurring` for current context.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add recurring transaction rules, manual execution, and a native Go scheduler that creates transactions automatically without duplicate charges.

**Architecture:** Store user-scoped recurring rules and execution history in PostgreSQL. Execute each occurrence inside one database transaction by locking the rule, inserting a unique run record for `scheduled_for`, creating a transaction via `transaction.Repository.CreateInTx`, and advancing `next_run_at`.

**Tech Stack:** Go 1.26, Gin, pgx, PostgreSQL, native `time.Ticker`.

---

### Task 1: Domain Rules

**Files:**
- Create: `internal/recurring/service.go`
- Create: `internal/recurring/service_test.go`

- [ ] Test weekly and monthly next-run advancement.
- [ ] Test catch-up advancement when a rule is far overdue.
- [ ] Test frequency/status validation.

### Task 2: Schema And Repository

**Files:**
- Create: `migrations/000003_recurring_transactions.sql`
- Create: `internal/recurring/repository.go`

- [ ] Add `recurring_transaction_rules`.
- [ ] Add `recurring_transaction_runs` with unique `(rule_id, scheduled_for)`.
- [ ] Implement CRUD rules.
- [ ] Implement manual and scheduled execution with row locks and idempotency.

### Task 3: API And Scheduler

**Files:**
- Create: `internal/recurring/handler.go`
- Create: `internal/recurring/scheduler.go`
- Modify: `internal/server/router.go`
- Modify: `cmd/api/main.go`
- Modify: `internal/config/config.go`

- [ ] Add protected recurring routes.
- [ ] Add background scheduler controlled by env config.
- [ ] Keep scheduler graceful under app shutdown.

### Task 4: Verification

**Files:**
- Modify: `README.md`

- [ ] Document endpoints and idempotency.
- [ ] Run `go test ./...`.
- [ ] Run `go build -o /tmp/affluena-api ./cmd/api`.
- [ ] Rebuild Docker and smoke test create/run/due scheduler behavior.
