# Seed Data for Frontend Integration

Currently, there is no automated Go or SQL seed script integrated into the Affluena API environment setup. The recommended approach for staging and frontend integration is to use the provided Postman collection to quickly bootstrap testing data.

## Bootstrap Recommendations

Until a `scripts/seed-dev.go` is implemented, follow these steps to seed an empty database using Postman:

1. **Create Dev User**:
   - Open the Postman collection.
   - Run **Auth > Register** with:
     - Email: `dev@example.com`
     - Password: `password123`
   - Run **Auth > Login** to populate your Bearer token variable.

2. **Create Core Wallets**:
   - Run **Wallets > Create Wallet** multiple times for:
     - `Cash` (Type: `cash`, Balance: `0`)
     - `Bank BCA` (Type: `bank`, Balance: `0`)
     - `GoPay` (Type: `e_wallet`, Balance: `0`)

3. **Create Categories**:
   - Run **Categories > Create Category**:
     - Income: `Salary`, `Bonus`
     - Expense: `Food`, `Transport`, `Entertainment`
     - Nested (Optional): Add `Coffee` with `parent_id` set to `Food`.

4. **Create Tags**:
   - Run **Tags > Create Tag**: `Urgent`, `Subscription`, `Holiday`.

5. **Generate Transactions (The Core Loop)**:
   - Run **Transactions > Create Transaction** to inject funds:
     - Type: `income`, Wallet: `Bank BCA`, Category: `Salary`, Amount: `10000000`.
   - Run **Transactions > Create Transaction** for expenses:
     - Type: `expense`, Wallet: `Bank BCA`, Category: `Food`, Amount: `50000`.
   - Run **Transactions > Create Transaction** for transfers:
     - Type: `transfer`, Wallet: `Bank BCA`, To Wallet: `GoPay`, Amount: `500000`.

6. **Generate Advanced Entities**:
   - Use the remaining endpoints in Postman (Budgets, Debts, Installments, Recurring, Goals) utilizing the UUIDs generated from steps 2-5.

## Expected Initial State
After completing the above, the `/api/v1/dashboard/summary` should reflect:
- Total Net Worth: `9,950,000`
- Cashflow: `+9,950,000` (Income 10m - Expense 50k)
- 3 Wallets with varying balances.

## Shared Wallet Scenario
If testing shared wallets:
1. Register a second user `dev2@example.com`.
2. Login as `dev@example.com`.
3. Invite `dev2`'s email via `POST /api/v1/wallets/:id/invites`.
4. Login as `dev2@example.com` and accept the invite via `PATCH /api/v1/wallets/:id/members/:member_id`.
5. Transactions made by `dev2` on the shared wallet will affect `dev`'s wallet balances, but will NOT affect `dev`'s personal category budgets.
