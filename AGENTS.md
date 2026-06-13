# Affluena-API Agent Handoff

This file gives future AI coding agents the project context and the expected workflow for continuing Affluena-API. Keep it current when the workflow, architecture, API surface, or verification gate changes.

## Project Context

Affluena-API is an API-first personal finance backend written in Go. It uses Gin, PostgreSQL via pgx, native JWT auth, Docker Compose, and a native Go recurring scheduler. There is no frontend in this repository.

Core domains:

- Auth: register, login, refresh token, protected routes.
- Dashboard: monthly summary for net worth, cashflow, budgets, and upcoming obligations.
- Wallets: cash, bank, e-wallet, investment/trading wallets.
- Categories: user-owned income/expense categories. List supports `GET /api/v1/categories?type=income|expense`.
- Transactions: income, expense, transfer, adjustment, with wallet balance updates and list filters.
- Quick entry templates: one-click transaction templates.
- Category budgets: monthly expense category budget summaries.
- Installments: finite payment tracking.
- Subscriptions: recurring subscription tracking; includes optional `account_detail` to distinguish multiple accounts for the same service.
- Recurring transactions: native scheduled transaction generation.
- Debts: payable/receivable tracking with payment lifecycle.

The PRD is in `affluena-api-lean-prd.md`. The API overview and examples are in `README.md`. The runnable Postman collection is `postman/Affluena-API.postman_collection.json`.

## Repository Shape

- `cmd/api`: API entrypoint.
- `internal/server`: router composition and cross-domain integration tests.
- `internal/<domain>/domain.go`: domain structs and constants.
- `internal/<domain>/handler.go`: Gin HTTP handlers and request binding.
- `internal/<domain>/usecase.go`: application/use-case boundary.
- `internal/<domain>/repository.go`: PostgreSQL persistence.
- `internal/tracker`: installment and subscription tracker implementation.
- `internal/db`: database connection and migrations.
- `migrations`: ordered SQL migrations.
- `scripts/verify.sh`: full verification gate.
- `Makefile`: `make verify` wrapper.

The codebase is not a strict textbook Clean Architecture project, but it follows a layered style: handlers call use cases, use cases depend on repository ports, and repositories isolate SQL.

## Working Rules

Follow existing patterns before inventing new abstractions. Keep changes scoped to the requested domain and avoid unrelated refactors.

Important invariants:

- Every user-owned resource must be scoped by `user_id`.
- Users must not see or mutate another user's wallets, categories, transactions, budgets, debts, installments, subscriptions, quick entries, or recurring rules.
- Money is stored as integer minor units, usually `amount_minor`.
- Operations that change balances or multiple related tables must be atomic PostgreSQL transactions.
- Create/update/delete transaction flows must preserve wallet balances.
- Debt, installment, subscription, quick entry, and recurring execution flows must not partially write data.
- Category CRUD endpoints are generic. Do not create separate income/expense CRUD routes; use list filtering where needed.
- User-facing list endpoints return `{collection, pagination}` and support `limit`, `offset`, and whitelisted `sort` values. Keep pagination metadata scoped to the authenticated user and matching active filters.

## Required Workflow For Every Change

The expected loop for code changes is:

1. Understand the current implementation with `rg`, file reads, and existing tests.
2. Implement the smallest scoped change that satisfies the requirement.
3. Add comprehensive unit/integration tests to cover all cases that a human or user might perform, including both **red flows** (edge cases, invalid inputs, unauthorized access) and **green flows** (successful operations).
4. Audit and update every related documentation artifact whenever code behavior changes: `README.md`, other relevant `.md` files, and the Postman collection.
5. Run targeted tests and the full verification gate while developing.
6. Run the full gate before commit:

   ```bash
   make verify
   ```

7. Review `git status --short` and the diff. Do not stage unrelated files such as `.DS_Store`.
8. Commit with a concise Conventional Commit message.
9. Push to GitHub.

The user expects Docker rebuild, full tests, Postman update, related `.md`/documentation updates, commit, and push after each meaningful project change unless they explicitly say otherwise.

## Verification Gate

Use `make verify` as the authoritative pre-commit/pre-push gate. It runs:

- Go formatting check.
- `go test ./... -count=1`.
- `go vet ./...`.
- `go build -o /tmp/affluena-api ./cmd/api`.
- Postman collection JSON validation.
- `docker compose up -d --build`.
- Integration tests against PostgreSQL:

  ```bash
  AFFLUENA_API_TEST_DATABASE_URL=postgres://affluena_api:affluena_api@localhost:5432/affluena_api?sslmode=disable \
    go test ./internal/db ./internal/debt ./internal/server -count=1
  ```

- `/healthz` check against `http://localhost:8080/healthz`.

Docker must be running for the full gate. The local database defaults to:

```text
postgres://affluena_api:affluena_api@localhost:5432/affluena_api?sslmode=disable
```

For targeted integration tests, use the same `AFFLUENA_API_TEST_DATABASE_URL`.

## Test Expectations

Use focused unit tests for domain/usecase logic and integration tests for API behavior, data isolation, migrations, balance changes, and multi-table workflows.

Existing high-value integration tests:

- `internal/server/isolation_integration_test.go`: proves users cannot access each other's data.
- `internal/server/financial_flow_integration_test.go`: proves transaction lifecycle preserves balances.
- `internal/server/category_filter_integration_test.go`: proves category type filtering.
- `internal/server/transaction_filter_integration_test.go`: proves transaction list filters.
- `internal/server/pagination_integration_test.go`: proves list endpoint pagination metadata and wallet sorting.
- `internal/server/dashboard_summary_integration_test.go`: proves dashboard summary aggregation and isolation.
- `internal/server/subscription_account_detail_integration_test.go`: proves subscription `account_detail` lifecycle.
- `internal/db/migration_integration_test.go`: proves database constraints and migrations.
- `internal/debt/repository_integration_test.go`: proves debt repository lifecycle.

When adding a new API behavior, prefer an HTTP integration test if the behavior is visible from Postman/client usage.

## Documentation, API, And Postman Rules

If an endpoint, payload, response field, or example changes:

- Update `README.md`.
- Update any other related `.md` files, including this `AGENTS.md` when workflow, architecture, or handoff context changes.
- Update `postman/Affluena-API.postman_collection.json`.
- Update examples, notes, and handoff context so they match the implemented behavior.
- Keep Postman JSON valid. `make verify` validates it, but `jq empty postman/Affluena-API.postman_collection.json` is useful for a quick check.
- Keep Postman request names aligned with real API contracts. Avoid labels that imply nonexistent backend routes.

Current notable API decisions:

- `GET /api/v1/categories` lists all user categories.
- `GET /api/v1/categories?type=income` lists income categories.
- `GET /api/v1/categories?type=expense` lists expense categories.
- `GET/PUT/DELETE /api/v1/categories/:id` remain generic category CRUD.
- `GET /api/v1/transactions` supports optional `type`, `wallet_id`, `category_id`, `from`, and `to` filters.
- List endpoints for wallets, categories, transactions, quick entry templates, category budgets, debts, installments, subscriptions, and recurring transactions support `limit`, `offset`, and `sort`.
- `GET /api/v1/dashboard/summary?month=YYYY-MM` returns monthly summary data scoped to the authenticated user.
- Subscriptions accept and return optional `account_detail`.

## Database Migration Rules

- Add new SQL files in numeric order under `migrations`.
- Prefer backward-compatible migrations for existing data, for example `NOT NULL DEFAULT ''` for new optional text fields.
- When adding user-owned references, enforce user ownership in database constraints where practical, following `000005_user_owned_foreign_keys.sql`.
- After adding migrations, run `make verify` so Docker and integration tests apply them.

## Git Rules

Before staging:

```bash
git status --short
git diff --check
```

Stage only files relevant to the task. Leave unrelated untracked files alone, especially `.DS_Store`.

Commit style:

- Use Conventional Commits, for example `feat(api): add subscription account detail`.
- Include a short body when the change includes a migration, security fix, data ownership rule, or non-obvious reason.
- Push after a successful commit:

  ```bash
  git push
  ```

## Communication Notes

The user usually writes in Indonesian. Keep updates concise, practical, and explicit about what was changed, what was tested, the commit hash, and whether it was pushed.

When finishing work, include:

- Summary of behavior changed.
- Verification commands that passed.
- Commit hash.
- Push status.
- Any leftover worktree noise such as untracked `.DS_Store`.
