# Debt & Loan Manager Design

> **Current status (2026-06-20):** Historical design artifact. Debt, payable/receivable payment lifecycle, soft-cancel behavior, shared-wallet debt support, and integration tests are implemented. Use `../../API_CONTRACT.md`, `../../system_map.md`, and `internal/debt` as the current source of truth.

## Scope

Implement an API-first Debt & Loan Manager for both money owed to the user and money owed by the user.

- `receivable`: the user lends money to someone else. Creating the record decreases the selected wallet. Repayment increases the wallet.
- `payable`: the user borrows money from someone else. Creating the record increases the selected wallet. Repayment decreases the wallet.

The feature covers tracking counterparties, due dates, paid amount, remaining amount, status, and payment history. It does not add interest, collateral, reminders, or recurring repayment schedules.

## API

Protected endpoints:

- `POST /api/v1/debts`
- `GET /api/v1/debts`
- `GET /api/v1/debts/:id`
- `PUT /api/v1/debts/:id`
- `DELETE /api/v1/debts/:id`
- `POST /api/v1/debts/:id/pay`

Create request:

- `type`: `receivable` or `payable`
- `counterparty_name`
- `wallet_id`
- `disbursement_category_id`
- `payment_category_id`
- `principal_amount_minor`
- `opened_at` optional RFC3339 timestamp; defaults to now
- `due_date` optional `YYYY-MM-DD`
- `note`

Payment request:

- `amount_minor`
- `paid_at` optional RFC3339 timestamp; defaults to now
- `note`

Update request changes tracking metadata only: `counterparty_name`, `due_date`, `status`, and `note`. It does not rewrite the original principal transaction because transaction history should remain explicit and auditable.

`DELETE` soft-cancels the debt record. It does not remove or reverse transaction history.

## Data Model

`debts` stores the tracked debt or loan:

- owner user id
- type, counterparty name, wallet id
- disbursement and payment category ids
- origination transaction id
- principal amount, paid amount
- optional due date
- status: `open`, `partial`, `paid_off`, `cancelled`
- note and timestamps

`debt_payments` stores each repayment:

- owner user id
- debt id
- generated transaction id
- amount, paid timestamp, note
- created timestamp

Remaining amount is computed as `principal_amount_minor - paid_amount_minor`.

## Transaction Behavior

All balance-changing actions use PostgreSQL transactions.

- Create `receivable`: create an expense transaction using `disbursement_category_id`.
- Pay `receivable`: create an income transaction using `payment_category_id`.
- Create `payable`: create an income transaction using `disbursement_category_id`.
- Pay `payable`: create an expense transaction using `payment_category_id`.

Payment rejects cancelled, already paid off, non-positive, or overpayment attempts.

## Testing

- Domain tests cover payment math, status transitions, overpayment rejection, and transaction type mapping.
- Full project tests, vet, and build verify compile-time and static correctness.
- Docker smoke test covers create/pay flows for both `receivable` and `payable`, including wallet balance changes.
