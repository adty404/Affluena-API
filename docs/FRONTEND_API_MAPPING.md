# Frontend API Mapping

This document maps the proposed frontend routes/pages (Stage 1-10) to the existing Affluena API endpoints.

| Frontend Route/Page | UI Action | Backend Endpoint | Method | Request Source | Response Data Needed | Loading State | Error State | Notes |
|---------------------|-----------|------------------|--------|----------------|----------------------|---------------|-------------|-------|
| **Foundation/Auth** | | | | | | | | |
| `/login` | Submit Login | `/api/v1/auth/login` | POST | form data | `tokens`, `user` | disable btn | show alert | - |
| `/register` | Submit Register | `/api/v1/auth/register` | POST | form data | `tokens`, `user` | disable btn | show alert | - |
| `/forgot-password` | Request reset | `/api/v1/auth/forgot-password` | POST | `{ email }` | - | disable btn | show alert | Returns 204 always (anti-enumeration) |
| `/reset-password` | Reset password | `/api/v1/auth/reset-password` | POST | `{ token, new_password }` | - | disable btn | show alert | Token from email link |
| `/onboarding` | Setup initial | Multiple | POST | form data | `wallet`, `category` | multi-step | inline error | Chain calls (e.g., Wallet -> Category) |
| **Dashboard** | | | | | | | | |
| `/dashboard` | View metrics | `/api/v1/dashboard/summary` | GET | `?month=` | stats, list | skeletons | retry btn | Combines multiple domains |
| `/dashboard/analytics` | View charts | `/api/v1/dashboard/expense-distribution` | GET | `?month=` | chart dataset | skeleton | empty state | - |
| `/dashboard/forecast` | View forecast | `/api/v1/dashboard/forecast` | GET | `?month=` | forecast dataset | skeleton | empty state | - |
| `/dashboard/widget-states` | - | - | - | - | - | - | - | **Frontend-only for now**: Save via localStorage |
| **Wallet** | | | | | | | | |
| `/wallets` | List wallets | `/api/v1/wallets` | GET | URL params | `resource array` | table skeleton| empty state | - |
| `/wallets/new` | Create wallet | `/api/v1/wallets` | POST | form data | `wallet` | disable btn | inline error | - |
| `/wallets/:id` | View wallet | `/api/v1/wallets/:id` | GET | path param | `wallet` | skeleton | 404 page | Transaction list queried separately |
| `/wallets/:id/edit` | Update wallet | `/api/v1/wallets/:id` | PUT | form data | `wallet` | disable btn | inline error | - |
| `/wallets/:id/sharing` | Manage shares | `/api/v1/wallets/:id/invites` | POST | form data | `wallet` | disable btn | inline error | Accept/reject via PATCH on member |
| **Category/Tag** | | | | | | | | |
| `/categories` | List categories | `/api/v1/categories` | GET | URL param | `resource array` | tree skeleton | empty state | Use `?type=` filter |
| `/categories/new` | Create category | `/api/v1/categories` | POST | form data | `category` | disable btn | inline error | Hierarchy max depth 3 |
| `/categories/:id/edit`| Update category | `/api/v1/categories/:id` | PUT | form data | `category` | disable btn | inline error | Parent must be same type |
| `/tags` | List tags | `/api/v1/tags` | GET | URL param | `resource array` | pill skeleton | empty state | - |
| `/tags/new` | Create tag | `/api/v1/tags` | POST | form data | `tag` | disable btn | inline error | - |
| `/tags/:id/edit` | Update tag | `/api/v1/tags/:id` | PUT | form data | `tag` | disable btn | inline error | - |
| **Transaction** | | | | | | | | |
| `/transactions` | List txns | `/api/v1/transactions` | GET | URL params | `resource array` | list skeleton | empty state | Includes `wallet_id`, `category_id` filters |
| `/transactions/new` | Create txn | `/api/v1/transactions` | POST | form data | `transaction` | disable btn | inline error | Validate `type` (income/expense) |
| `/transactions/:id` | View txn | `/api/v1/transactions/:id` | GET | path param | `transaction` | skeleton | 404 page | - |
| `/transactions/:id/edit`| Update txn | `/api/v1/transactions/:id` | PUT | form data | `transaction` | disable btn | inline error | - |
| `/transactions/transfer`| Create transfer | `/api/v1/transactions` | POST | form data | `transaction` | disable btn | inline error | Needs `to_wallet_id` |
| `/transactions/adjustment`| Create adjust | `/api/v1/transactions` | POST | form data | `transaction` | disable btn | inline error | Balance diffs calculated automatically |
| `/transactions/filter`| Apply filters | `/api/v1/transactions` | GET | URL params | `resource array` | list skeleton | empty state | `?from=x&to=y&category_id=z` |
| `/transactions/split` | Split bill | `/api/v1/transactions/split` | POST | form data | `transaction` | disable btn | inline error | Generates Debts automatically |
| `/quick-entry` | List QE | `/api/v1/quick-entry-templates` | GET | URL params | `resource array` | list skeleton | empty state | - |
| `/quick-entry/new` | Create QE | `/api/v1/quick-entry-templates` | POST | form data | `quick_entry` | disable btn | inline error | - |
| `/quick-entry/:id/edit`| Update QE | `/api/v1/quick-entry-templates/:id`| PUT | form data | `quick_entry` | disable btn | inline error | - |
| **Budget** | | | | | | | | |
| `/budgets` | List budgets | `/api/v1/category-budgets` | GET | URL params | `resource array` | list skeleton | empty state | - |
| `/budgets/new` | Create budget | `/api/v1/category-budgets` | POST | form data | `budget` | disable btn | inline error | - |
| `/budgets/:id` | View budget | `/api/v1/category-budgets/:id` | GET | path param | `budget` | skeleton | 404 page | - |
| `/budgets/:id/edit` | Update budget | `/api/v1/category-budgets/:id` | PUT | form data | `budget` | disable btn | inline error | - |
| `/budgets/alerts` | List alerts | `/api/v1/category-budgets/alerts` | GET | `?month=` | `{ alerts: BudgetAlert[] }` | list skeleton | empty state | Get budget alerts for a specific month |
| `/budgets/report` | View report | `/api/v1/category-budgets/report` | GET | `?month=` | `{ report: BudgetReportItem[], summary: BudgetReportSummary }` | skeleton | empty state | Get budget report for a specific month |
| **Debt & Tracker** | | | | | | | | |
| `/tracker` | View tracker | Multiple | GET | URL params | `resource array` | list skeleton | empty state | Fetches installments and subscriptions |
| `/debts` | List debts | `/api/v1/debts` | GET | URL params | `resource array` | list skeleton | empty state | - |
| `/debts/new/payable` | Create payable | `/api/v1/debts` | POST | form data | `debt` | disable btn | inline error | `type="payable"` |
| `/debts/new/receivable`| Create receive | `/api/v1/debts` | POST | form data | `debt` | disable btn | inline error | `type="receivable"` |
| `/debts/:id` | View debt | `/api/v1/debts/:id` | GET | path param | `debt` | skeleton | 404 page | - |
| `/debts/:id/pay` | Pay debt | `/api/v1/debts/:id/pay` | POST | form data | `transaction` | disable btn | inline error | - |
| `/installments` | List install | `/api/v1/installments` | GET | URL params | `resource array` | list skeleton | empty state | - |
| `/installments/new` | Create install | `/api/v1/installments` | POST | form data | `installment` | disable btn | inline error | - |
| `/installments/:id/pay`| Pay install | `/api/v1/installments/:id/pay`| POST | form data | `transaction` | disable btn | inline error | - |
| `/subscriptions` | List sub | `/api/v1/subscriptions` | GET | URL params | `resource array` | list skeleton | empty state | - |
| `/subscriptions/new` | Create sub | `/api/v1/subscriptions` | POST | form data | `subscription`| disable btn | inline error | - |
| `/subscriptions/:id/pay`| Pay sub | `/api/v1/subscriptions/:id/pay`| POST | form data | `transaction` | disable btn | inline error | - |
| **Recurring** | | | | | | | | |
| `/recurring` | List recur | `/api/v1/recurring-transactions` | GET | URL params | `resource array` | list skeleton | empty state | - |
| `/recurring/new` | Create recur | `/api/v1/recurring-transactions` | POST | form data | `recurring` | disable btn | inline error | - |
| `/recurring/:id` | View recur | `/api/v1/recurring-transactions/:id` | GET | path param | `recurring` | skeleton | 404 page | - |
| `/recurring/:id/edit` | Update recur | `/api/v1/recurring-transactions/:id` | PUT | form data | `recurring` | disable btn | inline error | - |
| `/recurring/:id/run` | Manual run | `/api/v1/recurring-transactions/:id/run`| POST | empty body | `transaction` | disable btn | inline error | - |
| `/recurring/:id/history`| View history | `/api/v1/transactions` | GET | query param | `resource array` | list skeleton | empty state | **Available via existing endpoint**: Filter txns by `source=recurring` (needs frontend correlation) |
| **Goals** | | | | | | | | |
| `/goals` | List goals | `/api/v1/goals` | GET | none | `[...]` array | grid skeleton | empty state | Array, no pagination metadata |
| `/goals/new` | Create goal | `/api/v1/goals` | POST | form data | `goal` | disable btn | inline error | - |
| `/goals/:id` | View goal | `/api/v1/goals/:id` | GET | path param | `goal` | skeleton | 404 page | - |
| `/goals/:id/edit` | Update goal | `/api/v1/goals/:id` | PUT | form data | `goal` | disable btn | inline error | - |
| `/goals/:id/contribute`| Contribute | `/api/v1/transactions` | POST | form data | `transaction` | disable btn | inline error | Transfer to goal wallet. |
| `/goals/:id/members` | Manage members | `/api/v1/goals/:id/members` | POST | form data | `goal` | disable btn | inline error | Accept via PUT `.../respond` |
| **Reports/System** | | | | | | | | |
| `/reports` | Overview | Multiple Dashboard APIs | GET | URL params | datasets | skeletons | empty state | Uses dashboard/summary endpoints |
| `/reports/income` | View income report | `/api/v1/reports/income` | GET | `?month=` | `{ metrics, rows }` | skeleton | empty state | Get income report |
| `/reports/expense` | View expense report | `/api/v1/reports/expense` | GET | `?month=` | `{ metrics, rows }` | skeleton | empty state | Get expense report |
| `/reports/cashflow` | View cashflow report | `/api/v1/reports/cashflow` | GET | `?month=` | `{ metrics, rows }` | skeleton | empty state | Get cashflow report |
| `/reports/debt` | View debt report | `/api/v1/reports/debt` | GET | `?month=` | `{ metrics, rows }` | skeleton | empty state | Get debt report |
| `/reports/goal` | View goal report | `/api/v1/reports/goal` | GET | `?month=` | `{ metrics, rows }` | skeleton | empty state | Get goal report |
| `/reports/overview` | View overview report | `/api/v1/reports/overview` | GET | `?month=` | `{ metrics, rows }` | skeleton | empty state | Get overview report |
| `/exports` | Trigger export | `/api/v1/export/csv` | GET | `?from=&to=` | CSV stream | disable btn | error alert | Direct download |
| `/exports/history` | List export jobs | `/api/v1/export/jobs` | GET | `?limit=&offset=` | `{ jobs: ExportJob[], pagination }` | list skeleton | empty state | View export jobs history |
| `/exports/:id` | View export job | `/api/v1/export/jobs/:id` | GET | path param | `ExportJob` | skeleton | 404 page | Get export job detail |
| `/activities` | List activities| `/api/v1/activities` | GET | URL params | `resource array` | list skeleton | empty state | - |
| `/activities/:id` | View activity detail | `/api/v1/activities/:id` | GET | path param | `Activity` | skeleton | 404 page | Get activity detail |
| `/alerts` | Inbox alerts | `/api/v1/alerts` | GET | `?month=` | `{ alerts: Alert[] }` | list skeleton | empty state | Get alerts feed for a specific month |
| `/system-logs` | List system logs | `/api/v1/system-logs` | GET | `?limit=` | `{ logs: SystemLog[] }` | list skeleton | empty state | View system logs |
| **Settings** | | | | | | | | |
| `/settings/profile` | View profile | `/api/v1/auth/me` | GET | none | `user` | skeleton | error alert | - |
| `/settings/account` | Update account | `/api/v1/auth/account` | PUT | `{ name, avatar_url }` | `user` | disable btn | inline error | - |
| `/settings/security` | Update password| `/api/v1/auth/password` | PUT | `{ current_password, new_password }` | - | disable btn | inline error | - |
| `/settings/sessions` | List sessions | `/api/v1/auth/sessions` | GET | none | `sessions` | list skeleton | empty state | Revoke via `DELETE /auth/sessions/:id` |
| `/settings/notifications`| Manage notification rules | `/api/v1/notifications/rules` | GET/PUT | `{ enabled, channel }` | `{ rules: NotificationRule[] }` or `NotificationRule` | skeleton | inline error | Manage notification rules |
| `/settings/preferences`| - | - | - | - | - | - | - | **Frontend-only**: Store locally |
