# Affluena API Contract

This document serves as the primary contract for frontend integration (React UI/Stage 1-10). It details the conventions, endpoints, request shapes, and business rules of the Affluena API.

## General Conventions

- **Base URL (Local)**: `http://localhost:8080`
- **API Prefix**: `/api/v1`
- **Auth Scheme**: Bearer Token (`Authorization: Bearer <access_token>`)
- **Rate Limits (Auth)**: Defaults to 5 req/s with a burst of 10. `429 Too Many Requests` is returned if exceeded.
- **Content Type**: `application/json` for both requests and responses (except export endpoints).
- **Money Format**: Integer minor units (e.g., IDR 1.000 = `1000`).
- **Date/Time Format**: ISO 8601 strings (e.g., `2024-01-01T15:04:05Z`).
- **Pagination Format**:
  Requests use query parameters: `?limit=10&offset=0&sort=created_at_desc`
  Responses wrap collections:
  ```json
  {
    "collection": [...],
    "pagination": { "total": 100, "limit": 10, "offset": 0 }
  }
  ```
- **Error Response Format**:
  ```json
  { "error": "human readable error message" }
  ```

## Modules

### Auth
- `POST /api/v1/auth/register` - `{ name, email, password }` -> `{ user, tokens: { access_token, refresh_token } }`
- `POST /api/v1/auth/login` - `{ email, password }` -> `{ user, tokens: { access_token, refresh_token } }`
- `POST /api/v1/auth/refresh` - `{ refresh_token }` -> `{ tokens: { access_token, refresh_token } }`
- `GET /api/v1/auth/me` - Profile data -> `{ id, name, email, created_at, updated_at }`
- *Note:* Logout/Revoke is handled purely client-side (token deletion) currently.

### Dashboard
- `GET /api/v1/dashboard/summary?month=YYYY-MM` - Summary stats including net worth, cashflow, budgets, upcoming trackers.
- `GET /api/v1/dashboard/cashflow-trend?months=6` - Time series data of income vs expense.
- `GET /api/v1/dashboard/expense-distribution?month=YYYY-MM` - Breakdowns by category.
- `GET /api/v1/dashboard/forecast?month=YYYY-MM` - Spend forecasting and budget alerts.

### Wallet
- `POST /api/v1/wallets` - `{ name, type, balance_minor }`
- `GET /api/v1/wallets` - Paginated wallet list.
- `GET /api/v1/wallets/:id`
- `PUT /api/v1/wallets/:id` - `{ name, type }` (Balance updates via transactions).
- `DELETE /api/v1/wallets/:id` - Soft archive behavior based on constraints.
- `POST /api/v1/wallets/:id/invites` - Share wallet. `{ user_id_to_invite }`
- `PATCH /api/v1/wallets/:id/members/:member_id` - `{ status: "joined"|"rejected" }`
- *Note:* Direct writes to `type="goal"` wallets are rejected.

### Category
- `POST /api/v1/categories` - `{ name, type, parent_id, icon, color }`
- `GET /api/v1/categories?type=income|expense` - List (optionally filtered by type).
- `GET /api/v1/categories/:id`
- `PUT /api/v1/categories/:id` - `{ name, icon, color, parent_id }`
- `DELETE /api/v1/categories/:id`
- *Note:* Tree depth max 3 levels. Parent and child must be the same type.

### Tag
- `POST /api/v1/tags` - `{ name, color }`
- `GET /api/v1/tags`
- `GET /api/v1/tags/:id`
- `PUT /api/v1/tags/:id` - `{ name, color }`
- `DELETE /api/v1/tags/:id`

### Transaction
- `POST /api/v1/transactions` - `{ type, amount_minor, date, notes, wallet_id, to_wallet_id, category_id, tag_ids }`
- `GET /api/v1/transactions` - Filter by `type`, `wallet_id`, `category_id`, `tag_id`, `from`, `to`.
- `GET /api/v1/transactions/:id`
- `PUT /api/v1/transactions/:id` - Fields same as POST.
- `DELETE /api/v1/transactions/:id`
- `POST /api/v1/transactions/split` - `{ transaction_id, splits: [{ user_id, amount_minor }] }`
- *Note:* `type` must be `income`, `expense`, `transfer`, or `adjustment`. Adjustments fix balance drifts. Transfers require `to_wallet_id`. Category belongs to the user regardless of shared wallet.

### Quick Entry
- `POST /api/v1/quick-entry-templates` - `{ name, type, amount_minor, category_id, wallet_id, to_wallet_id, notes, tag_ids }`
- `GET /api/v1/quick-entry-templates`
- `GET /api/v1/quick-entry-templates/:id`
- `PUT /api/v1/quick-entry-templates/:id`
- `DELETE /api/v1/quick-entry-templates/:id`
- `POST /api/v1/quick-entry-templates/:id/execute` - Applies template directly into a new transaction.

### Budget
- `POST /api/v1/category-budgets` - `{ category_id, amount_minor, month }`
- `GET /api/v1/category-budgets`
- `GET /api/v1/category-budgets/:id`
- `PUT /api/v1/category-budgets/:id`
- `DELETE /api/v1/category-budgets/:id`
- *Note:* Budgets are personal. Shared wallet expenses by other members do not decrement the owner's personal category budget.

### Debt
- `POST /api/v1/debts` - `{ type: "payable"|"receivable", counterparty_name, amount_minor, remaining_minor, start_date, due_date, status, wallet_id, disbursement_category_id, payment_category_id, notes }`
- `GET /api/v1/debts`
- `GET /api/v1/debts/:id`
- `PUT /api/v1/debts/:id`
- `DELETE /api/v1/debts/:id` - Deletes only if no payments exist. Acts as a soft-cancel if allowed.
- `POST /api/v1/debts/:id/pay` - `{ amount_minor, date, wallet_id, category_id, notes }`

### Tracker (Installments & Subscriptions)
- `POST /api/v1/installments` - `{ name, total_amount_minor, monthly_amount_minor, tenor_months, paid_months, start_date, wallet_id, category_id, is_active }`
- `GET /api/v1/installments`
- `GET /api/v1/installments/:id`
- `PUT /api/v1/installments/:id`
- `DELETE /api/v1/installments/:id`
- `POST /api/v1/installments/:id/pay` - `{ amount_minor, date, wallet_id, category_id, notes }`
- `POST /api/v1/subscriptions` - `{ name, amount_minor, billing_cycle: "monthly"|"yearly", start_date, next_billing_date, account_detail, wallet_id, category_id, is_active }`
- `GET /api/v1/subscriptions`
- `GET /api/v1/subscriptions/:id`
- `PUT /api/v1/subscriptions/:id`
- `DELETE /api/v1/subscriptions/:id`
- `POST /api/v1/subscriptions/:id/pay`

### Recurring
- `POST /api/v1/recurring-transactions` - `{ type, amount_minor, wallet_id, to_wallet_id, category_id, schedule: "daily"|"weekly"|"monthly", next_run_at, status: "active"|"paused" }`
- `GET /api/v1/recurring-transactions`
- `GET /api/v1/recurring-transactions/:id`
- `PUT /api/v1/recurring-transactions/:id`
- `DELETE /api/v1/recurring-transactions/:id`
- `POST /api/v1/recurring-transactions/:id/run` - Manual force execution.

### Goal
- `POST /api/v1/goals` - `{ name, target_amount_minor, current_amount_minor, deadline, notes }`
- `GET /api/v1/goals` - Array format (not `{collection, pagination}`).
- `GET /api/v1/goals/:id`
- `POST /api/v1/goals/:id/members` - Invite to goal.
- `PUT /api/v1/goals/:id/members/:user_id/respond` - `{ status: "joined" }`.
- *Note:* Contributions are made via `POST /api/v1/transactions` with `type=transfer` and `to_wallet_id=<goal_wallet_id>`.

### Reports / Export
- `GET /api/v1/export/csv?from=YYYY-MM-DD&to=YYYY-MM-DD`
- *Note:* Response is raw CSV bytes, `Content-Type: text/csv`.

### Activity
- `GET /api/v1/activities` - Pagination list of user activities.

## HTTP Status Codes
- `200 OK` - Success, reading data, or updates.
- `201 Created` - Resource created.
- `400 Bad Request` - Validation errors or invalid input format.
- `401 Unauthorized` - Missing or invalid token.
- `403 Forbidden` - Cross-user data access violation.
- `404 Not Found` - Resource doesn't exist.
- `409 Conflict` - Resource constraint violations (e.g., cyclic parent category).
- `500 Internal Server Error` - Unhandled panics/database drops.

## OpenAPI Specification

> **TODO**: OpenAPI generation/manual spec will be handled in a later sprint. For now, rely on this contract and the provided Postman collection.
