package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"affluena-api/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestCreateUserSeedsOnboardingDefaultsInOneTransaction proves the green path
// at the repository level: one CreateUser call leaves the user row plus its 8
// default categories and starter wallet.
func TestCreateUserSeedsOnboardingDefaultsInOneTransaction(t *testing.T) {
	pool := openAuthIntegrationPool(t)
	ctx := context.Background()
	repo := NewRepository(pool)

	email := integrationEmail("auth-onboarding-green")
	user, err := repo.CreateUser(ctx, email, "integration-test-hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	defer cleanupAuthIntegrationUser(t, pool, user.ID)

	var categories int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM categories WHERE user_id = $1`, user.ID).Scan(&categories); err != nil {
		t.Fatalf("count categories: %v", err)
	}
	if categories != len(defaultCategories) {
		t.Fatalf("expected %d default categories, got %d", len(defaultCategories), categories)
	}

	var wallets int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM wallets WHERE user_id = $1 AND name = $2 AND type = $3`, user.ID, defaultWalletName, defaultWalletType).Scan(&wallets); err != nil {
		t.Fatalf("count wallets: %v", err)
	}
	if wallets != 1 {
		t.Fatalf("expected exactly one starter wallet, got %d", wallets)
	}
}

// TestCreateUserRollsBackUserRowWhenOnboardingSeedFails injects a failure into
// the onboarding seed step and proves atomicity: when the defaults cannot be
// written, the user row must roll back with them — a half-onboarded account
// (user without defaults) must never exist.
func TestCreateUserRollsBackUserRowWhenOnboardingSeedFails(t *testing.T) {
	pool := openAuthIntegrationPool(t)
	ctx := context.Background()

	repo := NewRepository(pool)
	repo.seedDefaults = func(ctx context.Context, tx pgx.Tx, userID string) error {
		return errors.New("boom: injected onboarding failure")
	}

	email := integrationEmail("auth-onboarding-rollback")
	if _, err := repo.CreateUser(ctx, email, "integration-test-hash"); err == nil {
		t.Fatal("expected CreateUser to fail when onboarding seed fails")
	}

	var users int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE email = $1`, email).Scan(&users); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if users != 0 {
		t.Fatalf("expected no orphan user row after failed onboarding, got %d", users)
	}
}

func openAuthIntegrationPool(t *testing.T) *pgxpool.Pool {
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

	if err := db.Migrate(ctx, pool, authIntegrationMigrationsDir()); err != nil {
		t.Fatalf("migrate integration database: %v", err)
	}
	return pool
}

func authIntegrationMigrationsDir() string {
	if _, err := os.Stat("migrations"); err == nil {
		return "migrations"
	}
	return "../../migrations"
}

func integrationEmail(label string) string {
	return label + "-" + time.Now().UTC().Format("20060102150405.000000000") + "@example.test"
}

func cleanupAuthIntegrationUser(t *testing.T, pool *pgxpool.Pool, userID string) {
	t.Helper()
	// users cascades to categories/wallets/refresh_tokens via ON DELETE CASCADE.
	if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID); err != nil {
		t.Fatalf("cleanup auth integration user: %v", err)
	}
}
