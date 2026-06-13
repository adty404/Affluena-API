# Clean Architecture Refactor Design

## Goal

Refactor Affluena-API's backend modules one by one toward Clean Architecture while preserving API routes, request/response JSON, database schema, and business behavior.

## Target Boundaries

Each feature module should expose the same external HTTP behavior while moving responsibilities into clearer layers:

- `domain.go`: entities, value objects, constants, and pure business rules.
- `usecase.go`: application orchestration and repository interfaces owned by the module.
- `repository.go`: PostgreSQL/pgx persistence adapter.
- `handler.go`: Gin HTTP adapter and request binding.

Dependency direction:

- handlers depend on usecase-facing interfaces.
- usecases depend on domain types and repository interfaces.
- repositories implement interfaces and may depend on pgx/sql.
- domain/usecase code must not import Gin, `httpx`, or pgx.

## Migration Strategy

Refactor modules incrementally, committing after each safe checkpoint:

1. `wallet`
2. `category`
3. `budget`
4. `transaction`
5. `quickentry`
6. `debt`
7. `tracker`
8. `recurring`
9. `auth`

Simple CRUD modules go first to establish the pattern. `transaction` moves before modules that depend on transaction orchestration. Auth moves last because token/session behavior is security-sensitive.

## Compatibility Rules

The refactor must not change:

- public route paths or HTTP methods.
- request field names.
- response field names.
- migration files or table shapes.
- balance mutation semantics.
- status transition semantics.
- scheduler behavior.
- auth/token behavior.

The refactor may add internal interfaces, usecase structs, domain files, and characterization tests.

## Verification

Before each module refactor:

- run or add targeted characterization tests for current behavior.
- run the targeted package tests.

After each module refactor:

- run targeted tests for that module and direct dependents.
- run `go test ./...`.

After all modules:

- run `go test ./...`.
- run `go vet ./...`.
- run `go build -o /tmp/affluena-api ./cmd/api`.
- rebuild Docker and run API smoke tests covering auth, wallet/category, transactions, budget, quick entry, debt, tracker, and recurring flows.
