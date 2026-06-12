# Affluena Core API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first usable Affluena backend slice: auth, wallet, category, transaction, and quick entry APIs.

**Architecture:** Use a Gin HTTP API with pgx PostgreSQL access. Keep handlers thin, repositories explicit, and all balance-changing writes inside database transactions.

**Tech Stack:** Go 1.26, Gin, pgxpool, PostgreSQL, bcrypt, golang-jwt, Docker Compose, SQL migrations.

---

### Task 1: Project Scaffold

**Files:**
- Create: `go.mod`
- Create: `cmd/api/main.go`
- Create: `internal/config/config.go`
- Create: `internal/http/router.go`
- Create: `.env.example`
- Create: `docker-compose.yml`
- Create: `Dockerfile`

- [ ] Initialize module `affluena`.
- [ ] Add runtime config from environment.
- [ ] Add Gin router with `/healthz`.
- [ ] Add Docker setup for API and PostgreSQL.
- [ ] Run `go test ./...`.

### Task 2: Database Schema

**Files:**
- Create: `migrations/000001_init.sql`
- Create: `internal/db/db.go`

- [ ] Create SQL schema for users, refresh tokens, wallets, categories, transactions, quick entry templates.
- [ ] Add database pool setup.
- [ ] Run `go test ./...`.

### Task 3: Auth

**Files:**
- Create: `internal/auth/service.go`
- Create: `internal/auth/repository.go`
- Create: `internal/auth/tokens.go`
- Create: `internal/auth/handler.go`
- Create: `internal/auth/middleware.go`
- Create: `internal/auth/service_test.go`

- [ ] Write failing tests for password hashing and login failure behavior.
- [ ] Implement register, login, refresh, and me endpoints.
- [ ] Persist hashed refresh tokens.
- [ ] Protect private routes with JWT middleware.
- [ ] Run `go test ./...`.

### Task 4: Wallets And Categories

**Files:**
- Create: `internal/wallet/handler.go`
- Create: `internal/wallet/repository.go`
- Create: `internal/category/handler.go`
- Create: `internal/category/repository.go`

- [ ] Implement user-scoped CRUD wallets.
- [ ] Implement user-scoped CRUD categories.
- [ ] Validate enum values at request boundary.
- [ ] Run `go test ./...`.

### Task 5: Transactions

**Files:**
- Create: `internal/transaction/service.go`
- Create: `internal/transaction/repository.go`
- Create: `internal/transaction/handler.go`
- Create: `internal/transaction/service_test.go`

- [ ] Write failing tests for income, expense, transfer, adjustment, update reversal, and delete reversal balance behavior.
- [ ] Implement create/list/detail/update/delete transactions.
- [ ] Apply wallet balance deltas atomically in PostgreSQL transactions.
- [ ] Run `go test ./...`.

### Task 6: Quick Entry

**Files:**
- Create: `internal/quickentry/handler.go`
- Create: `internal/quickentry/repository.go`

- [ ] Implement user-scoped CRUD quick entry templates.
- [ ] Implement execute endpoint that creates a transaction from a template.
- [ ] Reuse transaction service for balance-changing behavior.
- [ ] Run `go test ./...`.

### Task 7: Verification

**Files:**
- Modify: `README.md`

- [ ] Document setup, migrations, and endpoint examples.
- [ ] Run `gofmt`.
- [ ] Run `go test ./...`.
- [ ] Run `go build ./cmd/api`.
