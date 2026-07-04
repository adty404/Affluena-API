# Seed Data for Frontend and QA Integration

Affluena-API now includes an idempotent Go seed command for local frontend review and QA exploration.

## Automated Demo Seed

Start PostgreSQL/API dependencies, then run:

```bash
docker compose up -d
make seed
```

`make seed` builds and runs `cmd/seed/main.go`. The command uses `DATABASE_URL`, then `AFFLUENA_API_TEST_DATABASE_URL`, and finally falls back to:

```text
postgres://affluena_api:affluena_api@localhost:5432/affluena_api?sslmode=disable
```

The seed deletes any existing demo user with the same email, then recreates a fresh data set.

Login credentials:

```text
Email:    demo@affluena.com
Password: password123
```

Seeded data:

- 5 wallets: Cash Wallet (cash), BCA Primary (bank), GoPay (e-wallet), Jenius (bank), OVO (e-wallet).
- 9 categories: 3 income and 6 expense.
- 2 tags.
- **~7 transactions per day for the whole previous complete calendar month (~210–230 total)** — a densely lived-in month so Kalender, Wawasan, and history screens feel real. The seeded month is the calendar month *before* the seed run (e.g. seeding on 2026-07-04 fills **June 2026**), so the history is always fully past — never future-dated. The daily expenses are **generated deterministically** in Go (a per-day loop over pools of food/transport/shopping/entertainment picks, no randomness), so every re-seed produces identical content (only the row UUIDs differ) — idempotent by the delete-then-recreate. Each day gets 3 food spends + 1 transport (+ a weekly fuel fill-up and roughly-twice-weekly grocery run) + a rotating shopping/entertainment spend, topped up to 7; fixed monthly bills land on days 3/10/15/25. Income is added on top (salary on day 1 + freelance on a few days), plus 3 inter-wallet topup transfers and 1 debt-disbursement expense. Dates anchor to the first of that month plus a day/time offset, built in **WIB (UTC+7)** so late-evening rows never spill into the next month for the WIB reader (`time.Date(monthStart.Year(), monthStart.Month(), day, ..., wib)`), covering every day. Expense mix keeps **Food & Dining the largest bucket (~55–60% by value, the majority of rows)**, then Transportation, Shopping, Entertainment, Bills; transactions spread across GoPay / Cash / OVO / BCA. Amounts are realistic IDR whole-rupiah minor units. Wallet `balance_minor` values are fixed, illustrative opening balances — the seeder does **not** recompute them from the transaction ledger (matching the long-standing seed convention), so there is no balance/ledger invariant to maintain.
- 4 category budgets for the seeded month.
- 1 payable debt.
- 1 subscription.
- 1 installment.
- 1 recurring rule.
- 1 financial goal with a goal wallet.
- 1 quick entry template.
- 2 extra accounts for the "Berbagi Dompet" demo (a connected viewer + a pending invite) — see [Berbagi Dompet demo](#berbagi-dompet-demo).
- Demo `color`/`icon` appearance values on wallets, categories, budgets, the goal, the installment, the subscription, and the recurring rule — see [Appearance values](#appearance-values-color--icon).

## Appearance values (color & icon)

The seed showcases the appearance feature: every module that supports `color`/`icon` (wallets, categories, category budgets, goals, installments, subscriptions, recurring rules) gets values from the shared client catalogs — the 10-swatch color palette (stored uppercase `#RRGGBB`) plus `kWalletIconCatalog` ids for wallets and `kCategoryIconCatalog` ids for categories and the entity modules. **Every category carries both a color and an icon** — categories surface in transaction history across every screen, so a colorless one would look unfinished. A few *other-module* rows (a wallet, a budget) are deliberately left **colorless** so clients can still verify their fallback rendering.

| Module | Item | Color | Icon |
|--------|------|-------|------|
| Wallet | Cash Wallet | — (fallback) | `cash` |
| Wallet | BCA Primary | `#3E72B8` denim | `bank` |
| Wallet | GoPay | `#2BB3A3` teal | `ewallet` |
| Wallet | Jenius | `#4256B8` indigo | `bank` |
| Wallet | OVO | `#7C5BC2` purple | `ewallet` |
| Wallet (goal) | Europe Trip Fund | `#3E72B8` denim | `travel` |
| Category | Salary | `#2E8B57` green | `salary` |
| Category | Freelance | `#9E7B4F` bronze | `work` |
| Category | Loan Repayment | `#2BB3A3` teal | `savings` |
| Category | Food & Dining | `#C2553F` coral | `food` |
| Category | Transportation | `#E0A23B` amber | `transport` |
| Category | Entertainment | `#7C5BC2` purple | `entertainment` |
| Category | Bills & Utilities | `#4256B8` indigo | `bills` |
| Category | Shopping | `#C2588A` pink | `shopping` |
| Category | Loan Given | `#5E6E80` slate | `gift` |
| Budget | Food & Dining | `#C2553F` coral | `food` |
| Budget | Transportation | `#E0A23B` amber | `transport` |
| Budget | Entertainment | `#7C5BC2` purple | `entertainment` |
| Budget | Shopping | — (fallback) | `shopping` |
| Goal | Europe Trip 2027 | `#3E72B8` denim | `travel` |
| Installment | MacBook M5 Pro | `#4256B8` indigo | `work` |
| Subscription | Netflix Premium | `#7C5BC2` purple | `entertainment` |
| Recurring rule | Spotify Family | `#2E8B57` green | `entertainment` |

The "Berbagi Dompet" accounts follow the same scheme: `Mandiri Pengamat` (`#E0A23B`/`bank`), `Dana` (`#4256B8`/`ewallet`), `Jenius` (colorless/`bank`), and pengamat's categories `Gaji` (`#2E8B57`/`salary`), `Belanja` (`#C2588A`/`shopping`), `Makan` (`#C2553F`/`food`). Budgets intentionally mirror their category's color/icon so the anggaran list reads consistently.

## Berbagi Dompet demo

The seed also creates two extra accounts (same password, `password123`) so the **Berbagi Dompet** (wallet-history sharing) feature is visible from every angle when logged in as `demo@affluena.com`:

| Email | Role in the demo |
|-------|------------------|
| `pengamat@affluena.com` | A real account with its own wallets (`Mandiri Pengamat`, `Dana`), categories, and transactions. Linked **both ways** with `demo`, so each can view the other's wallets. |
| `calon@affluena.com` | Has only a **pending** invite to `demo` (one wallet, `Jenius`). Demonstrates the "Undangan masuk" → Terima/Tolak UI. No wallet access is granted until accepted. |

Three `wallet_share_links` are created: `demo → pengamat` (joined), `pengamat → demo` (joined), and `calon → demo` (pending). For each **joined** link, the seed runs the exact same fan-out the API's accept handler uses (`internal/partner/repository.go` `Respond`): one read-only (`viewer`) `wallet_shares` row, tagged `source='link'`, for every wallet the link's owner owns.

When logged in as `demo@affluena.com`:

- **Pengaturan → Berbagi Dompet** lists the connected viewer (`Terhubung`) under "Pengamat saya (N/5)" and the pending `calon` invite under "Undangan masuk".
- **Beranda** shows a dedicated **"Dibagikan untukku"** section with the viewer's wallets (read-only, `LIHAT` badge), excluded from the personal **Dompet** section and the Total-saldo hero.

Logging in as `pengamat@affluena.com` is symmetric — its Beranda shows `demo`'s wallets in that section.

## Stable Demo Identifiers

The seed uses fixed UUIDs for core lookup data so local UI and QA checks can be deterministic.

| UUID | Type | Name |
|------|------|------|
| `22222222-2222-2222-2222-222222220001` | Wallet | Cash Wallet |
| `22222222-2222-2222-2222-222222220002` | Wallet | BCA Primary |
| `22222222-2222-2222-2222-222222220003` | Wallet | GoPay |
| `22222222-2222-2222-2222-222222220004` | Wallet | Jenius |
| `22222222-2222-2222-2222-222222220005` | Wallet | OVO |
| `33333333-3333-3333-3333-333333330001` | Income category | Salary |
| `33333333-3333-3333-3333-333333330002` | Income category | Freelance |
| `33333333-3333-3333-3333-333333330003` | Income category | Loan Repayment |
| `44444444-4444-4444-4444-444444440001` | Expense category | Food & Dining |
| `44444444-4444-4444-4444-444444440002` | Expense category | Transportation |
| `44444444-4444-4444-4444-444444440003` | Expense category | Entertainment |
| `44444444-4444-4444-4444-444444440004` | Expense category | Bills & Utilities |
| `44444444-4444-4444-4444-444444440005` | Expense category | Shopping |
| `44444444-4444-4444-4444-444444440006` | Expense category | Loan Given |
| `55555555-5555-5555-5555-555555550001` | Tag | #BaliTrip |
| `55555555-5555-5555-5555-555555550002` | Tag | #MonthlyBill |
| `77777777-7777-7777-7777-777777770001` | User | Pengamat (`pengamat@affluena.com`) |
| `77777777-7777-7777-7777-777777770002` | User | Calon Pengamat (`calon@affluena.com`) |
| `88888888-8888-8888-8888-888888880001` | Wallet | Mandiri Pengamat (pengamat) |
| `88888888-8888-8888-8888-888888880002` | Wallet | Dana (pengamat) |
| `88888888-8888-8888-8888-888888880003` | Wallet | Jenius (calon) |

The seeded quick entry template is `Daily Coffee`; it stores `wallet_id=22222222-2222-2222-2222-222222220003` and `category_id=44444444-4444-4444-4444-444444440001`, which resolve to `GoPay` and `Food & Dining`.

## Manual Postman Bootstrap

## Bootstrap Recommendations

Use the Postman collection when you need custom test data instead of the standard demo seed:

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
After running `make seed`, `/api/v1/dashboard/summary` should show non-empty wallet, cashflow, budget, debt, tracker, recurring, and goal data for `demo@affluena.com`. The exact monthly values depend on the current date because the seed uses `time.Now()` for current-month transactions and due dates.

## Shared Wallet Scenario
If testing shared wallets:
1. Register a second user `dev2@example.com`.
2. Login as `dev@example.com`.
3. Invite `dev2`'s email via `POST /api/v1/wallets/:id/invites`.
4. Login as `dev2@example.com` and accept the invite via `PATCH /api/v1/wallets/:id/members/:member_id`.
5. Transactions made by `dev2` on the shared wallet will affect `dev`'s wallet balances, but will NOT affect `dev`'s personal category budgets.
