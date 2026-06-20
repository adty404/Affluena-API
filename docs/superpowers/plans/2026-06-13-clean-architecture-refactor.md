# Clean Architecture Refactor Implementation Plan

> **Current status (2026-06-20):** Historical implementation plan. The current architecture is the package-per-feature layered style documented in `../../system_map.md` and `../../../AGENTS.md`.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor all Affluena-API backend modules toward Clean Architecture one by one without changing API behavior or business logic.

**Architecture:** Keep the existing package-per-feature layout, but split each module into domain, usecase, repository, and handler responsibilities. Usecases own repository interfaces; repositories remain pgx adapters; handlers remain Gin adapters.

**Tech Stack:** Go 1.26, Gin, pgxpool, PostgreSQL, Docker Compose.

---

### Task 1: Wallet Pilot

**Files:**
- Create: `internal/wallet/domain.go`
- Create: `internal/wallet/usecase.go`
- Modify: `internal/wallet/repository.go`
- Modify: `internal/wallet/handler.go`
- Create: `internal/wallet/usecase_test.go`

- [ ] Add characterization tests for create/list/get/update/delete usecase delegation and validation.
- [ ] Move wallet types into `domain.go`.
- [ ] Add a `UseCase` with a repository interface.
- [ ] Change handler to depend on a usecase interface instead of concrete repository.
- [ ] Run `go test ./internal/wallet`.
- [ ] Run `go test ./...`.

### Task 2: Category Pilot

**Files:**
- Create: `internal/category/domain.go`
- Create: `internal/category/usecase.go`
- Modify: `internal/category/repository.go`
- Modify: `internal/category/handler.go`
- Create: `internal/category/usecase_test.go`

- [ ] Add characterization tests for create/list/get/update/delete usecase delegation and validation.
- [ ] Move category types into `domain.go`.
- [ ] Add a `UseCase` with a repository interface.
- [ ] Change handler to depend on a usecase interface instead of concrete repository.
- [ ] Run `go test ./internal/category`.
- [ ] Run `go test ./...`.

### Task 3: Budget

**Files:**
- Create or update: `internal/budget/domain.go`
- Create or update: `internal/budget/usecase.go`
- Modify: `internal/budget/repository.go`
- Modify: `internal/budget/handler.go`

- [ ] Keep month parsing and usage math in domain/usecase code.
- [ ] Keep repository as persistence adapter only.
- [ ] Run `go test ./internal/budget`.
- [ ] Run `go test ./...`.

### Task 4: Transaction Core

**Files:**
- Create or update: `internal/transaction/domain.go`
- Create or update: `internal/transaction/usecase.go`
- Modify: `internal/transaction/repository.go`
- Modify: `internal/transaction/handler.go`

- [ ] Keep balance delta rules in domain code.
- [ ] Move create/update/delete orchestration into usecase.
- [ ] Keep pgx transaction mechanics in repository adapter methods.
- [ ] Run `go test ./internal/transaction`.
- [ ] Run `go test ./...`.

### Task 5: Dependent Modules

**Files:**
- `internal/quickentry/*`
- `internal/debt/*`
- `internal/tracker/*`
- `internal/recurring/*`

- [ ] Refactor each module using the same domain/usecase/repository/handler boundary.
- [ ] Replace concrete cross-module dependencies with small interfaces where practical.
- [ ] Run targeted tests after each module.
- [ ] Run `go test ./...` after each module.

### Task 6: Auth

**Files:**
- `internal/auth/*`

- [ ] Keep password hashing and token issuing behavior unchanged.
- [ ] Move repository dependency behind an interface owned by the usecase/service layer.
- [ ] Keep JWT middleware as HTTP adapter code.
- [ ] Run `go test ./internal/auth`.
- [ ] Run `go test ./...`.

### Task 7: Final Verification

**Files:**
- All modules.

- [ ] Run `go test ./...`.
- [ ] Run `go vet ./...`.
- [ ] Run `go build -o /tmp/affluena-api ./cmd/api`.
- [ ] Run `docker compose up -d --build`.
- [ ] Run API smoke tests covering major daily flows.
- [ ] Inspect `git diff` for route/schema/API behavior changes.
