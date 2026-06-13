package debt

import (
	"context"
	"os"
	"testing"
	"time"

	"affluena-api/internal/db"
	"affluena-api/internal/transaction"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDebtCreateRejectsPaymentCategoryOwnedByAnotherUser(t *testing.T) {
	pool := openIntegrationPool(t)
	ctx := context.Background()

	userA := createIntegrationUser(t, pool, "owner-a")
	userB := createIntegrationUser(t, pool, "owner-b")
	defer cleanupIntegrationUsers(t, pool, userA, userB)

	walletA := createIntegrationWallet(t, pool, userA, "Owner A wallet")
	expenseCategoryA := createIntegrationCategory(t, pool, userA, "Owner A loan", "expense")
	incomeCategoryB := createIntegrationCategory(t, pool, userB, "Owner B repayment", "income")

	repo := NewRepository(pool, transaction.NewRepository(pool))
	_, err := repo.Create(ctx, userA, DebtInput{
		Type:                   DebtTypeReceivable,
		CounterpartyName:       "Friend",
		WalletID:               walletA,
		DisbursementCategoryID: expenseCategoryA,
		PaymentCategoryID:      incomeCategoryB,
		PrincipalAmountMinor:   100_000,
		OpenedAt:               time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected create debt to reject payment category owned by another user")
	}
}

func openIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("AFFLUENA_API_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AFFLUENA_API_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := db.Migrate(ctx, pool, integrationMigrationsDir()); err != nil {
		t.Fatalf("migrate integration database: %v", err)
	}
	return pool
}

func integrationMigrationsDir() string {
	if _, err := os.Stat("migrations"); err == nil {
		return "migrations"
	}
	return "../../migrations"
}

func createIntegrationUser(t *testing.T, pool *pgxpool.Pool, label string) string {
	t.Helper()

	email := label + "-" + time.Now().UTC().Format("20060102150405.000000000") + "@example.test"
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO users (email, password_hash)
		VALUES ($1, 'integration-test')
		RETURNING id::text
	`, email).Scan(&id); err != nil {
		t.Fatalf("create integration user: %v", err)
	}
	return id
}

func createIntegrationWallet(t *testing.T, pool *pgxpool.Pool, userID string, name string) string {
	t.Helper()

	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO wallets (user_id, name, type, currency_code, balance_minor)
		VALUES ($1, $2, 'bank', 'IDR', 500000)
		RETURNING id::text
	`, userID, name).Scan(&id); err != nil {
		t.Fatalf("create integration wallet: %v", err)
	}
	return id
}

func createIntegrationCategory(t *testing.T, pool *pgxpool.Pool, userID string, name string, categoryType string) string {
	t.Helper()

	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO categories (user_id, name, type)
		VALUES ($1, $2, $3)
		RETURNING id::text
	`, userID, name, categoryType).Scan(&id); err != nil {
		t.Fatalf("create integration category: %v", err)
	}
	return id
}

func cleanupIntegrationUsers(t *testing.T, pool *pgxpool.Pool, userIDs ...string) {
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
				t.Fatalf("cleanup integration user %s: %v", userID, err)
			}
		}
	}
}
