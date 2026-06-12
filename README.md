# Affluena Backend API

Affluena is an API-first personal finance backend. It implements native auth, wallets, categories, transactions, quick entry templates, budgeting, trackers, recurring transactions, and debt/loan tracking.

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

## Endpoints

Public:

- `GET /healthz`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`

Protected with `Authorization: Bearer <access_token>`:

- `GET /api/v1/auth/me`
- `POST /api/v1/wallets`
- `GET /api/v1/wallets`
- `GET /api/v1/wallets/:id`
- `PUT /api/v1/wallets/:id`
- `DELETE /api/v1/wallets/:id`
- `POST /api/v1/categories`
- `GET /api/v1/categories`
- `GET /api/v1/categories/:id`
- `PUT /api/v1/categories/:id`
- `DELETE /api/v1/categories/:id`
- `POST /api/v1/transactions`
- `GET /api/v1/transactions`
- `GET /api/v1/transactions/:id`
- `PUT /api/v1/transactions/:id`
- `DELETE /api/v1/transactions/:id`
- `POST /api/v1/quick-entry-templates`
- `GET /api/v1/quick-entry-templates`
- `GET /api/v1/quick-entry-templates/:id`
- `PUT /api/v1/quick-entry-templates/:id`
- `DELETE /api/v1/quick-entry-templates/:id`
- `POST /api/v1/quick-entry-templates/:id/execute`
- `POST /api/v1/category-budgets`
- `GET /api/v1/category-budgets?month=YYYY-MM`
- `GET /api/v1/category-budgets/:id`
- `PUT /api/v1/category-budgets/:id`
- `DELETE /api/v1/category-budgets/:id`
- `POST /api/v1/debts`
- `GET /api/v1/debts`
- `GET /api/v1/debts/:id`
- `PUT /api/v1/debts/:id`
- `DELETE /api/v1/debts/:id`
- `POST /api/v1/debts/:id/pay`
- `POST /api/v1/installments`
- `GET /api/v1/installments`
- `GET /api/v1/installments/:id`
- `PUT /api/v1/installments/:id`
- `DELETE /api/v1/installments/:id`
- `POST /api/v1/installments/:id/pay`
- `POST /api/v1/subscriptions`
- `GET /api/v1/subscriptions`
- `GET /api/v1/subscriptions/:id`
- `PUT /api/v1/subscriptions/:id`
- `DELETE /api/v1/subscriptions/:id`
- `POST /api/v1/subscriptions/:id/pay`
- `POST /api/v1/recurring-transactions`
- `GET /api/v1/recurring-transactions`
- `GET /api/v1/recurring-transactions/:id`
- `PUT /api/v1/recurring-transactions/:id`
- `DELETE /api/v1/recurring-transactions/:id`
- `POST /api/v1/recurring-transactions/:id/run`

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

Create expense transaction:

```bash
curl -s http://localhost:8080/api/v1/transactions \
  -H 'content-type: application/json' \
  -H "authorization: Bearer $ACCESS_TOKEN" \
  -d '{"type":"expense","wallet_id":"<wallet_id>","category_id":"<category_id>","amount_minor":50000,"note":"Lunch"}'
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
  -d '{"name":"Meal Plan","wallet_id":"<wallet_id>","category_id":"<expense_category_id>","amount_minor":250000,"billing_cycle":"weekly","next_due_date":"2026-06-12"}'

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
