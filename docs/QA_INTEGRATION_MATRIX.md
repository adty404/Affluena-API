# QA Integration Matrix

This document provides a testing checklist for QA and Frontend engineers to ensure that the Affluena UI adheres to the backend's strict business rules.

Current automated E2E coverage lives in `Affluena-QA`: 85 Playwright spec files with 198 runnable tests across foundation, wallets, categories, tags, transactions, dashboard, budgets, debts, trackers, recurring, goals, reports, export, activity, alerts, system logs, settings, and security. Use this matrix as the business-rule checklist; use `Affluena-QA/tests/**` as the executable suite.

## Foundation & Auth
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Register** | Submit new valid user (email & password) | 201 Created. User logged in automatically. | Duplicate email throws 409. |
| **Login** | Valid credentials | 200 OK. Returns access and refresh tokens. | Wrong password throws 401. |
| **Protected Route**| Access without token | 401 Unauthorized. | Missing Bearer header. |
| **Token Refresh**| Refresh using valid refresh token| 200 OK. New token pair generated. | Expired refresh token returns 401. |
| **Change Password**| `PUT /auth/password` with valid current password | 200 OK. Returns `{ user, tokens }`. **All other sessions revoked**; client persists the new pair. | Old/other refresh tokens now 401. Wrong current → 401. Weak new → 400. |
| **Reset Password**| `POST /auth/reset-password` with valid token | 204. **All sessions revoked** (re-login required everywhere). | Invalid/expired token → 400. |

## Security (OWASP)
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Security Headers**| Any API response | `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, CSP `default-src 'none'`. | HSTS present only in production. |
| **Auth Rate Limit**| Burst of auth requests from one IP | `429 Too Many Requests` once burst exceeded. | Default 5 req/s, burst 10 per IP. |
| **CSV Injection**| Export note/wallet/category starting with `=`/`+`/`-`/`@` | Cell prefixed with `'` (inert text). | Defends spreadsheet formula execution. |
| **Session Revocation**| Change password, then refresh with an old token | Old + other-device refresh tokens return 401; only the freshly issued pair works. | - |

## Wallet & Permissions
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Create Wallet**| Submit valid wallet data | 201 Created. Initial balance is 0. | `type="goal"` is explicitly rejected (400). |
| **Update Wallet**| Edit name or type | 200 OK. | Balance cannot be modified directly via PUT. |
| **Delete Wallet**| Delete empty wallet | 204 No Content. Wallet is hard-deleted. | Non-empty wallet deletion may fail depending on constraint logic. |
| **Share Wallet** | Invite another user via email | 201 Created. Invite created. | Self-invite rejected. Non-existent user rejected. |
| **Accept Share** | Member accepts invite | 200 OK. Member can now see and use wallet. | - |
| **Cross-User Auth**| Try accessing another user's wallet| 403 Forbidden or 404 Not Found. | IDs manipulated in URL/Request payload. |

## Transactions
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Create Income**| Submit `transaction_at`, `note` to Wallet A | Wallet A balance increases by amount. | `transaction_at` is a full RFC3339 timestamp (date AND time-of-day). Backdated and future-dated values accepted. |
| **Create Expense**| Submit `transaction_at`, `note` from Wallet A | Wallet A balance decreases by amount. | Same `transaction_at` rules: RFC3339 date+time, backdate/future allowed. Edit (PUT) accepts the same. |
| **Transfer** | Transfer A -> B | Wallet A decreases, Wallet B increases. | Requires `to_wallet_id`. Must own both wallets. `transaction_at` is RFC3339 date+time; backdate/future allowed. |
| **Adjustment** | Adjust Wallet A | Calculates difference and creates adjustment transaction. | Adjustment transaction stamps `transaction_at` (RFC3339 date+time); backdate/future allowed via the picker. |
| **Split Bill** | Split expense | Debt generated automatically. `origination_transaction_id` is an internal flow. | Split total must equal transaction amount. `transaction_at` is RFC3339 date+time; backdate/future allowed. |
| **Data Isolation**| Submit expense to unowned wallet| 403 Forbidden. | - |

> **`transaction_at` is a full RFC3339 timestamp (date AND time-of-day).** Every transaction entry point — create/edit, transfer, adjustment, split-bill, and quick-entry execute — accepts any date, so backdating and future-dating are supported and the picked time-of-day is preserved. Clients send the local datetime normalized to UTC (e.g. `DateTime.toUtc().toIso8601String()`); the API parses it as RFC3339. Values are not coerced to date-only or midnight.

## Budget
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Create Budget**| Submit budget for Category A | 201 Created. | Duplicate category budget in same month rejected. |
| **Expense vs Budget**| Submit personal expense in Category A| Dashboard/Budget view shows budget utilized. | - |
| **Shared Wallet Budget**| Shared member submits expense in Category A| Owner's personal budget is NOT decremented. | Budgets are isolated to personal categories. |
| **Alerts** | Expense exceeds budget limit | Alert is visible through budget alerts and the unified alerts feed. | - |

## Debt
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Create Payable**| Submit payable debt | 201 Created. | Requires valid categories. |
| **Pay Debt** | Partial payment | Remaining amount decreases. Transaction created. | Payment amount > remaining amount rejected. |
| **Cancel Debt**| Cancel an unpaid debt | Debt marked as soft-cancelled/deleted. | Cancel fails if any payment has already been made. |

## Goals
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Create Goal**| Submit new goal | 201 Created. Goal wallet generated automatically. | - |
| **Invite Member**| Owner invites member | Member added to goal and goal wallet. | - |
| **Contribute** | Transfer from personal to Goal wallet | Goal progress increases. | Unauthorized transfer to goal wallet rejected. |

## Recurring Transactions
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Create Rule**| Submit schedule | Rule active. Next run calculated. | - |
| **Manual Run** | Execute rule immediately | Transaction created. Balances updated. | - |
| **Pause Rule** | Pause a running schedule | Rule status becomes paused. | - |

## Reports & Export
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Export CSV** | Valid date range | Returns CSV file stream. | Invalid dates 400. |
| **Export Isolation**| - | Only contains data accessible to authenticated user. | - |

## Tags & Trackers
| Flow | Action | Expected Result | Edge Cases |
|------|--------|-----------------|------------|
| **Create Tag** | Submit tag data | 201 Created. | Request body must only contain `name` (no color). |
| **Pay Installment** | Submit `paid_at` and `note` | 201 Created. Amount is internally configured. | Do not send `amount_minor` from frontend. |
| **Pay Subscription** | Submit `paid_at` and `note` | 201 Created. Amount is internally configured. | Do not send `amount_minor` from frontend. |
