# Frontend Integration Plan

This document outlines the recommended sequencing for integrating the frontend UI (Stage 1-10) with the Affluena API backend.

## Phase 1: Core Foundation

### 1. Auth & API Client
- **Prerequisite Backend**: Auth module running.
- **Prerequisite Frontend**: Axios/Fetch client with interceptors setup.
- **Endpoints**: `/api/v1/auth/login`, `/api/v1/auth/register`, `/api/v1/auth/me`.
- **Expected UI Behavior**: User can log in, receive JWT, and access protected pages. Automatic redirection on 401.
- **QA Test**: Register success, Login success, Invalid login, Expired token handling.
- **Done Criteria**: `Bearer` token reliably injected into all subsequent protected API requests.
- **Status**: ✅ Done

### 2. Wallet Management
- **Prerequisite Backend**: Wallet endpoints.
- **Prerequisite Frontend**: Auth context is active.
- **Endpoints**: `/api/v1/wallets` (CRUD), `/api/v1/wallets/:id/invites`.
- **Expected UI Behavior**: User can create their first wallet, view wallet balances (which start at 0), and see shared wallets.
- **QA Test**: Create wallet, Delete wallet, Share wallet, Accept member invite.
- **Done Criteria**: Wallets persist and populate the global state for dropdowns in forms.
- **Status**: ✅ Done

### 3. Categories & Tags
- **Prerequisite Backend**: Category and Tag endpoints.
- **Prerequisite Frontend**: Auth context.
- **Endpoints**: `/api/v1/categories`, `/api/v1/tags`.
- **Expected UI Behavior**: User can define hierarchical categories (max 3 levels) and tags for future transactions.
- **QA Test**: Create income/expense categories, cyclic parent prevention.
- **Done Criteria**: Category trees render correctly in select inputs.
- **Status**: ✅ Done

## Phase 2: The Core Loop

### 4. Transactions
- **Prerequisite Backend**: Wallet, Category, Tag, Transaction endpoints.
- **Prerequisite Frontend**: Forms for creating transactions.
- **Endpoints**: `/api/v1/transactions` (CRUD, Split Bill), `/api/v1/quick-entry-templates`.
- **Expected UI Behavior**: Creating an income/expense updates the wallet balance immediately (optimistic UI or refetch).
- **QA Test**: Create income, transfer, split bill. Check balance changes.
- **Done Criteria**: Transactions show up in list with correct category labels and wallet balances match.
- **Status**: ✅ Done

### 5. Dashboard
- **Prerequisite Backend**: Dashboard analytics.
- **Prerequisite Frontend**: Transaction flow is working.
- **Endpoints**: `/api/v1/dashboard/summary`, `/api/v1/dashboard/expense-distribution`.
- **Expected UI Behavior**: Charts render realistic data. Spending affects cashflow.
- **QA Test**: Ensure shared wallet member expenses affect the owner's dashboard if included in accessible-wallet views.
- **Done Criteria**: Dashboard accurately reflects data created in step 4.
- **Status**: ✅ Done

## Phase 3: Advanced Financial Tools

### 6. Budget
- **Prerequisite Backend**: Category budgets.
- **Prerequisite Frontend**: Category module.
- **Endpoints**: `/api/v1/category-budgets`.
- **Expected UI Behavior**: Budget progress bars fill up as expenses are added.
- **QA Test**: Create budget, add expense, check budget threshold alert.
- **Done Criteria**: Budget vs Actual logic works accurately on personal budgets.
- **Status**: ✅ Done

### 7. Debt & Tracker
- **Prerequisite Backend**: Debt, Tracker (Installments/Subscriptions).
- **Prerequisite Frontend**: Transaction logic.
- **Endpoints**: `/api/v1/debts`, `/api/v1/installments`, `/api/v1/subscriptions`, and their `/pay` endpoints.
- **Expected UI Behavior**: Debts reflect remaining amounts. Paying a debt automatically creates a transaction.
- **QA Test**: Create payable, pay debt, attempt to soft-cancel debt.
- **Done Criteria**: Financial trackers stay in sync with transaction history.
- **Status**: ✅ Done

### 8. Goals
- **Prerequisite Backend**: Goal endpoints.
- **Prerequisite Frontend**: Transaction logic (for contributions).
- **Endpoints**: `/api/v1/goals` and members endpoints.
- **Expected UI Behavior**: Goals create internal wallets. Contributions are tracked as transfers.
- **QA Test**: Create goal, invite member, contribute via transfer, unauthorized contribution denied.
- **Done Criteria**: Goal progress calculates correctly based on wallet balance.
- **Status**: ✅ Done

### 9. Recurring Transactions
- **Prerequisite Backend**: Recurring endpoints, backend scheduler running.
- **Prerequisite Frontend**: Form to define chron rules.
- **Endpoints**: `/api/v1/recurring-transactions`.
- **Expected UI Behavior**: Users can schedule future transactions and manually force run them.
- **QA Test**: Create rule, manual run, pause rule.
- **Done Criteria**: Manual runs produce actual transactions.
- **Status**: ✅ Done

## Phase 4: Polish & Reporting

### 10. Reports & Export
- **Prerequisite Backend**: Export CSV endpoints.
- **Prerequisite Frontend**: Date pickers.
- **Endpoints**: `/api/v1/export/csv`.
- **Expected UI Behavior**: Browser triggers file download on export.
- **QA Test**: Export CSV with date filters.
- **Done Criteria**: Downloaded CSV is valid and contains user data.
- **Status**: ✅ Done

### 11. Activity & Alerts
- **Prerequisite Backend**: Activity module.
- **Prerequisite Frontend**: Notification bell icon.
- **Endpoints**: `/api/v1/activities`.
- **Expected UI Behavior**: Alerts for over-budget show up in activity stream.
- **QA Test**: Check recent activity logs.
- **Done Criteria**: Real-time or polling updates the unread count.
- **Status**: ✅ Done

### 12. Settings / Profile
- **Prerequisite Backend**: Auth/Me.
- **Prerequisite Frontend**: Global state.
- **Endpoints**: `/api/v1/auth/me`.
- **Expected UI Behavior**: Profile page shows basic info. App settings (theme, etc.) stored in localStorage.
- **Done Criteria**: Settings persist across reloads.
- **Status**: ✅ Done

## Phase 5: Advanced Reporting & Notifications

### 13. System Logs & Activity Detail
- **Prerequisite Backend**: System logs and activity detail endpoints.
- **Prerequisite Frontend**: SystemLogList, SystemLogDetail, ActivityDetail pages.
- **Endpoints**: `GET /api/v1/system-logs`, `GET /api/v1/system-logs/:id`, `GET /api/v1/activities/:id`.
- **Expected UI Behavior**: Admins can view system logs and details, and users can view activity details.
- **Status**: ✅ Done

### 14. Budget Alerts & Report
- **Prerequisite Backend**: Budget alerts and report endpoints.
- **Prerequisite Frontend**: Budget alerts feed and report pages.
- **Endpoints**: `GET /api/v1/category-budgets/alerts`, `GET /api/v1/category-budgets/report`.
- **Expected UI Behavior**: Users can view budget alerts and monthly budget reports.
- **Status**: ✅ Done

### 15. Alerts Feed
- **Prerequisite Backend**: Alerts feed endpoints.
- **Prerequisite Frontend**: Alerts feed page.
- **Endpoints**: `GET /api/v1/alerts`, `GET /api/v1/alerts/:id`.
- **Expected UI Behavior**: Users can view a unified alerts feed.
- **Status**: ✅ Done

### 16. Export Jobs Audit
- **Prerequisite Backend**: Export jobs endpoints.
- **Prerequisite Frontend**: Export history page.
- **Endpoints**: `GET /api/v1/export/jobs`, `GET /api/v1/export/jobs/:id`.
- **Expected UI Behavior**: Users can view their export history and job details.
- **Status**: ✅ Done

### 17. Reports Aggregation
- **Prerequisite Backend**: Reports aggregation endpoints.
- **Prerequisite Frontend**: Reports pages.
- **Endpoints**: `GET /api/v1/reports/income`, `GET /api/v1/reports/expense`, `GET /api/v1/reports/cashflow`, `GET /api/v1/reports/debt`, `GET /api/v1/reports/goal`, `GET /api/v1/reports/overview`.
- **Expected UI Behavior**: Users can view detailed financial reports.
- **Status**: ✅ Done

### 18. Notification Rules
- **Prerequisite Backend**: Notification rules endpoints.
- **Prerequisite Frontend**: Notification settings page.
- **Endpoints**: `GET /api/v1/notifications/rules`, `PUT /api/v1/notifications/rules/:id`.
- **Expected UI Behavior**: Users can manage their notification preferences.
- **Status**: ✅ Done

### 19. Mock Data Cleanup
- **Prerequisite Backend**: None.
- **Prerequisite Frontend**: Clean up mock files.
- **Expected UI Behavior**: All pages use real API data.
- **Status**: ✅ Done

## Summary

All 19 integration steps complete. 17 new endpoints. 186 QA tests pass. Zero mock data remaining (except 3 legit static files).
