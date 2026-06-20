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
  Responses wrap collections with a specific key per resource (e.g. `wallets`, `categories`, `tags`, `transactions`, `budgets`, `debts`, `templates`, `installments`, `subscriptions`, `recurring_transactions`):
  ```json
  {
    "wallets": [...],
    "pagination": { "total": 100, "limit": 10, "offset": 0 }
  }
  ```
- **Error Response Format**:
  ```json
  { "error": "human readable error message" }
  ```

## Modules

### Auth
- `POST /api/v1/auth/register` - `{ email, password }` -> `{ user, tokens: { access_token, refresh_token } }`
- `POST /api/v1/auth/login` - `{ email, password }` -> `{ user, tokens: { access_token, refresh_token } }`
- `POST /api/v1/auth/refresh` - `{ refresh_token }` -> `{ user: { id, email, created_at, updated_at }, tokens: { access_token, refresh_token } }`
- `GET /api/v1/auth/me` - Profile data -> `{ user: { id, email, created_at, updated_at } }`
- `PUT /api/v1/auth/account` - Update account -> `{ name, avatar_url }` -> `{ user }`
- `PUT /api/v1/auth/password` - Change password -> `{ current_password, new_password }`
- `GET /api/v1/auth/sessions` - List sessions -> `{ sessions: [...] }`
- `DELETE /api/v1/auth/sessions/:id` - Revoke session
- `POST /api/v1/auth/forgot-password` - `{ email }` -> 204
- `POST /api/v1/auth/reset-password` - `{ token, new_password }` -> 204
- *Note:* Logout is handled client-side (token deletion). Session revocation is available via `DELETE /api/v1/auth/sessions/:id`.

### Dashboard
- `GET /api/v1/dashboard/summary?month=YYYY-MM` - Summary stats including net worth, cashflow, budgets, upcoming trackers.
- `GET /api/v1/dashboard/cashflow-trend?months=6` - Time series data of income vs expense.
- `GET /api/v1/dashboard/expense-distribution?month=YYYY-MM` - Breakdowns by category.
- `GET /api/v1/dashboard/forecast?month=YYYY-MM` - Spend forecasting and budget alerts.

### Wallet
- `POST /api/v1/wallets` - `{ name, type, currency_code, balance_minor }`
- `GET /api/v1/wallets` - Paginated wallet list.
- `GET /api/v1/wallets/:id`
- `PUT /api/v1/wallets/:id` - `{ name, type, currency_code }` (Balance updates via transactions).
- `DELETE /api/v1/wallets/:id` - Soft archive behavior based on constraints.
- `POST /api/v1/wallets/:id/invites` - Share wallet. `{ email }`
- `PATCH /api/v1/wallets/:id/members/:member_id` - `{ status: "joined"|"rejected" }`
- *Note:* Direct writes to `type="goal"` wallets are rejected. Supported types: `cash`, `bank`, `e_wallet`, `investment`. `goal_id` is not sent by frontend when creating a wallet; goal wallets are created and managed internally by the goal module.

### Category
- `POST /api/v1/categories` - `{ name, type, parent_id }`
- `GET /api/v1/categories?type=income|expense` - List (optionally filtered by type).
- `GET /api/v1/categories/:id`
- `PUT /api/v1/categories/:id` - `{ name, type, parent_id }`
- `DELETE /api/v1/categories/:id`
- *Note:* Tree depth max 3 levels. Parent and child must be the same type.

### Tag
- `POST /api/v1/tags` - `{ name }`
- `GET /api/v1/tags`
- `GET /api/v1/tags/:id`
- `PUT /api/v1/tags/:id` - `{ name }`
- `DELETE /api/v1/tags/:id`

### Transaction
- `POST /api/v1/transactions` - `{ type, amount_minor, transaction_at, note, wallet_id, to_wallet_id, category_id, tag_ids }`
- `GET /api/v1/transactions` - Filter by `type`, `wallet_id`, `category_id`, `tag_id`, `from`, `to`.
- `GET /api/v1/transactions/:id`
- `PUT /api/v1/transactions/:id` - Fields same as POST.
- `DELETE /api/v1/transactions/:id`
- `POST /api/v1/transactions/split` - `{ wallet_id, category_id, total_amount_minor, transaction_at, note, tag_ids, splits: [{ counterparty_name, amount_minor, disbursement_category_id, payment_category_id }] }`
- *Note:* `type` must be `income`, `expense`, `transfer`, or `adjustment`. Adjustments fix balance drifts. Transfers require `to_wallet_id`. Category belongs to the user regardless of shared wallet.
- *Split Bill Rules:* Parent transaction uses `total_amount_minor`, decreasing wallet balance by the total bill. Splits create `payable`/`receivable` debts. Full split is valid if total split == total amount. Over split is invalid. Transaction amount cannot be 0.

### Quick Entry
- `POST /api/v1/quick-entry-templates` - `{ name, type, amount_minor, category_id, wallet_id, to_wallet_id, note, tag_ids }`
- `GET /api/v1/quick-entry-templates`
- `GET /api/v1/quick-entry-templates/:id`
- `PUT /api/v1/quick-entry-templates/:id`
- `DELETE /api/v1/quick-entry-templates/:id`
- `POST /api/v1/quick-entry-templates/:id/execute` - `{ transaction_at, note }` (Applies template directly into a new transaction.)

### Budget
- `POST /api/v1/category-budgets` - `{ category_id, limit_minor, month }`
- `GET /api/v1/category-budgets`
- `GET /api/v1/category-budgets/:id`
- `PUT /api/v1/category-budgets/:id` - `{ limit_minor }`
- `DELETE /api/v1/category-budgets/:id`
- `GET /api/v1/category-budgets/alerts?month=YYYY-MM` - Get budget alerts for a specific month.
  Response shape:
  ```json
  {
    "alerts": [
      {
        "id": "string",
        "budget_id": "string",
        "category_id": "string",
        "category_name": "string",
        "title": "string",
        "message": "string",
        "threshold": 80,
        "severity": "warning",
        "usage_percent": 85.5,
        "spent_minor": 85000,
        "limit_minor": 100000,
        "notified_at": "string",
        "month": "string"
      }
    ]
  }
  ```
- `GET /api/v1/category-budgets/report?month=YYYY-MM` - Get budget report for a specific month.
  Response shape:
  ```json
  {
    "report": [
      {
        "id": "string",
        "user_id": "string",
        "category_id": "string",
        "month": "string",
        "limit_minor": 100000,
        "created_at": "string",
        "updated_at": "string",
        "spent_minor": 85000,
        "remaining_minor": 15000,
        "usage_percent": 85.5,
        "variance_minor": 15000,
        "daily_allowance_minor": 3225,
        "recommendation": "string"
      }
    ],
    "summary": {
      "total_limit_minor": 100000,
      "total_spent_minor": 85000,
      "total_remaining_minor": 15000,
      "safe_count": 1,
      "warning_count": 0,
      "exceeded_count": 0,
      "daily_allowance_minor": 3225,
      "forecast_minor": 95000
    }
  }
  ```
- *Note:* Budgets are personal. Shared wallet expenses by other members do not decrement the owner's personal category budget.

### Debt
- `POST /api/v1/debts` - `{ type: "payable"|"receivable", counterparty_name, wallet_id, disbursement_category_id, payment_category_id, principal_amount_minor, opened_at, due_date, note }`
- `GET /api/v1/debts`
- `GET /api/v1/debts/:id`
- `PUT /api/v1/debts/:id` - `{ counterparty_name, due_date, status, note }`
- `DELETE /api/v1/debts/:id` - Cancel Debt / Soft Cancel. Updates status to `cancelled`. Cannot be paid or counted as active debt. Cancel fails if payments exist.
- `POST /api/v1/debts/:id/pay` - `{ amount_minor, paid_at, note }` (Wallet/category are automatically inferred from the debt configuration.)
- *Note:* `origination_transaction_id` is an internal-only field populated during the split bill workflow and must not be supplied by clients in public creation.

### Tracker (Installments & Subscriptions)
- `POST /api/v1/installments` - `{ name, wallet_id, category_id, total_amount_minor, monthly_amount_minor, tenor_months, remaining_months, start_date, due_day, status, note }`
- `GET /api/v1/installments`
- `GET /api/v1/installments/:id`
- `PUT /api/v1/installments/:id`
- `DELETE /api/v1/installments/:id`
- `POST /api/v1/installments/:id/pay` - `{ paid_at, note }` (Amount is inferred from installment configuration.)
- `POST /api/v1/subscriptions` - `{ name, account_detail, wallet_id, category_id, amount_minor, billing_cycle: "monthly"|"yearly", next_due_date, status, note }`
- `GET /api/v1/subscriptions`
- `GET /api/v1/subscriptions/:id`
- `PUT /api/v1/subscriptions/:id`
- `DELETE /api/v1/subscriptions/:id`
- `POST /api/v1/subscriptions/:id/pay` - `{ paid_at, note }` (Amount is inferred from subscription configuration.)

### Recurring
- `POST /api/v1/recurring-transactions` - `{ name, type, wallet_id, to_wallet_id, category_id, amount_minor, frequency: "weekly"|"monthly", interval_count, next_run_at, end_at, status: "active"|"paused", note }`
- `GET /api/v1/recurring-transactions`
- `GET /api/v1/recurring-transactions/:id`
- `PUT /api/v1/recurring-transactions/:id` - Same as POST.
- `DELETE /api/v1/recurring-transactions/:id`
- `POST /api/v1/recurring-transactions/:id/run` - Manual force execution.

### Goal
- `POST /api/v1/goals` - `{ name, target_amount_minor, deadline }`
- `GET /api/v1/goals` - Array format (not wrapped in pagination).
- `GET /api/v1/goals/:id`
- `POST /api/v1/goals/:id/members` - Invite to goal. `{ email }`
- `PUT /api/v1/goals/:id/members/:user_id/respond` - `{ status: "joined" }`.
- *Note:* Collected amount is calculated from the goal wallet balance. Contributions are made via `POST /api/v1/transactions` with `type=transfer` and `to_wallet_id=<goal_wallet_id>`. There is no dedicated `/goals/:id/contribute` endpoint.

### Reports, Export & Jobs
- `GET /api/v1/export/csv?from=2026-06-01T00:00:00Z&to=2026-06-30T23:59:59Z`
- `GET /api/v1/export/jobs?limit=10&offset=0` - List export jobs.
  Response shape:
  ```json
  {
    "jobs": [
      {
        "id": "string",
        "user_id": "string",
        "format": "csv",
        "from_at": "string",
        "to_at": "string",
        "row_count": 100,
        "status": "completed",
        "created_at": "string"
      }
    ],
    "pagination": { "total": 1, "limit": 10, "offset": 0 }
  }
  ```
- `GET /api/v1/export/jobs/:id` - Get export job detail.
  Response shape:
  ```json
  {
    "id": "string",
    "user_id": "string",
    "format": "csv",
    "from_at": "string",
    "to_at": "string",
    "row_count": 100,
    "status": "completed",
    "created_at": "string"
  }
  ```
- *Note:* `from` and `to` query parameters must be in valid RFC3339 format (e.g. `2026-06-01T00:00:00Z`).
- *Note:* Response for `/export/csv` is raw CSV bytes, `Content-Type: text/csv`.
- *Note:* Triggering `/export/csv` now records an audit row in the `export_jobs` table with status "completed" or "failed".

### Reports
- `GET /api/v1/reports/income?month=YYYY-MM` - Get income report.
- `GET /api/v1/reports/expense?month=YYYY-MM` - Get expense report.
- `GET /api/v1/reports/cashflow?month=YYYY-MM` - Get cashflow report.
- `GET /api/v1/reports/debt?month=YYYY-MM` - Get debt report.
- `GET /api/v1/reports/goal?month=YYYY-MM` - Get goal report.
- `GET /api/v1/reports/overview?month=YYYY-MM` - Get overview report.
  Response shape:
  ```json
  {
    "metrics": [
      {
        "id": "string",
        "label": "string",
        "value_minor": 100000,
        "helper": "string",
        "tone": "string"
      }
    ],
    "rows": [
      {
        "id": "string",
        "name": "string",
        "category": "string",
        "amount_minor": 100000,
        "previous_amount_minor": 90000,
        "change_percent": 11.1,
        "wallet": "string",
        "status": "healthy"
      }
    ]
  }
  ```
- *Note:* Previous-period comparison is provided via `change_percent`. If the user has no data, empty arrays `[]` are returned instead of null.

### Activity
- `GET /api/v1/activities` - Pagination list of user activities.
- `GET /api/v1/activities/:id` - Get activity detail.
  Response shape:
  ```json
  {
    "id": "string",
    "user_id": "string",
    "action_type": "string",
    "entity_type": "string",
    "entity_id": "string",
    "description": "string",
    "created_at": "string"
  }
  ```

### System Logs
- `GET /api/v1/system-logs?limit=N` - List system logs.
  Response shape:
  ```json
  {
    "logs": [
      {
        "id": "string",
        "method": "string",
        "path": "string",
        "status_code": 200,
        "latency_ms": 15,
        "client_ip": "string",
        "user_agent": "string",
        "user_id": "string",
        "request_payload": "string",
        "response_payload": "string",
        "created_at": "string"
      }
    ]
  }
  ```
- `GET /api/v1/system-logs/:id` - Get system log detail.
  Response shape:
  ```json
  {
    "id": "string",
    "method": "string",
    "path": "string",
    "status_code": 200,
    "latency_ms": 15,
    "client_ip": "string",
    "user_agent": "string",
    "user_id": "string",
    "request_payload": "string",
    "response_payload": "string",
    "created_at": "string"
  }
  ```

### Alerts
- `GET /api/v1/alerts?month=YYYY-MM` - Get alerts feed for a specific month.
  Response shape:
  ```json
  {
    "alerts": [
      {
        "id": "string",
        "type": "budget",
        "title": "string",
        "module": "string",
        "description": "string",
        "severity": "warning",
        "created_at": "string",
        "action_path": "string"
      }
    ]
  }
  ```
- `GET /api/v1/alerts/:id` - Get alert detail.
  Response shape:
  ```json
  {
    "id": "string",
    "type": "budget",
    "title": "string",
    "module": "string",
    "description": "string",
    "severity": "warning",
    "created_at": "string",
    "action_path": "string"
  }
  ```
- *Note:* Alerts are computed dynamically. They compose budget overruns (>=80% warning, >=100% danger), overdue payable debts, and recent recurring-run activities (last 24 hours).

### Notifications
- `GET /api/v1/notifications/rules` - Get notification rules.
  Response shape:
  ```json
  {
    "rules": [
      {
        "id": "string",
        "user_id": "string",
        "rule_key": "budget-alert",
        "title": "string",
        "description": "string",
        "enabled": true,
        "channel": "email",
        "tone": "string",
        "created_at": "string",
        "updated_at": "string"
      }
    ]
  }
  ```
- `PUT /api/v1/notifications/rules/:id` - Update notification rule.
  Request body:
  ```json
  {
    "enabled": true,
    "channel": "email"
  }
  ```
  Response shape:
  ```json
  {
    "id": "string",
    "user_id": "string",
    "rule_key": "budget-alert",
    "title": "string",
    "description": "string",
    "enabled": true,
    "channel": "email",
    "tone": "string",
    "created_at": "string",
    "updated_at": "string"
  }
  ```
- *Note:* The system lazy-seeds 5 default rules for each user: `budget-alert`, `due-reminder`, `recurring-run`, `security-alert`, and `weekly-summary`.
- *Note:* Supported channels are `email`, `in-app`, and `both`.

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
