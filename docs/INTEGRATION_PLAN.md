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

### 2. Wallet Management
- **Prerequisite Backend**: Wallet endpoints.
- **Prerequisite Frontend**: Auth context is active.
- **Endpoints**: `/api/v1/wallets` (CRUD), `/api/v1/wallets/:id/invites`.
- **Expected UI Behavior**: User can create their first wallet, view wallet balances (which start at 0), and see shared wallets.
- **QA Test**: Create wallet, Delete wallet, Share wallet, Accept member invite.
- **Done Criteria**: Wallets persist and populate the global state for dropdowns in forms.

### 3. Categories & Tags
- **Prerequisite Backend**: Category and Tag endpoints.
- **Prerequisite Frontend**: Auth context.
- **Endpoints**: `/api/v1/categories`, `/api/v1/tags`.
- **Expected UI Behavior**: User can define hierarchical categories (max 3 levels) and tags for future transactions.
- **QA Test**: Create income/expense categories, cyclic parent prevention.
- **Done Criteria**: Category trees render correctly in select inputs.

## Phase 2: The Core Loop

### 4. Transactions
- **Prerequisite Backend**: Wallet, Category, Tag, Transaction endpoints.
- **Prerequisite Frontend**: Forms for creating transactions.
- **Endpoints**: `/api/v1/transactions` (CRUD, Split Bill), `/api/v1/quick-entry-templates`.
- **Expected UI Behavior**: Creating an income/expense updates the wallet balance immediately (optimistic UI or refetch).
- **QA Test**: Create income, transfer, split bill. Check balance changes.
- **Done Criteria**: Transactions show up in list with correct category labels and wallet balances match.

### 5. Dashboard
- **Prerequisite Backend**: Dashboard analytics.
- **Prerequisite Frontend**: Transaction flow is working.
- **Endpoints**: `/api/v1/dashboard/summary`, `/api/v1/dashboard/expense-distribution`.
- **Expected UI Behavior**: Charts render realistic data. Spending affects cashflow.
- **QA Test**: Ensure shared wallet member expenses affect the owner's dashboard if included in accessible-wallet views.
- **Done Criteria**: Dashboard accurately reflects data created in step 4.

## Phase 3: Advanced Financial Tools

### 6. Budget
- **Prerequisite Backend**: Category budgets.
- **Prerequisite Frontend**: Category module.
- **Endpoints**: `/api/v1/category-budgets`.
- **Expected UI Behavior**: Budget progress bars fill up as expenses are added.
- **QA Test**: Create budget, add expense, check budget threshold alert.
- **Done Criteria**: Budget vs Actual logic works accurately on personal budgets.

### 7. Debt & Tracker
- **Prerequisite Backend**: Debt, Tracker (Installments/Subscriptions).
- **Prerequisite Frontend**: Transaction logic.
- **Endpoints**: `/api/v1/debts`, `/api/v1/installments`, `/api/v1/subscriptions`, and their `/pay` endpoints.
- **Expected UI Behavior**: Debts reflect remaining amounts. Paying a debt automatically creates a transaction.
- **QA Test**: Create payable, pay debt, attempt to soft-cancel debt.
- **Done Criteria**: Financial trackers stay in sync with transaction history.

### 8. Goals
- **Prerequisite Backend**: Goal endpoints.
- **Prerequisite Frontend**: Transaction logic (for contributions).
- **Endpoints**: `/api/v1/goals` and members endpoints.
- **Expected UI Behavior**: Goals create internal wallets. Contributions are tracked as transfers.
- **QA Test**: Create goal, invite member, contribute via transfer, unauthorized contribution denied.
- **Done Criteria**: Goal progress calculates correctly based on wallet balance.

### 9. Recurring Transactions
- **Prerequisite Backend**: Recurring endpoints, backend scheduler running.
- **Prerequisite Frontend**: Form to define chron rules.
- **Endpoints**: `/api/v1/recurring-transactions`.
- **Expected UI Behavior**: Users can schedule future transactions and manually force run them.
- **QA Test**: Create rule, manual run, pause rule.
- **Done Criteria**: Manual runs produce actual transactions.

## Phase 4: Polish & Reporting

### 10. Reports & Export
- **Prerequisite Backend**: Export CSV endpoints.
- **Prerequisite Frontend**: Date pickers.
- **Endpoints**: `/api/v1/export/csv`.
- **Expected UI Behavior**: Browser triggers file download on export.
- **QA Test**: Export CSV with date filters.
- **Done Criteria**: Downloaded CSV is valid and contains user data.

### 11. Activity & Alerts
- **Prerequisite Backend**: Activity module.
- **Prerequisite Frontend**: Notification bell icon.
- **Endpoints**: `/api/v1/activities`.
- **Expected UI Behavior**: Alerts for over-budget show up in activity stream.
- **QA Test**: Check recent activity logs.
- **Done Criteria**: Real-time or polling updates the unread count.

### 12. Settings / Profile
- **Prerequisite Backend**: Auth/Me.
- **Prerequisite Frontend**: Global state.
- **Endpoints**: `/api/v1/auth/me`.
- **Expected UI Behavior**: Profile page shows basic info. App settings (theme, etc.) stored in localStorage.
- **Done Criteria**: Settings persist across reloads.
