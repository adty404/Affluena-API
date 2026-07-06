package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pgxstdlib "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

const (
	testEmail    = "demo@affluena.com"
	testPassword = "password123"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("AFFLUENA_API_TEST_DATABASE_URL")
	}
	if dbURL == "" {
		dbURL = "postgres://affluena_api:affluena_api@localhost:5432/affluena_api?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer pool.Close()

	db := pgxstdlib.OpenDBFromPool(pool)
	defer db.Close()

	pwHash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	// Anchor all seeded history to the LAST 3 COMPLETE calendar months, in WIB
	// (UTC+7, the app's audience timezone). Seeding on e.g. 2026-07-04 fills
	// April, May and June 2026 — all fully-past months, so nothing is
	// future-dated — so the 6/12-month cashflow-trend and net-worth views are
	// non-flat and the app feels lived-in. Building the timestamps in WIB (not
	// UTC) keeps late-evening rows from spilling into the next month for the WIB
	// reader. `now` is the seed's "present" (the last moment of the LATEST seeded
	// month), used for the relative operational dates below.
	//
	// `monthStarts` is ordered OLDEST -> NEWEST; `monthStart` (the newest, i.e.
	// the previous complete month) still drives the operational entities and the
	// current-month budgets below, preserving the previous behaviour.
	const seedMonths = 3
	wib := time.FixedZone("WIB", 7*60*60)
	firstOfThisMonth := time.Date(time.Now().In(wib).Year(), time.Now().In(wib).Month(), 1, 0, 0, 0, 0, wib)
	monthStarts := make([]time.Time, seedMonths)
	for i := 0; i < seedMonths; i++ {
		// i=0 -> oldest (-3 months from firstOfThisMonth), i=seedMonths-1 -> newest (-1).
		monthStarts[i] = firstOfThisMonth.AddDate(0, -(seedMonths - i), 0)
	}
	monthStart := monthStarts[seedMonths-1]
	now := firstOfThisMonth.Add(-time.Second)

	// Fixed UUIDs for idempotency
	const (
		uID        = "11111111-1111-1111-1111-111111111111"
		wCash      = "22222222-2222-2222-2222-222222220001"
		wBank      = "22222222-2222-2222-2222-222222220002"
		wGoPay     = "22222222-2222-2222-2222-222222220003"
		wJenius    = "22222222-2222-2222-2222-222222220004"
		wOVO       = "22222222-2222-2222-2222-222222220005"
		cSalary    = "33333333-3333-3333-3333-333333330001"
		cFreelance = "33333333-3333-3333-3333-333333330002"
		cFood      = "44444444-4444-4444-4444-444444440001"
		cTrans     = "44444444-4444-4444-4444-444444440002"
		cEnt       = "44444444-4444-4444-4444-444444440003"
		cBills     = "44444444-4444-4444-4444-444444440004"
		cShop      = "44444444-4444-4444-4444-444444440005"
		cDebtDisb  = "44444444-4444-4444-4444-444444440006"
		cDebtPay   = "33333333-3333-3333-3333-333333330003"
		tagBali    = "55555555-5555-5555-5555-555555550001"
		tagMonthly = "55555555-5555-5555-5555-555555550002"

		// "Berbagi Dompet" (account-level wallet-history sharing) demo.
		// viewerEmail is a real second account (with its own wallets) that demo
		// shares with both ways; calonEmail is a third account used only to show
		// an incoming pending invite.
		viewerEmail = "pengamat@affluena.com"
		calonEmail  = "calon@affluena.com"

		uViewer    = "77777777-7777-7777-7777-777777770001"
		uCalon     = "77777777-7777-7777-7777-777777770002"
		wpMandiri  = "88888888-8888-8888-8888-888888880001"
		wpDana     = "88888888-8888-8888-8888-888888880002"
		wcJenius   = "88888888-8888-8888-8888-888888880003"
		cpSalary   = "99999999-9999-9999-9999-999999990001"
		cpShopping = "99999999-9999-9999-9999-999999990002"
		cpFood     = "99999999-9999-9999-9999-999999990003"
	)

	fmt.Println("Seeding Affluena demo data...")
	fmt.Printf("  User: %s / %s\n", testEmail, testPassword)

	// Clean up existing demo users (cascade deletes all owned data, including
	// wallet_share_links and the auto-managed wallet_shares, via ON DELETE
	// CASCADE). Match on both the stable demo UUIDs and the emails, so re-seeding
	// is clean even if an earlier seed used different demo emails at the same
	// UUIDs (e.g. the pre-rebrand "pasangan@affluena.com").
	mustExec(db, `DELETE FROM users WHERE id IN ($1, $2, $3) OR email IN ($4, $5, $6)`,
		uID, uViewer, uCalon, testEmail, viewerEmail, calonEmail)

	mustExec(db, `INSERT INTO users (id, email, password_hash, name) VALUES ($1, $2, $3, 'Aditya (Demo)')`,
		uID, testEmail, string(pwHash))

	// color/icon values come from the shared client catalogs (10-swatch color
	// palette + kWalletIconCatalog / kCategoryIconCatalog ids). EVERY category
	// carries both a color and an icon (they surface in transaction history
	// everywhere, so a colorless category would look unfinished); a couple of
	// wallet/other-module rows are still left colorless on purpose to exercise
	// the client-side fallback rendering.
	mustExec(db, `INSERT INTO wallets (id, user_id, name, type, currency_code, balance_minor, color, icon) VALUES
		($1, $2, 'Cash Wallet', 'cash', 'IDR', 850000, '', 'cash'),
		($3, $2, 'BCA Primary', 'bank', 'IDR', 15200000, '#3E72B8', 'bank'),
		($4, $2, 'GoPay', 'e_wallet', 'IDR', 320000, '#2BB3A3', 'ewallet'),
		($5, $2, 'Jenius', 'bank', 'IDR', 7400000, '#4256B8', 'bank'),
		($6, $2, 'OVO', 'e_wallet', 'IDR', 180000, '#7C5BC2', 'ewallet')
		ON CONFLICT DO NOTHING`, wCash, uID, wBank, wGoPay, wJenius, wOVO)

	mustExec(db, `INSERT INTO categories (id, user_id, name, type, color, icon) VALUES
		($1, $2, 'Salary', 'income', '#2E8B57', 'salary'),
		($3, $2, 'Freelance', 'income', '#9E7B4F', 'work'),
		($4, $2, 'Loan Repayment', 'income', '#2BB3A3', 'savings'),
		($5, $2, 'Food & Dining', 'expense', '#C2553F', 'food'),
		($6, $2, 'Transportation', 'expense', '#E0A23B', 'transport'),
		($7, $2, 'Entertainment', 'expense', '#7C5BC2', 'entertainment'),
		($8, $2, 'Bills & Utilities', 'expense', '#4256B8', 'bills'),
		($9, $2, 'Shopping', 'expense', '#C2588A', 'shopping'),
		($10, $2, 'Loan Given', 'expense', '#5E6E80', 'gift')
		ON CONFLICT DO NOTHING`,
		cSalary, uID, cFreelance, cDebtPay, cFood, cTrans, cEnt, cBills, cShop, cDebtDisb)

	mustExec(db, `INSERT INTO tags (id, user_id, name) VALUES
		($1, $2, '#BaliTrip'),
		($3, $2, '#MonthlyBill')
		ON CONFLICT DO NOTHING`, tagBali, uID, tagMonthly)

	// --- Multi-month transaction simulation ---------------------------------
	// The Kalender / Wawasan / history screens (and the 6/12-month cashflow-trend
	// and net-worth views) need LIVING history, so we spread ~7 transactions PER
	// DAY across each of the last 3 complete calendar months (~210/month, ~630
	// total). Each month is generated by seedMonthTransactions with a per-month
	// offset so amounts/notes vary a little between months (they are not carbon
	// copies) while staying deterministic — re-seeds produce identical content
	// (only the row UUIDs differ), idempotent via the delete-then-recreate above.
	//
	// Every row for a month lands strictly INSIDE that month (WIB): dates anchor
	// to the month's day 1 plus a day/time offset, days are clamped to the real
	// month length, and the time zone is WIB so late-evening rows never spill into
	// the next month. Nothing is future-dated (all 3 months are fully past).
	//
	// Balances follow the existing seed convention: wallets carry a fixed,
	// illustrative opening balance_minor (set above) and are NOT recomputed from
	// these transactions. The seeder never derived balances from the ledger, so
	// we keep that pattern and do not introduce a balance/ledger invariant.
	demoWallets := demoWalletIDs{cash: wCash, bank: wBank, goPay: wGoPay, jenius: wJenius, ovo: wOVO}
	demoCats := demoCategoryIDs{salary: cSalary, freelance: cFreelance, food: cFood, trans: cTrans, ent: cEnt, bills: cBills, shop: cShop}

	var incomeCount, expenseCount, transferCount int
	for offset, ms := range monthStarts {
		inc, exp, tr := seedMonthTransactions(db, uID, ms, wib, offset, demoWallets, demoCats)
		incomeCount += inc
		expenseCount += exp
		transferCount += tr
	}

	// Budgets — LATEST seeded month only (monthStart = the previous complete
	// month); the two older months intentionally have no budgets. Appearance
	// follows each budget's category. The Shopping budget is intentionally
	// colorless (client fallback).
	mustExec(db, `INSERT INTO category_budgets (id, user_id, category_id, month, limit_minor, color, icon) VALUES
		(gen_random_uuid(), $1, $2, $3, 2000000, '#C2553F', 'food'),
		(gen_random_uuid(), $1, $4, $3, 1500000, '#E0A23B', 'transport'),
		(gen_random_uuid(), $1, $5, $3, 3000000, '#7C5BC2', 'entertainment'),
		(gen_random_uuid(), $1, $6, $3, 5000000, '', 'shopping')
		ON CONFLICT DO NOTHING`,
		uID, cFood, monthStart, cTrans, cEnt, cShop)

	// Payable debt
	debtTxID := queryUUID(db, `INSERT INTO transactions (id, user_id, wallet_id, category_id, type, amount_minor, transaction_at, note) VALUES
		(gen_random_uuid(), $1, $2, $3, 'expense', 5000000, $4, 'KTA Loan Disbursement') RETURNING id::text`,
		uID, wBank, cDebtDisb, now.AddDate(0, 0, -20))
	mustExec(db, `INSERT INTO debts (id, user_id, type, counterparty_name, wallet_id, disbursement_category_id, payment_category_id, origination_transaction_id, principal_amount_minor, paid_amount_minor, opened_at, due_date, status, note) VALUES
		(gen_random_uuid(), $1, 'payable', 'BCA KTA', $2, $3, $4, $5::uuid, 5000000, 1000000, $6, $7::date, 'open', 'Car repair loan') ON CONFLICT DO NOTHING`,
		uID, wBank, cDebtDisb, cDebtPay, debtTxID, now.AddDate(0, 0, -20), now.AddDate(0, 0, 10))

	// Subscription
	mustExec(db, `INSERT INTO subscriptions (id, user_id, name, account_detail, wallet_id, category_id, amount_minor, billing_cycle, next_due_date, status, note, color, icon) VALUES
		(gen_random_uuid(), $1, 'Netflix Premium', 'demo@affluena.com', $2, $3, 186000, 'monthly', $4::date, 'active', 'Family Plan', '#7C5BC2', 'entertainment') ON CONFLICT DO NOTHING`,
		uID, wBank, cEnt, now.AddDate(0, 0, 14))

	// Installment
	mustExec(db, `INSERT INTO installments (id, user_id, name, wallet_id, category_id, total_amount_minor, monthly_amount_minor, tenor_months, remaining_months, start_date, due_day, status, note, color, icon) VALUES
		(gen_random_uuid(), $1, 'MacBook M5 Pro', $2, $3, 30000000, 2500000, 12, 10, $4::date, 15, 'active', '0% Cicilan', '#4256B8', 'work') ON CONFLICT DO NOTHING`,
		uID, wBank, cShop, now.AddDate(0, -2, 0))

	// Recurring rule
	mustExec(db, `INSERT INTO recurring_transaction_rules (id, user_id, name, type, wallet_id, category_id, amount_minor, frequency, next_run_at, status, color, icon) VALUES
		(gen_random_uuid(), $1, 'Spotify Family', 'expense', $2, $3, 86000, 'monthly', $4, 'active', '#2E8B57', 'entertainment') ON CONFLICT DO NOTHING`,
		uID, wGoPay, cEnt, now.AddDate(0, 0, 5))

	// Goal + goal wallet (goal wallet mirrors the goal's appearance)
	mustExec(db, `INSERT INTO goals (id, user_id, name, target_amount_minor, deadline, status, color, icon) VALUES
		('66666666-6666-6666-6666-666666660001', $1, 'Europe Trip 2027', 50000000, $2::date, 'active', '#3E72B8', 'travel') ON CONFLICT DO NOTHING`,
		uID, now.AddDate(1, 0, 0))
	mustExec(db, `INSERT INTO wallets (id, user_id, name, type, currency_code, balance_minor, goal_id, color, icon) VALUES
		(gen_random_uuid(), $1, 'Europe Trip Fund', 'goal', 'IDR', 8500000, '66666666-6666-6666-6666-666666660001', '#3E72B8', 'travel') ON CONFLICT DO NOTHING`,
		uID)

	// Quick entry template
	mustExec(db, `INSERT INTO quick_entry_templates (id, user_id, name, type, wallet_id, category_id, amount_minor, note) VALUES
		(gen_random_uuid(), $1, 'Daily Coffee', 'expense', $2, $3, 35000, 'Iced Latte') ON CONFLICT DO NOTHING`,
		uID, wGoPay, cFood)

	// --- "Berbagi Dompet" (account-level wallet-history sharing) demo ----------
	// Two extra accounts (same password) so the sharing feature is visible from
	// every angle when logged in as demo@affluena.com:
	//   * pengamat@affluena.com -- a real account with its own wallets/txns.
	//     Linked both ways with demo, so demo can VIEW their wallets (shown in
	//     the Beranda "Dibagikan untukku" section) and they can view demo's.
	//   * calon@affluena.com -- has only a PENDING invite to demo, to show the
	//     "Undangan masuk" (Terima/Tolak) UI.

	mustExec(db, `INSERT INTO users (id, email, password_hash, name) VALUES ($1, $2, $3, 'Pengamat (Demo)')`,
		uViewer, viewerEmail, string(pwHash))
	mustExec(db, `INSERT INTO users (id, email, password_hash, name) VALUES ($1, $2, $3, 'Calon Pengamat (Demo)')`,
		uCalon, calonEmail, string(pwHash))

	// The viewer's own finances (so demo sees real balances + transactions when
	// viewing them read-only under "Dibagikan untukku").
	mustExec(db, `INSERT INTO wallets (id, user_id, name, type, currency_code, balance_minor, color, icon) VALUES
		($1, $2, 'Mandiri Pengamat', 'bank', 'IDR', 6750000, '#E0A23B', 'bank'),
		($3, $2, 'Dana', 'e_wallet', 'IDR', 410000, '#4256B8', 'ewallet')
		ON CONFLICT DO NOTHING`, wpMandiri, uViewer, wpDana)
	mustExec(db, `INSERT INTO categories (id, user_id, name, type, color, icon) VALUES
		($1, $2, 'Gaji', 'income', '#2E8B57', 'salary'),
		($3, $2, 'Belanja', 'expense', '#C2588A', 'shopping'),
		($4, $2, 'Makan', 'expense', '#C2553F', 'food')
		ON CONFLICT DO NOTHING`, cpSalary, uViewer, cpShopping, cpFood)
	mustExec(db, `INSERT INTO transactions (id, user_id, wallet_id, category_id, type, amount_minor, transaction_at, note) VALUES
		(gen_random_uuid(), $1, $2, $3, 'income', 9000000, $6, 'Gaji bulanan'),
		(gen_random_uuid(), $1, $2, $4, 'expense', 525000, $6 - interval '2 days', 'Belanja bulanan'),
		(gen_random_uuid(), $1, $5, $7, 'expense', 88000, $6 - interval '1 day', 'Makan siang'),
		(gen_random_uuid(), $1, $5, $7, 'expense', 132000, $6 - interval '3 days', 'Makan malam')
		ON CONFLICT DO NOTHING`,
		uViewer, wpMandiri, cpSalary, cpShopping, wpDana, now.AddDate(0, 0, -2), cpFood)

	// Calon only needs a wallet so that, if demo accepts the pending invite in
	// the app, there is something to view. Intentionally colorless (fallback).
	mustExec(db, `INSERT INTO wallets (id, user_id, name, type, currency_code, balance_minor, color, icon) VALUES
		($1, $2, 'Jenius', 'bank', 'IDR', 2000000, '', 'bank')
		ON CONFLICT DO NOTHING`, wcJenius, uCalon)

	// Share links. Each link is one-way; the two between demo and the viewer
	// model two people who each share with the other. The third is an invite to
	// demo still awaiting a response.
	mustExec(db, `INSERT INTO wallet_share_links (id, owner_id, viewer_id, status) VALUES
		(gen_random_uuid(), $1, $2, 'joined'),
		(gen_random_uuid(), $2, $1, 'joined'),
		(gen_random_uuid(), $3, $1, 'pending')
		ON CONFLICT (owner_id, viewer_id) DO NOTHING`,
		uID, uViewer, uCalon)

	// Fan out viewer shares for every JOINED link -- identical SQL to the API's
	// accept handler (internal/partner/repository.go Respond). Every wallet of a
	// link's owner becomes a read-only ('viewer') share for the invited person,
	// tagged source='link'. Pending links (calon) grant nothing.
	mustExec(db, `INSERT INTO wallet_shares (wallet_id, user_id, status, role, source)
		SELECT w.id, pl.viewer_id, 'joined', 'viewer', 'link'
		FROM wallet_share_links pl
		JOIN wallets w ON w.user_id = pl.owner_id
		WHERE pl.status = 'joined'
		ON CONFLICT (wallet_id, user_id) DO NOTHING`)

	fmt.Println("Seed complete!")
	fmt.Println()
	fmt.Println("Login credentials:")
	fmt.Printf("  Email:    %s\n", testEmail)
	fmt.Printf("  Password: %s\n", testPassword)
	fmt.Println()
	// The transaction counts are DERIVED from the rows actually inserted above
	// (seedMonthTransactions returns its own per-call counts, accumulated across
	// the 3 months) so the printout never drifts from the generator. The day loop
	// produces ~7 expense rows per day for each seeded month, so the total varies
	// with month length (roughly 620–680 over 3 months). The debt disbursement is
	// a fixed extra insert.
	const debtDisbursementCount = 1
	demoTxTotal := incomeCount + expenseCount + transferCount + debtDisbursementCount
	// e.g. "April–June 2026" (or spanning years, "December 2025–February 2026").
	oldest, newest := monthStarts[0], monthStarts[seedMonths-1]
	var monthRange string
	if oldest.Year() == newest.Year() {
		monthRange = fmt.Sprintf("%s–%s", oldest.Format("January"), newest.Format("January 2006"))
	} else {
		monthRange = fmt.Sprintf("%s–%s", oldest.Format("January 2006"), newest.Format("January 2006"))
	}

	fmt.Println("Seeded data:")
	fmt.Println("  5 wallets (Cash, BCA Bank, GoPay, Jenius bank, OVO e-wallet)")
	fmt.Println("  9 categories (3 income, 6 expense)")
	fmt.Println("  2 tags")
	fmt.Printf("  %d transactions spread across %s (last 3 complete months)\n", demoTxTotal, monthRange)
	fmt.Printf("    (%d income, %d expense, %d transfer + %d debt disbursement)\n",
		incomeCount, expenseCount, transferCount, debtDisbursementCount)
	fmt.Printf("  4 budgets for %s only (Food, Transport, Entertainment, Shopping)\n", monthStart.Format("January 2006"))
	fmt.Println("  1 payable debt (BCA KTA)")
	fmt.Println("  1 subscription (Netflix)")
	fmt.Println("  1 installment (MacBook)")
	fmt.Println("  1 recurring rule (Spotify)")
	fmt.Println("  1 goal (Europe Trip)")
	fmt.Println("  1 quick entry template")
	fmt.Println("  Appearance: wallets/categories/budgets/goal/installment/")
	fmt.Println("  subscription/recurring carry demo color+icon values from the")
	fmt.Println("  client catalogs (a few rows left colorless for fallback UI)")
	fmt.Println()
	fmt.Println("\"Berbagi Dompet\" (wallet-history sharing) demo:")
	fmt.Printf("  %s / %s  (own wallets, shared both ways with demo)\n", viewerEmail, testPassword)
	fmt.Printf("  %s / %s  (pending invite to demo -> Terima/Tolak)\n", calonEmail, testPassword)
	fmt.Println("  Logged in as demo: Pengaturan > Berbagi Dompet shows the")
	fmt.Println("  connected viewer + the pending invite; Beranda shows a")
	fmt.Println("  \"Dibagikan untukku\" section with the viewer's wallets (read-only).")
}

// demoWalletIDs / demoCategoryIDs pass the demo user's fixed wallet/category
// UUIDs into seedMonthTransactions without a long positional argument list.
type demoWalletIDs struct {
	cash   string
	bank   string
	goPay  string
	jenius string
	ovo    string
}

type demoCategoryIDs struct {
	salary    string
	freelance string
	food      string
	trans     string
	ent       string
	bills     string
	shop      string
}

// seedMonthTransactions generates one lived-in calendar month of demo history
// (income + ~7 expenses/day + a few inter-wallet transfers) for `monthStart`
// (day 1 of the target month, in WIB) and inserts it. It is called once per
// seeded month; `monthOffset` (0 = oldest) keys the deterministic amount/note
// helpers so each month varies a little without introducing randomness — the
// content is fully determined by (monthStart, monthOffset), so re-seeds are
// idempotent (only the row UUIDs change). Every row lands strictly inside the
// target month: `dayAt` clamps the day into the real month length and builds
// the timestamp in WIB, so late-evening rows never spill into the next month.
//
// Returns the number of income, expense and transfer rows inserted so the
// caller can accumulate an accurate total for the printout.
func seedMonthTransactions(db *sql.DB, uID string, monthStart time.Time, wib *time.Location, monthOffset int, w demoWalletIDs, c demoCategoryIDs) (incomeCount, expenseCount, transferCount int) {
	daysInMonth := monthStart.AddDate(0, 1, -1).Day()
	dayAt := func(day, hour, minsec int) time.Time {
		if day < 1 {
			day = 1
		}
		if day > daysInMonth {
			day = daysInMonth
		}
		return time.Date(monthStart.Year(), monthStart.Month(), day, hour, minsec/60, minsec%60, 0, wib)
	}

	type tx struct {
		wallet   string
		category string
		amount   int64
		day      int
		hour     int
		note     string
	}

	// mk offsets the deterministic key by the month so the same day in different
	// months does not produce identical amounts/notes (months are not carbon
	// copies) while staying fully deterministic.
	mk := func(k int) int { return k + monthOffset*101 }

	// Income: salary (BCA) + freelance payments spread through the month. Salary
	// dominates income; freelance adds variety to the Wawasan income breakdown.
	// Amounts are nudged per month so the cashflow-trend line is not flat.
	monthlySalary := int64(18500000) + int64(monthOffset)*250000
	incomes := []tx{
		{w.bank, c.salary, monthlySalary, 1, 9, "Gaji bulanan"},
		{w.bank, c.freelance, 3200000 + int64((monthOffset*7)%5)*100000, 5, 14, "Freelance - landing page"},
		{w.bank, c.freelance, 1750000 + int64((monthOffset*3)%4)*100000, 12, 11, "Freelance - logo design"},
		{w.bank, c.freelance, 2400000 + int64((monthOffset*5)%6)*100000, 19, 16, "Freelance - app maintenance"},
		{w.bank, c.freelance, 1400000 + int64((monthOffset*2)%3)*100000, 26, 15, "Freelance - bug fixing"},
	}

	// Expenses — a LIVED-IN month: ~7 transactions per day, every day, generated
	// deterministically (no randomness) so the seed stays idempotent across runs.
	// Food & Dining stays the biggest bucket (3 of the 7 daily slots), then
	// Transportation, with Shopping / Entertainment / Bills rotating through the
	// remaining slots. Amounts are realistic IDR whole-rupiah values, nudged a
	// little per day (and per month via mk) so rows are not carbon copies.
	type slot struct {
		wallet   string
		category string
		base     int64
		notes    []string
	}
	// Daily food/transport pools (small, frequent) + occasional bigger buckets.
	foodPool := []slot{
		{w.cash, c.food, 38000, []string{"Sarapan bubur ayam", "Sarapan lontong", "Nasi uduk pagi"}},
		{w.goPay, c.food, 45000, []string{"Kopi pagi", "Kopi + roti", "Es kopi susu"}},
		{w.cash, c.food, 47000, []string{"Makan siang warteg", "Makan siang padang", "Nasi ayam"}},
		{w.goPay, c.food, 68000, []string{"GoFood makan malam", "GoFood ayam geprek", "GoFood martabak"}},
		{w.ovo, c.food, 52000, []string{"Cemilan sore", "Kopi kekinian", "Boba + snack"}},
		{w.goPay, c.food, 60000, []string{"Makan siang mall", "Bakso malam", "Mie ayam"}},
	}
	foodBigPool := []slot{
		{w.cash, c.food, 235000, []string{"Groceries Superindo", "Belanja mingguan", "Groceries Indomaret"}},
		{w.ovo, c.food, 175000, []string{"Dinner bareng teman", "Dinner date", "Makan malam keluarga"}},
	}
	transportPool := []slot{
		{w.goPay, c.trans, 28000, []string{"GoRide ke kantor", "GoRide", "GoRide meeting"}},
		{w.goPay, c.trans, 45000, []string{"GoCar pulang", "GoCar malam", "GrabCar"}},
		{w.ovo, c.trans, 22000, []string{"Parkir", "Ojek pangkalan", "Angkot"}},
	}
	transportBigPool := []slot{
		{w.cash, c.trans, 290000, []string{"Bensin mobil", "Isi Pertamax", "Bensin + tol"}},
	}
	rotatePool := []slot{
		{w.bank, c.shop, 210000, []string{"Baju kerja", "Skincare Shopee", "Aksesoris HP", "Sepatu baru"}},
		{w.goPay, c.ent, 60000, []string{"Nonton bioskop", "Game top-up", "Karaoke", "Langganan streaming"}},
		{w.ovo, c.food, 55000, []string{"Ngopi sore", "Jajan pasar", "Gorengan"}},
	}
	// Bills land on a few fixed days (like real monthly utilities).
	billsByDay := map[int][]slot{
		3:  {{w.bank, c.bills, 285000, []string{"Listrik PLN"}}, {w.bank, c.bills, 155000, []string{"Air PDAM"}}},
		10: {{w.bank, c.bills, 250000, []string{"Internet + TV"}}},
		15: {{w.bank, c.bills, 150000, []string{"Pulsa + paket data"}}},
		25: {{w.bank, c.bills, 120000, []string{"Langganan cloud"}}},
	}

	// Deterministic pick helpers (no randomness → stable across seed runs).
	pick := func(pool []slot, k int) slot { return pool[((k%len(pool))+len(pool))%len(pool)] }
	note := func(s slot, k int) string { return s.notes[((k%len(s.notes))+len(s.notes))%len(s.notes)] }
	amount := func(base int64, k int) int64 { return base + int64((k*7)%9)*1000 }

	var expenses []tx
	for d := 1; d <= daysInMonth; d++ {
		day := []tx{}
		// 3 everyday food spends (breakfast, lunch, dinner-ish) at spread hours.
		foodHours := []int{8, 12, 19}
		for i, h := range foodHours {
			s := pick(foodPool, mk(d+i))
			day = append(day, tx{s.wallet, s.category, amount(s.base, mk(d*3+i)), d, h, note(s, mk(d+i))})
		}
		// A bigger food/grocery run roughly twice a week.
		if d%4 == 0 {
			s := pick(foodBigPool, mk(d))
			day = append(day, tx{s.wallet, s.category, amount(s.base, mk(d)), d, 11, note(s, mk(d))})
		}
		// 1 daily transport + a fuel fill-up about weekly.
		st := pick(transportPool, mk(d))
		day = append(day, tx{st.wallet, st.category, amount(st.base, mk(d*2)), d, 9, note(st, mk(d))})
		if d%7 == 3 {
			sb := pick(transportBigPool, mk(d))
			day = append(day, tx{sb.wallet, sb.category, amount(sb.base, mk(d)), d, 17, note(sb, mk(d))})
		}
		// A rotating shopping / entertainment / extra-food spend.
		sr := pick(rotatePool, mk(d))
		day = append(day, tx{sr.wallet, sr.category, amount(sr.base, mk(d)), d, 16, note(sr, mk(d))})
		// Fixed monthly bills on their days.
		for _, sb := range billsByDay[d] {
			day = append(day, tx{sb.wallet, sb.category, sb.base, d, 10, sb.notes[0]})
		}
		// Top up to ~7/day when a day fell short (no big food/fuel that day).
		for len(day) < 7 {
			s := pick(foodPool, mk(d+len(day)))
			day = append(day, tx{s.wallet, s.category, amount(s.base, mk(d*5+len(day))), d, 14 + len(day)%4, note(s, mk(d+len(day)))})
		}
		expenses = append(expenses, day...)
	}

	for _, e := range incomes {
		mustExec(db, `INSERT INTO transactions (id, user_id, wallet_id, category_id, type, amount_minor, transaction_at, note) VALUES
			(gen_random_uuid(), $1, $2, $3, 'income', $4, $5, $6) ON CONFLICT DO NOTHING`,
			uID, e.wallet, e.category, e.amount, dayAt(e.day, e.hour, 0), e.note)
	}
	for _, e := range expenses {
		mustExec(db, `INSERT INTO transactions (id, user_id, wallet_id, category_id, type, amount_minor, transaction_at, note) VALUES
			(gen_random_uuid(), $1, $2, $3, 'expense', $4, $5, $6) ON CONFLICT DO NOTHING`,
			uID, e.wallet, e.category, e.amount, dayAt(e.day, e.hour, 0), e.note)
	}

	// Transfers (topups) — spread across the month between wallets.
	mustExec(db, `INSERT INTO transactions (id, user_id, wallet_id, to_wallet_id, type, amount_minor, transaction_at, note) VALUES
		(gen_random_uuid(), $1, $2, $3, 'transfer', 1500000, $6, 'Topup GoPay'),
		(gen_random_uuid(), $1, $2, $4, 'transfer', 500000, $7, 'Topup OVO'),
		(gen_random_uuid(), $1, $2, $5, 'transfer', 1000000, $8, 'Pindah ke Jenius')
		ON CONFLICT DO NOTHING`,
		uID, w.bank, w.goPay, w.ovo, w.jenius,
		dayAt(2, 8, 0), dayAt(9, 8, 0), dayAt(18, 8, 0))

	return len(incomes), len(expenses), 3
}

func mustExec(db *sql.DB, query string, args ...interface{}) {
	_, err := db.Exec(query, args...)
	if err != nil {
		log.Fatalf("seed failed: %s\n  query: %s\n  error: %v", query[:min(80, len(query))], query, err)
	}
}

func queryUUID(db *sql.DB, query string, args ...interface{}) string {
	var id string
	err := db.QueryRow(query, args...).Scan(&id)
	if err != nil {
		log.Fatalf("seed query failed: %v", err)
	}
	return id
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
