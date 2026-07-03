# Affluena-API Agent Handoff

This file gives future AI coding agents the project context and the expected workflow for continuing Affluena-API. Keep it current when the workflow, architecture, API surface, or verification gate changes.

## Project Context

Affluena-API is an API-first personal finance backend written in Go. It uses Gin, PostgreSQL via pgx, native JWT auth, Docker Compose, and a native Go recurring scheduler. There is no frontend in this repository.

Core domains:

- Auth: register, login, refresh token, protected routes.
- Dashboard: monthly summary for net worth, cashflow, budgets, upcoming obligations, plus cashflow trend, expense distribution, and spend forecasting analytics.
- Wallets: cash, bank, e-wallet, investment/trading wallets. Wallets can be shared with other users; a share carries a `role` of `member` (read+write) or `viewer` (read-only).
- Categories: user-owned income/expense categories with optional same-user, same-type `parent_id` nesting up to 3 levels. List supports `GET /api/v1/categories?type=income|expense`.
- Tags: user-owned labels attachable to transactions through `tag_ids`, with transaction filtering by `tag_id`.
- Transactions: income, expense, transfer, adjustment, with wallet balance updates, tag links, and list filters.
- Quick entry templates: one-click transaction templates; `wallet_id` can be an owned wallet or a joined shared wallet.
- Category budgets: monthly expense category budget summaries.
- Installments: finite payment tracking; `wallet_id` can be an owned wallet or a joined shared wallet.
- Subscriptions: recurring subscription tracking; includes optional `account_detail` to distinguish multiple accounts for the same service; `wallet_id` can be an owned wallet or a joined shared wallet.
- Recurring transactions: native scheduled transaction generation; `wallet_id` and `to_wallet_id` can be owned wallets or joined shared wallets.
- Debts: payable/receivable tracking with payment lifecycle; `wallet_id` can be an owned wallet or a joined shared wallet.
- Financial goals: collaborative saving goals that create goal wallets and support member invitations.

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
- `postman`: runnable API collection that should match the documented API surface.
- `scripts/verify.sh`: full verification gate.
- `Makefile`: `make verify` wrapper.

The codebase is not a strict textbook Clean Architecture project, but it follows a layered style: handlers call use cases, use cases depend on repository ports, and repositories isolate SQL.

## Working Rules

Follow existing patterns before inventing new abstractions. Keep changes scoped to the requested domain and avoid unrelated refactors.

Important invariants:

- Every user-owned resource must be scoped by `user_id`.
- Users must not see or mutate another user's wallets, categories, tags, transactions, budgets, debts, goals, installments, subscriptions, quick entries, or recurring rules.
- API logs must not persist authentication secrets; auth request payloads and auth responses containing access/refresh tokens must be masked.
- Money is stored as integer minor units, usually `amount_minor`.
- Operations that change balances or multiple related tables must be atomic PostgreSQL transactions.
- Create/update/delete transaction flows must preserve wallet balances.
- Transaction balance deltas must only update wallets still accessible to the authenticated user, either as owner or joined shared-wallet member.
- Shared-wallet writes require a `member` role. A `viewer`-role share is read-only: it may see the wallet, its transactions, reports, exports, and dashboard analytics, but every write path (transaction balance apply, quick entry, tracker, debt, recurring) must reject it. `wallet.CheckMemberAccess` enforces `>= AccessMember`, and the copy-pasted owner-or-joined-member SQL predicates additionally require `ws.role = 'member'`.
- A pending (not-yet-joined) invitee must not see a wallet's balance, member roster, or goal-pool total; those are withheld until the invite is accepted.
- Goal wallets use internal type `goal`; generic wallet create/update/delete endpoints must not create, convert, or delete goal-managed wallets.
- Installment plans must keep `total_amount_minor == monthly_amount_minor * tenor_months`; otherwise payments can overcharge or undercharge the declared total.
- Debt, installment, subscription, quick entry, and recurring execution flows must not partially write data.
- Financial goal creation and invite acceptance create related goal wallets atomically; treat those as cross-module workflows and test partial-write risks when touching them.
- Financial goal invitation response is terminal after `joined` or `rejected`; do not let `PUT /api/v1/goals/:id/members/:user_id/respond` turn a joined member into a rejected member because that can orphan goal wallets.
- Category CRUD endpoints are generic. Do not create separate income/expense CRUD routes; use list filtering where needed.
- Category nesting must not exceed 3 levels and must reject cyclic parent relationships, cross-user parents, and parent categories with a different type.
- User-facing list endpoints usually return `{collection, pagination}` and support `limit`, `offset`, and whitelisted `sort` values. Keep pagination metadata scoped to the authenticated user and matching active filters. Financial goals are a current exception: `GET /api/v1/goals` returns a JSON array ordered by creation time.
- Dashboard analytics (e.g. summary, cashflow) evaluate all accessible wallets (owned wallets and joined shared wallets). However, category budgets strictly evaluate against personal budgets and are not shared.
- Debts cannot be canceled/deleted if payments have been made.
- Background jobs and goroutines must use `internal/async.SafeGo` for panic recovery.
- Use `httpx.GetUUIDParam` to parse path UUIDs safely, rather than raw `c.Param`.
- Use `httpx.WriteError` along with `errors.Is(err, ...)` checking sentinel errors (`IsNotFound`, `IsForbidden`, etc.) for robust HTTP error mapping without leaking database errors.
- A centralized `wallet.AccessChecker` helper is available for checking user access to wallets (including shared wallets). It is currently used by the `transaction` module, but migration for other modules (`splitbill`, `debt`, `tracker`, `quickentry`) is intentionally deferred and should be done gradually. Modules not yet migrated are still protected by their existing DB-level checks and integration tests.

## Required Workflow For Every Change

The expected loop for code changes is:

1. Understand the current implementation with `rg`, file reads, and existing tests.
2. Implement the smallest scoped change that satisfies the requirement.
3. Add comprehensive unit/integration tests to cover all cases that a human or user might perform, including both **red flows** (edge cases, invalid inputs, unauthorized access) and **green flows** (successful operations). **Crucially, if the feature interacts with or impacts other modules, the tests MUST cover those cross-module integrations (e.g., wallet balance updates when achieving a goal) to guarantee data integrity across the system.**
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
- `internal/server/category_hierarchy_test.go`: proves category parent nesting, max-depth behavior, and cyclic-reference protection.
- `internal/server/category_reorder_integration_test.go`: proves category icon/color round-trip, position-based default ordering, the reorder endpoint (full and partial), and that reorder rejects non-owned ids without partial writes.
- `internal/server/transaction_filter_integration_test.go`: proves transaction list filters.
- `internal/server/pagination_integration_test.go`: proves list endpoint pagination metadata and wallet sorting.
- `internal/server/dashboard_summary_integration_test.go`: proves dashboard summary aggregation and isolation.
- `internal/server/dashboard_integration_test.go`: proves advanced dashboard analytics/reporting behavior.
- `internal/server/wallet_share_integration_test.go`: proves shared wallet invite lifecycle, member transactions, owner visibility, export, analytics without duplicate counting, and joined member quick entry templates on shared wallets.
- `internal/server/wallet_validation_integration_test.go`: proves public wallet endpoints reject direct goal-wallet creation, conversion, update, and deletion.
- `internal/server/goal_integration_test.go`: proves financial goal creation, membership, and access behavior.
- `internal/server/tag_integration_test.go`: proves tag CRUD and transaction tag integration.
- `internal/server/splitbill_integration_test.go`: proves split bill success behavior and rollback when debt creation fails.
- `internal/server/recurring_integration_test.go`: proves manual recurring execution creates a transaction and updates wallet balance, including joined shared-wallet member rules.
- `internal/server/subscription_account_detail_integration_test.go`: proves subscription `account_detail` lifecycle.
- `internal/server/tracker_shared_wallet_integration_test.go`: proves joined shared-wallet members can create and pay installment/subscription trackers on shared wallets.
- `internal/server/debt_shared_wallet_integration_test.go`: proves joined shared-wallet members can create and pay debt records on shared wallets.
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
- `POST/PUT /api/v1/categories` accept optional `parent_id`; category trees are limited to 3 levels and parents must be owned by the authenticated user with the same category type.
- `POST/PUT /api/v1/categories` also accept optional client-owned `icon` and `color` strings (not validated by the server, same as wallets). `position` is returned but not settable via create/update: creates append at `MAX(position)+1` per user.
- `PUT /api/v1/categories/reorder` accepts `{ "ids": [...] }` and sets `position` = array index per id in one transaction, scoped to the user; ids omitted keep their position, any non-owned id returns 404 and rolls everything back. Category list ordering defaults to `position_asc` (position, then name).
- `GET/PUT/DELETE /api/v1/categories/:id` remain generic category CRUD.
- `POST/GET/PUT/DELETE /api/v1/tags` manage user-owned transaction labels.
- `POST/PUT /api/v1/transactions` accept `tag_ids`.
- `GET /api/v1/transactions` supports optional `type`, `wallet_id`, `category_id`, `tag_id`, `from`, and `to` filters.
- `POST/PUT /api/v1/transactions` accept `transaction_at` as a full RFC3339 timestamp (date **and** time-of-day). Any past or future instant is accepted, so transactions can be backdated or future-dated; it is not date-only or normalized to midnight. Quick-entry execute and split-bill creation parse `transaction_at` the same way. Clients send the user-picked local datetime normalized to UTC.
- List endpoints for wallets, categories, transactions, quick entry templates, category budgets, debts, installments, subscriptions, recurring transactions, and tags support `limit`, `offset`, and `sort`.
- Generic wallet create/update/delete endpoints reject direct `goal` wallet writes; goal-managed wallets are created and retained by the financial goal workflow.
- `POST/PUT /api/v1/quick-entry-templates` accept `wallet_id` and `to_wallet_id` for wallets owned by the authenticated user or joined shared wallets; categories remain owned by the template user.
- `POST/PUT /api/v1/recurring-transactions` accept `wallet_id` and `to_wallet_id` for wallets owned by the authenticated user or joined shared wallets; categories remain owned by the recurring-rule user.
- `POST/PUT /api/v1/installments` and `POST/PUT /api/v1/subscriptions` accept `wallet_id` for wallets owned by the authenticated user or joined shared wallets; expense categories remain owned by the tracker user.
- `POST /api/v1/debts` accepts `wallet_id` for wallets owned by the authenticated user or joined shared wallets; disbursement and payment categories remain owned by the debt user.
- `POST /api/v1/wallets/:id/invites` accepts an optional `role` (`member` (read+write, default) or `viewer` (read-only)) alongside `email`. The role is stored on `wallet_shares.role`; `viewer` shares can read the wallet/transactions/reports/exports/analytics but are denied every shared-wallet write path. `GET /api/v1/wallets`, `GET /api/v1/wallets/:id`, and `GET /api/v1/wallets/:id/members` surface the real share role.
- `GET /api/v1/goals` currently returns all accessible goals as a JSON array ordered by `created_at DESC`; it does not yet return `{goals, pagination}`.
- Financial goal creation and invite acceptance create goal wallets in the same PostgreSQL transaction. Goal wallet names include a goal ID suffix so duplicate goal names can coexist. Joined goal invitations cannot be rejected later via the invitation-response endpoint.
- `GET /api/v1/dashboard/summary?month=YYYY-MM` returns monthly summary data scoped to the authenticated user.
- `GET /api/v1/dashboard/cashflow-trend?months=6` returns income/expense trends.
- `GET /api/v1/dashboard/expense-distribution?month=YYYY-MM` returns category expense distribution.
- `GET /api/v1/dashboard/forecast?month=YYYY-MM` returns spend forecasting and overbudget warnings.
- Subscriptions accept and return optional `account_detail`.
- `POST /api/v1/goals`, `GET /api/v1/goals`, `GET /api/v1/goals/:id`, `POST /api/v1/goals/:id/members`, and `PUT /api/v1/goals/:id/members/:user_id/respond` implement financial goals and invitations. Contributions to goals are made via `transfer` transactions to the goal wallet.
- `PUT /api/v1/goals/:id` updates name/target/deadline and accepts an optional `status` (`active`/`achieved`/`cancelled`) to transition the goal lifecycle. The owner-only update writes status via `COALESCE`, so a status-less update preserves the current value; invalid status values are rejected with 400.

## Database Migration Rules

- Add new SQL files in numeric order under `migrations`.
- Migration files are executed in full by the native Go migration runner. Keep them as forward-only SQL for this project; do not include goose-style `Down` blocks in files consumed by `internal/db/migrate.go`.
- Prefer backward-compatible migrations for existing data, for example `NOT NULL DEFAULT ''` for new optional text fields.
- When adding user-owned references, enforce user ownership in database constraints where practical, following `000005_user_owned_foreign_keys.sql`.
- Current later migrations include financial goals (`000007`), tags (`000008`), category hierarchy (`000009`), transaction tag ownership (`000010`), category parent ownership/type checks (`000011`), shared-wallet compatibility migrations through debt records (`000016`-`000019`), sent alerts (`000020`), wallet display metadata (`000021`), user profile fields (`000022`), password reset tokens (`000023`), export jobs (`000024`), notification rules (`000025`), the shared-wallet `role` column (`000026`, `wallet_shares.role` ∈ {`member`,`viewer`} default `member`), account-level partner links (`000027`-`000028`), wallet icon (`000029`), entity colors/icons (`000030`), and category icon/color plus user-arrangeable `position` with per-user backfill (`000031`).
- After adding migrations, run `make verify` so Docker and integration tests apply them.

## Known Tech Debt

These are tracked, intentionally-deferred issues. They are **not** live bugs today — do not "fix" them with a large unprompted refactor, but be aware when touching the area.

- **HTTP status mapping is partly string-based** (`internal/httpx/response.go`). `WriteError` resolves status via typed sentinels/`PublicError` **and** fragile fallbacks that match error message text (e.g. `strings.Contains(msg, "already exists") -> 409`, prefix `"invalid" -> 400`). Today the messages happen to contain the matched substrings so statuses are correct, but **changing an error's wording can silently change its HTTP status**. When adding errors, prefer wrapping a sentinel (`fmt.Errorf("...: %w", httpx.ErrValidation)`) or returning `httpx.NewPublicError(msg, status)` directly rather than relying on the text fallback. Long-term fix: make sentinels the sole contract and delete the string fallbacks (~160 call sites; do incrementally).
- **Shared-wallet access check is duplicated, not centralized.** The composite FKs that used to enforce ownership at the DB level are intentionally dropped for shared wallets (migrations `000012`, `000016`-`000019`); access is re-enforced in app code. Only the `transaction` module uses the canonical `wallet.AccessChecker`. The owner-or-joined-member SQL predicate is copy-pasted across `transaction/repository.go`, `tracker/repository_helpers.go`, `quickentry/repository.go`, and `recurring/repository.go`. All current write paths to a shared wallet funnel through `transaction/repository.go applyDeltas` (scoped to owner-or-member, returns `pgx.ErrNoRows` on 0 rows), so there is **no known cross-user write hole** — but a new module that forgets the predicate would create one. **Invariant for any new shared-wallet write path: validate access via the canonical checker (or the same SQL predicate) before mutating balances.** Optional hardening: extract one shared predicate, and/or add a DB-level CHECK/trigger as defense-in-depth.

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
