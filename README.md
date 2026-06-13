# Affluena-API

Affluena-API is an API-first personal finance backend. It implements native auth, wallets, categories, transactions, quick entry templates, budgeting, trackers, recurring transactions, and debt/loan tracking.

## Stack

- Go 1.26
- Gin
- PostgreSQL with pgx
- Native JWT auth
- Docker Compose

## Run Locally

```bash
cp .env.example .env
docker compose up --build
```

The API listens on `http://localhost:8080`. Migrations run automatically when `RUN_MIGRATIONS=true`.

For local Go execution without Docker:

```bash
docker compose up postgres
go run ./cmd/api
```

## Verify

Run the full pre-commit/pre-push gate:

```bash
make verify
```

The verify script checks Go formatting, unit tests, `go vet`, API build, Postman JSON validity, Docker Compose rebuild, integration tests against PostgreSQL, and `/healthz`.

## Endpoints

Public:

- `GET /healthz`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`

Protected with `Authorization: Bearer <access_token>`:

- `GET /api/v1/auth/me`
- `GET /api/v1/dashboard/summary?month=YYYY-MM`
- `POST /api/v1/wallets`
- `GET /api/v1/wallets[?limit=100&offset=0&sort=created_at_desc]`
- `GET /api/v1/wallets/:id`
- `PUT /api/v1/wallets/:id`
- `DELETE /api/v1/wallets/:id`
### 📂 Categories (Up to 3 Levels)
- `POST /api/v1/categories` (Supports same-user, same-type `parent_id` for nesting)
- `GET /api/v1/categories[?type=income|expense&limit=100&offset=0&sort=type_name_asc]`
- `GET /api/v1/categories/:id`
- `PUT /api/v1/categories/:id` (Supports updating same-user, same-type `parent_id`)
- `DELETE /api/v1/categories/:id`
- `POST /api/v1/tags`
- `GET /api/v1/tags[?limit=100&offset=0&sort=created_at_desc]`
- `GET /api/v1/tags/:id`
- `PUT /api/v1/tags/:id`
- `DELETE /api/v1/tags/:id`
- `POST /api/v1/transactions`
- `GET /api/v1/transactions[?type=income|expense|transfer|adjustment&wallet_id=<id>&category_id=<id>&tag_id=<id>&from=YYYY-MM-DD&to=YYYY-MM-DD&limit=100&offset=0&sort=transaction_at_desc]`
- `GET /api/v1/transactions/:id`
- `PUT /api/v1/transactions/:id`
- `DELETE /api/v1/transactions/:id`
- `POST /api/v1/quick-entry-templates`
- `GET /api/v1/quick-entry-templates[?limit=100&offset=0&sort=name_asc]`
- `GET /api/v1/quick-entry-templates/:id`
- `PUT /api/v1/quick-entry-templates/:id`
- `DELETE /api/v1/quick-entry-templates/:id`
- `POST /api/v1/quick-entry-templates/:id/execute`
- `POST /api/v1/category-budgets`
- `GET /api/v1/category-budgets?month=YYYY-MM[&limit=100&offset=0&sort=created_at_desc]`
- `GET /api/v1/category-budgets/:id`
- `PUT /api/v1/category-budgets/:id`
- `DELETE /api/v1/category-budgets/:id`
- `POST /api/v1/debts`
- `GET /api/v1/debts[?limit=100&offset=0&sort=opened_at_desc]`
- `GET /api/v1/debts/:id`
- `PUT /api/v1/debts/:id`
- `DELETE /api/v1/debts/:id`
- `POST /api/v1/debts/:id/pay`
- `POST /api/v1/installments`
- `GET /api/v1/installments[?limit=100&offset=0&sort=created_at_desc]`
- `GET /api/v1/installments/:id`
- `PUT /api/v1/installments/:id`
- `DELETE /api/v1/installments/:id`
- `POST /api/v1/installments/:id/pay`
- `POST /api/v1/subscriptions`
- `GET /api/v1/subscriptions[?limit=100&offset=0&sort=next_due_date_asc]`
- `GET /api/v1/subscriptions/:id`
- `PUT /api/v1/subscriptions/:id`
- `DELETE /api/v1/subscriptions/:id`
- `POST /api/v1/subscriptions/:id/pay`
- `POST /api/v1/recurring-transactions`
- `GET /api/v1/recurring-transactions[?limit=100&offset=0&sort=next_run_at_asc]`
- `GET /api/v1/recurring-transactions/:id`
- `PUT /api/v1/recurring-transactions/:id`
- `DELETE /api/v1/recurring-transactions/:id`
- `POST /api/v1/recurring-transactions/:id/run`
- `POST /api/v1/goals`
- `GET /api/v1/goals`
- `GET /api/v1/goals/:id`
- `POST /api/v1/goals/:id/members`
- `PUT /api/v1/goals/:id/members/:user_id/respond`

## Pagination And Sorting

List endpoints for wallets, categories, transactions, quick entry templates, category budgets, debts, installments, subscriptions, and recurring transactions support `limit`, `offset`, and `sort`.

- `limit` defaults to `100`, must be positive, and is capped at `200`.
- `offset` defaults to `0`.
- Responses include the collection plus `pagination`: `{"limit":100,"offset":0,"total":123}`.

Supported `sort` values:

- Wallets: `created_at_desc`, `created_at_asc`, `name_asc`, `name_desc`, `balance_desc`, `balance_asc`.
- Categories: `type_name_asc`, `type_name_desc`, `name_asc`, `name_desc`, `created_at_desc`, `created_at_asc`.
- Transactions: `transaction_at_desc`, `transaction_at_asc`, `created_at_desc`, `created_at_asc`, `amount_desc`, `amount_asc`.
- Quick entry templates: `name_asc`, `name_desc`, `created_at_desc`, `created_at_asc`.
- Category budgets: `created_at_desc`, `created_at_asc`, `limit_desc`, `limit_asc`, `spent_desc`, `spent_asc`.
- Debts: `opened_at_desc`, `opened_at_asc`, `due_date_asc`, `due_date_desc`, `amount_desc`, `amount_asc`.
- Installments: `created_at_desc`, `created_at_asc`, `name_asc`, `name_desc`, `due_day_asc`, `due_day_desc`.
- Subscriptions: `next_due_date_asc`, `next_due_date_desc`, `created_at_desc`, `created_at_asc`, `name_asc`, `name_desc`.
- Recurring transactions: `next_run_at_asc`, `next_run_at_desc`, `created_at_desc`, `created_at_asc`, `name_asc`, `name_desc`.
- Tags: `created_at_desc`, `created_at_asc`, `name_asc`, `name_desc`.

`GET /api/v1/goals` is the current exception: it returns a JSON array of accessible goals ordered by `created_at DESC`, without pagination metadata.

## Example Flow

Register:

```bash
curl -s http://localhost:8080/api/v1/auth/register \
  -H 'content-type: application/json' \
  -d '{"email":"user@example.com","password":"password123"}'
```

Create wallet:

```bash
curl -s http://localhost:8080/api/v1/wallets \
  -H 'content-type: application/json' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -d '{"name":"BCA Main","type":"bank","currency_code":"IDR","balance_minor":10000000}'
```

Create expense category:

```bash
curl -s http://localhost:8080/api/v1/categories \
  -H 'content-type: application/json' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -d '{"name":"Lunch","type":"expense"}'
```

List expense categories:

```bash
curl -s 'http://localhost:8080/api/v1/categories?type=expense' \
  -H "authorization: Bearer $ACCESS_TOKEN"
```

Create expense transaction:

```bash
curl -s http://localhost:8080/api/v1/transactions \
  -H 'content-type: application/json' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -d '{"type":"expense","wallet_id":"<wallet_id>","category_id":"<category_id>","amount_minor":50000,"note":"Lunch"}'
```

List filtered transactions:

```bash
curl -s 'http://localhost:8080/api/v1/transactions?type=expense&tag_id=<tag_id>&from=2026-06-01&to=2026-06-30' \
  -H "authorization: Bearer $ACCESS_TOKEN"
```

Get dashboard summary:

```bash
curl -s 'http://localhost:8080/api/v1/dashboard/summary?month=2026-06' \
  -H "authorization: Bearer $ACCESS_TOKEN"
```

Execute a quick entry template:

```bash
curl -s http://localhost:8080/api/v1/quick-entry-templates/<template_id>/execute \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -X POST
```

Create category budget:

```bash
curl -s http://localhost:8080/api/v1/category-budgets \
  -H 'content-type: application/json' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -d '{"category_id":"<expense_category_id>","month":"2026-06","limit_minor":2000000}'
```

List category budget summaries:

```bash
curl -s 'http://localhost:8080/api/v1/category-budgets?month=2026-06' \
  -H "authorization: Bearer $ACCESS_TOKEN"
```

Create and receive payment for a receivable:

```bash
curl -s http://localhost:8080/api/v1/debts \
  -H 'content-type: application/json' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -d '{"type":"receivable","counterparty_name":"Friend","wallet_id":"<wallet_id>","disbursement_category_id":"<expense_category_id>","payment_category_id":"<income_category_id>","principal_amount_minor":500000,"due_date":"2026-07-01","note":"Short loan"}'

curl -s http://localhost:8080/api/v1/debts/<debt_id>/pay \
  -H 'content-type: application/json' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -d '{"amount_minor":200000,"note":"First repayment"}'
```

Create and pay down a payable:

```bash
curl -s http://localhost:8080/api/v1/debts \
  -H 'content-type: application/json' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -d '{"type":"payable","counterparty_name":"Family","wallet_id":"<wallet_id>","disbursement_category_id":"<income_category_id>","payment_category_id":"<expense_category_id>","principal_amount_minor":1000000,"due_date":"2026-08-01"}'

curl -s http://localhost:8080/api/v1/debts/<debt_id>/pay \
  -H 'content-type: application/json' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -d '{"amount_minor":250000}'
```

Create and pay an installment:

```bash
curl -s http://localhost:8080/api/v1/installments \
  -H 'content-type: application/json' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -d '{"name":"Gym Membership","wallet_id":"<wallet_id>","category_id":"<expense_category_id>","total_amount_minor":900000,"monthly_amount_minor":300000,"tenor_months":3,"start_date":"2026-06-01","due_day":5}'

curl -s http://localhost:8080/api/v1/installments/<installment_id>/pay \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -X POST
```

Create and pay a subscription:

```bash
curl -s http://localhost:8080/api/v1/subscriptions \
  -H 'content-type: application/json' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -d '{"name":"Google One","account_detail":"personal@example.com","wallet_id":"<wallet_id>","category_id":"<expense_category_id>","amount_minor":250000,"billing_cycle":"weekly","next_due_date":"2026-06-12"}'

curl -s http://localhost:8080/api/v1/subscriptions/<subscription_id>/pay \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -X POST
```

Create and manually run a recurring transaction:

```bash
curl -s http://localhost:8080/api/v1/recurring-transactions \
  -H 'content-type: application/json' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -d '{"name":"Monthly internet","type":"expense","wallet_id":"<wallet_id>","category_id":"<expense_category_id>","amount_minor":350000,"frequency":"monthly","interval_count":1,"next_run_at":"2026-06-01T00:00:00Z"}'

curl -s http://localhost:8080/api/v1/recurring-transactions/<recurring_id>/run \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -X POST
```

## Money Representation

Amounts use `amount_minor` as integer minor units. For IDR, `50000` means Rp50,000. This avoids floating point rounding in balance changes.

## Transaction Balance Rules

- `income`: adds `amount_minor` to `wallet_id`
- `expense`: subtracts `amount_minor` from `wallet_id`
- `transfer`: subtracts from `wallet_id`, adds to `to_wallet_id`
- `adjustment`: applies signed `amount_minor` directly to `wallet_id`

Create, update, delete, and quick entry execution use database transactions so wallet balances and transaction rows change atomically.

Transaction `tag_ids` must be valid UUIDs owned by the authenticated user. Duplicate tag IDs in a request are stored once, and transaction listing can filter by one owned tag with `tag_id=<id>`.

Category `parent_id` must point to a category owned by the authenticated user with the same category `type`. Category trees are limited to 3 levels and cyclic parent relationships are rejected.

Financial goal creation and invitation acceptance create goal wallets atomically. Goal wallet names include the goal ID suffix, so duplicate goal names can safely coexist. Rejected invitations are not returned as accessible goals, and `PUT /api/v1/goals/:id/members/:user_id/respond` only lets the authenticated member respond for their own `:user_id`.

Debt creation and debt payment endpoints also run atomically:

- `receivable` creation creates an expense transaction and decreases the wallet.
- `receivable` payment creates an income transaction and increases the wallet.
- `payable` creation creates an income transaction and increases the wallet.
- `payable` payment creates an expense transaction and decreases the wallet.

Debt statuses are `open`, `partial`, `paid_off`, and `cancelled`. Payment over the remaining amount is rejected. `DELETE /api/v1/debts/:id` soft-cancels tracking and keeps transaction history intact.

Installment and subscription payment endpoints also run atomically: the API creates an expense transaction, updates wallet balance, and advances tracker state in one PostgreSQL transaction.

Quick entry execution, installment payment, subscription payment, and manual recurring run endpoints accept an empty request body for one-click daily actions. Optional JSON bodies can still override fields such as payment date, note, wallet, category, or scheduled run time where the endpoint supports it.

Monthly subscription and recurring schedules clamp month-end dates to the target month's last day. For example, January 31 advances to February 28 or 29 instead of rolling into March.

Recurring transactions are executed atomically too. Each occurrence stores a unique `(rule_id, scheduled_for)` run record before creating the transaction, so scheduler races cannot double-charge the same scheduled occurrence. The native scheduler is controlled by:

- `RECURRING_SCHEDULER_ENABLED`
- `RECURRING_SCHEDULER_INTERVAL`
- `RECURRING_SCHEDULER_BATCH_SIZE`

### 📊 Dashboard & Analytics
- `GET /api/v1/dashboard/summary` - Get monthly financial overview.
- `GET /api/v1/dashboard/cashflow-trend?months=6` - Get income/expense trends over time. `months` must be between 1 and 12.
- `GET /api/v1/dashboard/expense-distribution?month=YYYY-MM` - Get breakdown of expenses by category.
- `GET /api/v1/dashboard/forecast?month=YYYY-MM` - Predict if spending will exceed budget based on daily average. Months with no budget remain `safe` instead of overbudget.
