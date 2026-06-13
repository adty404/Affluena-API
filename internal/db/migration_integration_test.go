package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOwnershipForeignKeysExist(t *testing.T) {
	pool := openMigrationIntegrationPool(t)
	ctx := context.Background()

	expectedConstraints := []string{
		"transactions_user_wallet_fk",
		"transactions_user_to_wallet_fk",
		"transactions_user_category_fk",
		"quick_entry_templates_user_wallet_fk",
		"quick_entry_templates_user_to_wallet_fk",
		"quick_entry_templates_user_category_fk",
		"category_budgets_user_category_fk",
		"installments_user_wallet_fk",
		"installments_user_category_fk",
		"subscriptions_user_wallet_fk",
		"subscriptions_user_category_fk",
		"recurring_rules_user_wallet_fk",
		"recurring_rules_user_to_wallet_fk",
		"recurring_rules_user_category_fk",
		"recurring_runs_user_rule_fk",
		"recurring_runs_user_transaction_fk",
		"debts_user_wallet_fk",
		"debts_user_disbursement_category_fk",
		"debts_user_payment_category_fk",
		"debts_user_origination_transaction_fk",
		"debt_payments_user_debt_fk",
		"debt_payments_user_transaction_fk",
	}

	for _, name := range expectedConstraints {
		t.Run(name, func(t *testing.T) {
			var exists bool
			err := pool.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1
					FROM pg_constraint
					WHERE conname = $1
						AND contype = 'f'
						AND cardinality(conkey) = 2
						AND cardinality(confkey) = 2
				)
			`, name).Scan(&exists)
			if err != nil {
				t.Fatalf("query constraint %s: %v", name, err)
			}
			if !exists {
				t.Fatalf("expected composite ownership foreign key %s to exist", name)
			}
		})
	}
}

func TestOwnershipForeignKeysRejectCrossUserReferences(t *testing.T) {
	pool := openMigrationIntegrationPool(t)
	ctx := context.Background()
	fixture := createOwnershipFixture(t, pool)
	defer cleanupOwnershipFixture(t, pool, fixture.userA, fixture.userB)

	attempts := []struct {
		name string
		sql  string
		args []any
	}{
		{
			name: "transaction cannot use another user's wallet",
			sql:  `INSERT INTO transactions (user_id, type, wallet_id, category_id, amount_minor, transaction_at) VALUES ($1, 'income', $2, $3, 1000, now())`,
			args: []any{fixture.userA, fixture.walletB, fixture.incomeCategoryA},
		},
		{
			name: "transaction cannot use another user's category",
			sql:  `INSERT INTO transactions (user_id, type, wallet_id, category_id, amount_minor, transaction_at) VALUES ($1, 'income', $2, $3, 1000, now())`,
			args: []any{fixture.userA, fixture.walletA, fixture.incomeCategoryB},
		},
		{
			name: "quick entry cannot use another user's destination wallet",
			sql:  `INSERT INTO quick_entry_templates (user_id, name, type, wallet_id, to_wallet_id, amount_minor) VALUES ($1, 'Cross transfer', 'transfer', $2, $3, 1000)`,
			args: []any{fixture.userA, fixture.walletA, fixture.walletB},
		},
		{
			name: "budget cannot use another user's category",
			sql:  `INSERT INTO category_budgets (user_id, category_id, month, limit_minor) VALUES ($1, $2, '2026-06-01', 1000)`,
			args: []any{fixture.userA, fixture.expenseCategoryB},
		},
		{
			name: "installment cannot use another user's wallet",
			sql:  `INSERT INTO installments (user_id, name, wallet_id, category_id, total_amount_minor, monthly_amount_minor, tenor_months, remaining_months, start_date, due_day, status) VALUES ($1, 'Cross installment', $2, $3, 3000, 1000, 3, 3, '2026-06-01', 5, 'active')`,
			args: []any{fixture.userA, fixture.walletB, fixture.expenseCategoryA},
		},
		{
			name: "subscription cannot use another user's category",
			sql:  `INSERT INTO subscriptions (user_id, name, wallet_id, category_id, amount_minor, billing_cycle, next_due_date, status) VALUES ($1, 'Cross subscription', $2, $3, 1000, 'monthly', '2026-07-01', 'active')`,
			args: []any{fixture.userA, fixture.walletA, fixture.expenseCategoryB},
		},
		{
			name: "recurring rule cannot use another user's wallet",
			sql:  `INSERT INTO recurring_transaction_rules (user_id, name, type, wallet_id, category_id, amount_minor, frequency, interval_count, next_run_at, status) VALUES ($1, 'Cross recurring', 'expense', $2, $3, 1000, 'monthly', 1, '2030-01-01T00:00:00Z', 'active')`,
			args: []any{fixture.userA, fixture.walletB, fixture.expenseCategoryA},
		},
		{
			name: "debt cannot use another user's payment category",
			sql:  `INSERT INTO debts (user_id, type, counterparty_name, wallet_id, disbursement_category_id, payment_category_id, origination_transaction_id, principal_amount_minor, opened_at, status) VALUES ($1, 'receivable', 'Cross debt', $2, $3, $4, $5, 1000, now(), 'open')`,
			args: []any{fixture.userA, fixture.walletA, fixture.expenseCategoryA, fixture.incomeCategoryB, fixture.transactionA},
		},
		{
			name: "debt cannot use another user's origination transaction",
			sql:  `INSERT INTO debts (user_id, type, counterparty_name, wallet_id, disbursement_category_id, payment_category_id, origination_transaction_id, principal_amount_minor, opened_at, status) VALUES ($1, 'receivable', 'Cross debt tx', $2, $3, $4, $5, 1000, now(), 'open')`,
			args: []any{fixture.userA, fixture.walletA, fixture.expenseCategoryA, fixture.incomeCategoryA, fixture.transactionB},
		},
		{
			name: "debt payment cannot use another user's debt",
			sql:  `INSERT INTO debt_payments (user_id, debt_id, transaction_id, amount_minor, paid_at) VALUES ($1, $2, $3, 1000, now())`,
			args: []any{fixture.userA, fixture.debtB, fixture.transactionA},
		},
		{
			name: "debt payment cannot use another user's transaction",
			sql:  `INSERT INTO debt_payments (user_id, debt_id, transaction_id, amount_minor, paid_at) VALUES ($1, $2, $3, 1000, now())`,
			args: []any{fixture.userA, fixture.debtA, fixture.transactionB},
		},
	}

	for _, attempt := range attempts {
		t.Run(attempt.name, func(t *testing.T) {
			if _, err := pool.Exec(ctx, attempt.sql, attempt.args...); err == nil {
				t.Fatal("expected cross-user reference to be rejected")
			}
		})
	}
}

func openMigrationIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("AFFLUENA_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AFFLUENA_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := Migrate(ctx, pool, integrationMigrationsDir()); err != nil {
		t.Fatalf("migrate integration database: %v", err)
	}
	return pool
}

type ownershipFixture struct {
	userA            string
	userB            string
	walletA          string
	walletB          string
	incomeCategoryA  string
	incomeCategoryB  string
	expenseCategoryA string
	expenseCategoryB string
	transactionA     string
	transactionB     string
	debtA            string
	debtB            string
}

func createOwnershipFixture(t *testing.T, pool *pgxpool.Pool) ownershipFixture {
	t.Helper()
	ctx := context.Background()
	suffix := time.Now().UTC().Format("20060102150405.000000000")
	var f ownershipFixture

	f.userA = insertFixtureUser(t, pool, "db-owner-a-"+suffix+"@example.test")
	f.userB = insertFixtureUser(t, pool, "db-owner-b-"+suffix+"@example.test")
	f.walletA = insertFixtureWallet(t, pool, f.userA, "Owner A wallet")
	f.walletB = insertFixtureWallet(t, pool, f.userB, "Owner B wallet")
	f.incomeCategoryA = insertFixtureCategory(t, pool, f.userA, "Owner A income", "income")
	f.incomeCategoryB = insertFixtureCategory(t, pool, f.userB, "Owner B income", "income")
	f.expenseCategoryA = insertFixtureCategory(t, pool, f.userA, "Owner A expense", "expense")
	f.expenseCategoryB = insertFixtureCategory(t, pool, f.userB, "Owner B expense", "expense")

	if err := pool.QueryRow(ctx, `
		INSERT INTO transactions (user_id, type, wallet_id, category_id, amount_minor, transaction_at)
		VALUES ($1, 'income', $2, $3, 1000, now())
		RETURNING id::text
	`, f.userA, f.walletA, f.incomeCategoryA).Scan(&f.transactionA); err != nil {
		t.Fatalf("insert fixture transaction A: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO transactions (user_id, type, wallet_id, category_id, amount_minor, transaction_at)
		VALUES ($1, 'income', $2, $3, 1000, now())
		RETURNING id::text
	`, f.userB, f.walletB, f.incomeCategoryB).Scan(&f.transactionB); err != nil {
		t.Fatalf("insert fixture transaction B: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO debts (user_id, type, counterparty_name, wallet_id, disbursement_category_id, payment_category_id, origination_transaction_id, principal_amount_minor, opened_at, status)
		VALUES ($1, 'receivable', 'Owner A debt', $2, $3, $4, $5, 1000, now(), 'open')
		RETURNING id::text
	`, f.userA, f.walletA, f.expenseCategoryA, f.incomeCategoryA, f.transactionA).Scan(&f.debtA); err != nil {
		t.Fatalf("insert fixture debt A: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO debts (user_id, type, counterparty_name, wallet_id, disbursement_category_id, payment_category_id, origination_transaction_id, principal_amount_minor, opened_at, status)
		VALUES ($1, 'receivable', 'Owner B debt', $2, $3, $4, $5, 1000, now(), 'open')
		RETURNING id::text
	`, f.userB, f.walletB, f.expenseCategoryB, f.incomeCategoryB, f.transactionB).Scan(&f.debtB); err != nil {
		t.Fatalf("insert fixture debt B: %v", err)
	}

	return f
}

func insertFixtureUser(t *testing.T, pool *pgxpool.Pool, email string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `INSERT INTO users (email, password_hash) VALUES ($1, 'integration-test') RETURNING id::text`, email).Scan(&id); err != nil {
		t.Fatalf("insert fixture user: %v", err)
	}
	return id
}

func insertFixtureWallet(t *testing.T, pool *pgxpool.Pool, userID string, name string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `INSERT INTO wallets (user_id, name, type, currency_code, balance_minor) VALUES ($1, $2, 'bank', 'IDR', 0) RETURNING id::text`, userID, name).Scan(&id); err != nil {
		t.Fatalf("insert fixture wallet: %v", err)
	}
	return id
}

func insertFixtureCategory(t *testing.T, pool *pgxpool.Pool, userID string, name string, categoryType string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `INSERT INTO categories (user_id, name, type) VALUES ($1, $2, $3) RETURNING id::text`, userID, name, categoryType).Scan(&id); err != nil {
		t.Fatalf("insert fixture category: %v", err)
	}
	return id
}

func cleanupOwnershipFixture(t *testing.T, pool *pgxpool.Pool, userIDs ...string) {
	t.Helper()
	statements := []string{
		`DELETE FROM debt_payments WHERE user_id = $1`,
		`DELETE FROM debts WHERE user_id = $1`,
		`DELETE FROM recurring_transaction_runs WHERE user_id = $1`,
		`DELETE FROM recurring_transaction_rules WHERE user_id = $1`,
		`DELETE FROM quick_entry_templates WHERE user_id = $1`,
		`DELETE FROM category_budgets WHERE user_id = $1`,
		`DELETE FROM installments WHERE user_id = $1`,
		`DELETE FROM subscriptions WHERE user_id = $1`,
		`DELETE FROM transactions WHERE user_id = $1`,
		`DELETE FROM wallets WHERE user_id = $1`,
		`DELETE FROM categories WHERE user_id = $1`,
		`DELETE FROM refresh_tokens WHERE user_id = $1`,
		`DELETE FROM users WHERE id = $1`,
	}
	for _, userID := range userIDs {
		for _, statement := range statements {
			if _, err := pool.Exec(context.Background(), statement, userID); err != nil {
				t.Fatalf("cleanup ownership fixture %s: %v", userID, err)
			}
		}
	}
}

func integrationMigrationsDir() string {
	if _, err := os.Stat("migrations"); err == nil {
		return "migrations"
	}
	return "../../migrations"
}
