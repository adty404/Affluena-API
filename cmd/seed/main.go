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

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Fixed UUIDs for idempotency
	const (
		uID       = "11111111-1111-1111-1111-111111111111"
		wCash     = "22222222-2222-2222-2222-222222220001"
		wBank     = "22222222-2222-2222-2222-222222220002"
		wGoPay    = "22222222-2222-2222-2222-222222220003"
		cSalary   = "33333333-3333-3333-3333-333333330001"
		cFreelance = "33333333-3333-3333-3333-333333330002"
		cFood     = "44444444-4444-4444-4444-444444440001"
		cTrans    = "44444444-4444-4444-4444-444444440002"
		cEnt      = "44444444-4444-4444-4444-444444440003"
		cBills    = "44444444-4444-4444-4444-444444440004"
		cShop     = "44444444-4444-4444-4444-444444440005"
		cDebtDisb = "44444444-4444-4444-4444-444444440006"
		cDebtPay  = "33333333-3333-3333-3333-333333330003"
		tagBali   = "55555555-5555-5555-5555-555555550001"
		tagMonthly = "55555555-5555-5555-5555-555555550002"
	)

	fmt.Println("Seeding Affluena demo data...")
	fmt.Printf("  User: %s / %s\n", testEmail, testPassword)

	// Clean up existing demo user (cascade deletes all owned data)
	mustExec(db, `DELETE FROM users WHERE email = $1`, testEmail)

	mustExec(db, `INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)`,
		uID, testEmail, string(pwHash))

	mustExec(db, `INSERT INTO wallets (id, user_id, name, type, currency_code, balance_minor) VALUES
		($1, $2, 'Cash Wallet', 'cash', 'IDR', 850000),
		($3, $2, 'BCA Primary', 'bank', 'IDR', 15200000),
		($4, $2, 'GoPay', 'e_wallet', 'IDR', 320000)
		ON CONFLICT DO NOTHING`, wCash, uID, wBank, wGoPay)

	mustExec(db, `INSERT INTO categories (id, user_id, name, type) VALUES
		($1, $2, 'Salary', 'income'),
		($3, $2, 'Freelance', 'income'),
		($4, $2, 'Loan Repayment', 'income'),
		($5, $2, 'Food & Dining', 'expense'),
		($6, $2, 'Transportation', 'expense'),
		($7, $2, 'Entertainment', 'expense'),
		($8, $2, 'Bills & Utilities', 'expense'),
		($9, $2, 'Shopping', 'expense'),
		($10, $2, 'Loan Given', 'expense')
		ON CONFLICT DO NOTHING`,
		cSalary, uID, cFreelance, cDebtPay, cFood, cTrans, cEnt, cBills, cShop, cDebtDisb)

	mustExec(db, `INSERT INTO tags (id, user_id, name) VALUES
		($1, $2, '#BaliTrip'),
		($3, $2, '#MonthlyBill')
		ON CONFLICT DO NOTHING`, tagBali, uID, tagMonthly)

	// Income transactions
	mustExec(db, `INSERT INTO transactions (id, user_id, wallet_id, category_id, type, amount_minor, transaction_at, note) VALUES
		(gen_random_uuid(), $1, $2, $3, 'income', 18500000, $4, 'Monthly Salary'),
		(gen_random_uuid(), $1, $2, $5, 'income', 2500000, $4 - interval '3 days', 'Freelance Project Payment')
		ON CONFLICT DO NOTHING`,
		uID, wBank, cSalary, now.AddDate(0, 0, -5), cFreelance)

	// Expense transactions (this month)
	mustExec(db, `INSERT INTO transactions (id, user_id, wallet_id, category_id, type, amount_minor, transaction_at, note) VALUES
		(gen_random_uuid(), $1, $2, $3, 'expense', 450000, $4, 'Groceries at Indomaret'),
		(gen_random_uuid(), $1, $2, $3, 'expense', 180000, $4 - interval '1 day', 'Lunch Meeting'),
		(gen_random_uuid(), $1, $5, $6, 'expense', 350000, $4 - interval '2 days', 'Fuel and Parking'),
		(gen_random_uuid(), $1, $5, $7, 'expense', 220000, $4 - interval '3 days', 'Movie Night'),
		(gen_random_uuid(), $1, $2, $8, 'expense', 850000, $4 - interval '4 days', 'Electricity and Water Bill'),
		(gen_random_uuid(), $1, $2, $9, 'expense', 1200000, $4 - interval '5 days', 'New Running Shoes'),
		(gen_random_uuid(), $1, $2, $3, 'expense', 95000, $4 - interval '6 days', 'Coffee x5')
		ON CONFLICT DO NOTHING`,
		uID, wGoPay, cFood, now.AddDate(0, 0, -1), wCash, cTrans, cEnt, cBills, cShop)

	// Transfer
	mustExec(db, `INSERT INTO transactions (id, user_id, wallet_id, to_wallet_id, type, amount_minor, transaction_at, note) VALUES
		(gen_random_uuid(), $1, $2, $3, 'transfer', 2000000, $4, 'Topup GoPay')
		ON CONFLICT DO NOTHING`,
		uID, wBank, wGoPay, now.AddDate(0, 0, -7))

	// Budgets (current month)
	mustExec(db, `INSERT INTO category_budgets (id, user_id, category_id, month, limit_minor) VALUES
		(gen_random_uuid(), $1, $2, $3, 2000000),
		(gen_random_uuid(), $1, $4, $3, 1500000),
		(gen_random_uuid(), $1, $5, $3, 3000000),
		(gen_random_uuid(), $1, $6, $3, 5000000)
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
	mustExec(db, `INSERT INTO subscriptions (id, user_id, name, account_detail, wallet_id, category_id, amount_minor, billing_cycle, next_due_date, status, note) VALUES
		(gen_random_uuid(), $1, 'Netflix Premium', 'demo@affluena.com', $2, $3, 186000, 'monthly', $4::date, 'active', 'Family Plan') ON CONFLICT DO NOTHING`,
		uID, wBank, cEnt, now.AddDate(0, 0, 14))

	// Installment
	mustExec(db, `INSERT INTO installments (id, user_id, name, wallet_id, category_id, total_amount_minor, monthly_amount_minor, tenor_months, remaining_months, start_date, due_day, status, note) VALUES
		(gen_random_uuid(), $1, 'MacBook M5 Pro', $2, $3, 30000000, 2500000, 12, 10, $4::date, 15, 'active', '0% Cicilan') ON CONFLICT DO NOTHING`,
		uID, wBank, cShop, now.AddDate(0, -2, 0))

	// Recurring rule
	mustExec(db, `INSERT INTO recurring_transaction_rules (id, user_id, name, type, wallet_id, category_id, amount_minor, frequency, next_run_at, status) VALUES
		(gen_random_uuid(), $1, 'Spotify Family', 'expense', $2, $3, 86000, 'monthly', $4, 'active') ON CONFLICT DO NOTHING`,
		uID, wGoPay, cEnt, now.AddDate(0, 0, 5))

	// Goal + goal wallet
	mustExec(db, `INSERT INTO goals (id, user_id, name, target_amount_minor, deadline, status) VALUES
		('66666666-6666-6666-6666-666666660001', $1, 'Europe Trip 2027', 50000000, $2::date, 'active') ON CONFLICT DO NOTHING`,
		uID, now.AddDate(1, 0, 0))
	mustExec(db, `INSERT INTO wallets (id, user_id, name, type, currency_code, balance_minor, goal_id) VALUES
		(gen_random_uuid(), $1, 'Europe Trip Fund', 'goal', 'IDR', 8500000, '66666666-6666-6666-6666-666666660001') ON CONFLICT DO NOTHING`,
		uID)

	// Quick entry template
	mustExec(db, `INSERT INTO quick_entry_templates (id, user_id, name, type, wallet_id, category_id, amount_minor, note) VALUES
		(gen_random_uuid(), $1, 'Daily Coffee', 'expense', $2, $3, 35000, 'Iced Latte') ON CONFLICT DO NOTHING`,
		uID, wGoPay, cFood)

	fmt.Println("Seed complete!")
	fmt.Println()
	fmt.Println("Login credentials:")
	fmt.Printf("  Email:    %s\n", testEmail)
	fmt.Printf("  Password: %s\n", testPassword)
	fmt.Println()
	fmt.Println("Seeded data:")
	fmt.Println("  3 wallets (Cash, BCA Bank, GoPay e-wallet)")
	fmt.Println("  9 categories (3 income, 6 expense)")
	fmt.Println("  2 tags")
	fmt.Println("  10 transactions (2 income, 7 expense, 1 transfer)")
	fmt.Println("  4 budgets (Food, Transport, Entertainment, Shopping)")
	fmt.Println("  1 payable debt (BCA KTA)")
	fmt.Println("  1 subscription (Netflix)")
	fmt.Println("  1 installment (MacBook)")
	fmt.Println("  1 recurring rule (Spotify)")
	fmt.Println("  1 goal (Europe Trip)")
	fmt.Println("  1 quick entry template")
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
