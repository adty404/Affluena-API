package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"affluena/internal/config"
	"affluena/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUsersCannotReadOrMutateEachOthersWalletsCategoriesAndTransactions(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "integration-test-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userAID, userAToken := registerIntegrationAPIUser(t, router, "api-owner-a")
	userBID, userBToken := registerIntegrationAPIUser(t, router, "api-owner-b")
	defer cleanupServerIntegrationUsers(t, pool, userAID, userBID)

	walletAID := createAPIResource(t, router, userAToken, "/api/v1/wallets", `{
		"name": "Owner A wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 100000
	}`)
	toWalletAID := createAPIResource(t, router, userAToken, "/api/v1/wallets", `{
		"name": "Owner A destination wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 0
	}`)
	categoryAID := createAPIResource(t, router, userAToken, "/api/v1/categories", `{
		"name": "Owner A income",
		"type": "income"
	}`)
	expenseCategoryAID := createAPIResource(t, router, userAToken, "/api/v1/categories", `{
		"name": "Owner A expense",
		"type": "expense"
	}`)

	assertResourceHiddenFromOtherUser(t, router, userAToken, userBToken, "/api/v1/wallets", "/api/v1/wallets/"+walletAID, walletAID, `{
		"name": "Stolen wallet",
		"type": "cash",
		"currency_code": "IDR"
	}`)

	assertResourceHiddenFromOtherUser(t, router, userAToken, userBToken, "/api/v1/categories", "/api/v1/categories/"+categoryAID, categoryAID, `{
		"name": "Stolen category",
		"type": "income"
	}`)

	assertAPIStatus(t, router, userBToken, http.MethodPost, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+walletAID+`",
		"category_id": "`+categoryAID+`",
		"amount_minor": 50000,
		"transaction_at": "2026-06-13T08:00:00Z"
	}`, http.StatusNotFound)

	transactionAID := createAPIResource(t, router, userAToken, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+walletAID+`",
		"category_id": "`+categoryAID+`",
		"amount_minor": 50000,
		"transaction_at": "2026-06-13T08:00:00Z"
	}`)
	assertResourceHiddenFromOtherUser(t, router, userAToken, userBToken, "/api/v1/transactions", "/api/v1/transactions/"+transactionAID, transactionAID, `{
		"type": "income",
		"wallet_id": "`+walletAID+`",
		"category_id": "`+categoryAID+`",
		"amount_minor": 60000,
		"transaction_at": "2026-06-13T08:00:00Z"
	}`)

	quickEntryAID := createAPIResource(t, router, userAToken, "/api/v1/quick-entry-templates", `{
		"name": "Owner A quick income",
		"type": "income",
		"wallet_id": "`+walletAID+`",
		"category_id": "`+categoryAID+`",
		"amount_minor": 75000,
		"note": "Quick salary"
	}`)
	assertResourceHiddenFromOtherUser(t, router, userAToken, userBToken, "/api/v1/quick-entry-templates", "/api/v1/quick-entry-templates/"+quickEntryAID, quickEntryAID, `{
		"name": "Stolen quick income",
		"type": "income",
		"wallet_id": "`+walletAID+`",
		"category_id": "`+categoryAID+`",
		"amount_minor": 75000
	}`)
	assertAPIStatus(t, router, userBToken, http.MethodPost, "/api/v1/quick-entry-templates/"+quickEntryAID+"/execute", `{}`, http.StatusNotFound)
	assertAPIStatus(t, router, userBToken, http.MethodPost, "/api/v1/quick-entry-templates", `{
		"name": "Cross owner quick",
		"type": "income",
		"wallet_id": "`+walletAID+`",
		"category_id": "`+categoryAID+`",
		"amount_minor": 75000
	}`, http.StatusBadRequest)

	budgetAID := createAPIResource(t, router, userAToken, "/api/v1/category-budgets", `{
		"category_id": "`+expenseCategoryAID+`",
		"month": "2026-06",
		"limit_minor": 200000
	}`)
	assertResourceHiddenFromOtherUser(t, router, userAToken, userBToken, "/api/v1/category-budgets?month=2026-06", "/api/v1/category-budgets/"+budgetAID, budgetAID, `{
		"category_id": "`+expenseCategoryAID+`",
		"month": "2026-06",
		"limit_minor": 250000
	}`)
	assertAPIStatus(t, router, userBToken, http.MethodPost, "/api/v1/category-budgets", `{
		"category_id": "`+expenseCategoryAID+`",
		"month": "2026-07",
		"limit_minor": 200000
	}`, http.StatusNotFound)

	debtAID := createAPIResource(t, router, userAToken, "/api/v1/debts", `{
		"type": "receivable",
		"counterparty_name": "Friend",
		"wallet_id": "`+walletAID+`",
		"disbursement_category_id": "`+expenseCategoryAID+`",
		"payment_category_id": "`+categoryAID+`",
		"principal_amount_minor": 100000,
		"opened_at": "2026-06-13T08:00:00Z",
		"due_date": "2026-07-01"
	}`)
	assertResourceHiddenFromOtherUser(t, router, userAToken, userBToken, "/api/v1/debts", "/api/v1/debts/"+debtAID, debtAID, `{
		"counterparty_name": "Stolen Friend",
		"due_date": "2026-07-01",
		"status": "open"
	}`)
	assertAPIStatus(t, router, userBToken, http.MethodPost, "/api/v1/debts/"+debtAID+"/pay", `{
		"amount_minor": 10000,
		"paid_at": "2026-06-14T08:00:00Z"
	}`, http.StatusNotFound)
	assertAPIStatus(t, router, userBToken, http.MethodPost, "/api/v1/debts", `{
		"type": "receivable",
		"counterparty_name": "Cross Friend",
		"wallet_id": "`+walletAID+`",
		"disbursement_category_id": "`+expenseCategoryAID+`",
		"payment_category_id": "`+categoryAID+`",
		"principal_amount_minor": 100000
	}`, http.StatusNotFound)

	installmentAID := createAPIResource(t, router, userAToken, "/api/v1/installments", `{
		"name": "Owner A laptop",
		"wallet_id": "`+walletAID+`",
		"category_id": "`+expenseCategoryAID+`",
		"total_amount_minor": 600000,
		"monthly_amount_minor": 200000,
		"tenor_months": 3,
		"start_date": "2026-06-01",
		"due_day": 5
	}`)
	assertResourceHiddenFromOtherUser(t, router, userAToken, userBToken, "/api/v1/installments", "/api/v1/installments/"+installmentAID, installmentAID, `{
		"name": "Stolen laptop",
		"wallet_id": "`+walletAID+`",
		"category_id": "`+expenseCategoryAID+`",
		"total_amount_minor": 600000,
		"monthly_amount_minor": 200000,
		"tenor_months": 3,
		"remaining_months": 3,
		"start_date": "2026-06-01",
		"due_day": 5,
		"status": "active"
	}`)
	assertAPIStatus(t, router, userBToken, http.MethodPost, "/api/v1/installments/"+installmentAID+"/pay", `{}`, http.StatusNotFound)
	assertAPIStatus(t, router, userBToken, http.MethodPost, "/api/v1/installments", `{
		"name": "Cross laptop",
		"wallet_id": "`+walletAID+`",
		"category_id": "`+expenseCategoryAID+`",
		"total_amount_minor": 600000,
		"monthly_amount_minor": 200000,
		"tenor_months": 3,
		"start_date": "2026-06-01",
		"due_day": 5
	}`, http.StatusNotFound)

	subscriptionAID := createAPIResource(t, router, userAToken, "/api/v1/subscriptions", `{
		"name": "Owner A internet",
		"wallet_id": "`+walletAID+`",
		"category_id": "`+expenseCategoryAID+`",
		"amount_minor": 250000,
		"billing_cycle": "monthly",
		"next_due_date": "2026-07-01"
	}`)
	assertResourceHiddenFromOtherUser(t, router, userAToken, userBToken, "/api/v1/subscriptions", "/api/v1/subscriptions/"+subscriptionAID, subscriptionAID, `{
		"name": "Stolen internet",
		"wallet_id": "`+walletAID+`",
		"category_id": "`+expenseCategoryAID+`",
		"amount_minor": 250000,
		"billing_cycle": "monthly",
		"next_due_date": "2026-07-01",
		"status": "active"
	}`)
	assertAPIStatus(t, router, userBToken, http.MethodPost, "/api/v1/subscriptions/"+subscriptionAID+"/pay", `{}`, http.StatusNotFound)
	assertAPIStatus(t, router, userBToken, http.MethodPost, "/api/v1/subscriptions", `{
		"name": "Cross internet",
		"wallet_id": "`+walletAID+`",
		"category_id": "`+expenseCategoryAID+`",
		"amount_minor": 250000,
		"billing_cycle": "monthly",
		"next_due_date": "2026-07-01"
	}`, http.StatusNotFound)

	recurringAID := createAPIResource(t, router, userAToken, "/api/v1/recurring-transactions", `{
		"name": "Owner A rent",
		"type": "expense",
		"wallet_id": "`+walletAID+`",
		"category_id": "`+expenseCategoryAID+`",
		"amount_minor": 350000,
		"frequency": "monthly",
		"interval_count": 1,
		"next_run_at": "2030-01-01T00:00:00Z"
	}`)
	assertResourceHiddenFromOtherUser(t, router, userAToken, userBToken, "/api/v1/recurring-transactions", "/api/v1/recurring-transactions/"+recurringAID, recurringAID, `{
		"name": "Stolen rent",
		"type": "expense",
		"wallet_id": "`+walletAID+`",
		"category_id": "`+expenseCategoryAID+`",
		"amount_minor": 350000,
		"frequency": "monthly",
		"interval_count": 1,
		"next_run_at": "2030-02-01T00:00:00Z",
		"status": "active"
	}`)
	assertAPIStatus(t, router, userBToken, http.MethodPost, "/api/v1/recurring-transactions/"+recurringAID+"/run", `{
		"now": "2030-01-01T00:00:00Z"
	}`, http.StatusNotFound)
	assertAPIStatus(t, router, userBToken, http.MethodPost, "/api/v1/recurring-transactions", `{
		"name": "Cross transfer",
		"type": "transfer",
		"wallet_id": "`+walletAID+`",
		"to_wallet_id": "`+toWalletAID+`",
		"amount_minor": 10000,
		"frequency": "monthly",
		"interval_count": 1,
		"next_run_at": "2030-01-01T00:00:00Z"
	}`, http.StatusNotFound)
}

func openServerIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("AFFLUENA_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AFFLUENA_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := db.Migrate(ctx, pool, serverIntegrationMigrationsDir()); err != nil {
		t.Fatalf("migrate integration database: %v", err)
	}
	return pool
}

func registerIntegrationAPIUser(t *testing.T, router http.Handler, label string) (string, string) {
	t.Helper()

	email := label + "-" + time.Now().UTC().Format("20060102150405.000000000") + "@example.test"
	body := `{"email":"` + email + `","password":"super-secret-password"}`
	response := performAPIRequest(t, router, "", http.MethodPost, "/api/v1/auth/register", body, http.StatusCreated)

	var parsed struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(response, &parsed); err != nil {
		t.Fatalf("parse register response: %v", err)
	}
	if parsed.User.ID == "" || parsed.Tokens.AccessToken == "" {
		t.Fatalf("register response missing user id or access token: %s", string(response))
	}
	return parsed.User.ID, parsed.Tokens.AccessToken
}

func createAPIResource(t *testing.T, router http.Handler, token string, path string, body string) string {
	t.Helper()

	response := performAPIRequest(t, router, token, http.MethodPost, path, body, http.StatusCreated)
	var parsed struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response, &parsed); err != nil {
		t.Fatalf("parse create response for %s: %v", path, err)
	}
	if parsed.ID == "" {
		t.Fatalf("create response for %s missing id: %s", path, string(response))
	}
	return parsed.ID
}

func assertResponseDoesNotContain(t *testing.T, router http.Handler, token string, method string, path string, body string, forbidden string) {
	t.Helper()

	response := performAPIRequest(t, router, token, method, path, body, http.StatusOK)
	if strings.Contains(string(response), forbidden) {
		t.Fatalf("response for %s %s leaked forbidden id %s: %s", method, path, forbidden, string(response))
	}
}

func assertResourceHiddenFromOtherUser(t *testing.T, router http.Handler, ownerToken string, otherToken string, listPath string, itemPath string, resourceID string, updateBody string) {
	t.Helper()

	assertResponseDoesNotContain(t, router, otherToken, http.MethodGet, listPath, "", resourceID)
	assertAPIStatus(t, router, otherToken, http.MethodGet, itemPath, "", http.StatusNotFound)
	assertAPIStatus(t, router, otherToken, http.MethodPut, itemPath, updateBody, http.StatusNotFound)
	assertAPIStatus(t, router, otherToken, http.MethodDelete, itemPath, "", http.StatusNotFound)
	assertAPIStatus(t, router, ownerToken, http.MethodGet, itemPath, "", http.StatusOK)
}

func assertAPIStatus(t *testing.T, router http.Handler, token string, method string, path string, body string, wantStatus int) {
	t.Helper()
	performAPIRequest(t, router, token, method, path, body, wantStatus)
}

func performAPIRequest(t *testing.T, router http.Handler, token string, method string, path string, body string, wantStatus int) []byte {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != wantStatus {
		t.Fatalf("%s %s expected status %d, got %d: %s", method, path, wantStatus, recorder.Code, recorder.Body.String())
	}
	return recorder.Body.Bytes()
}

func cleanupServerIntegrationUsers(t *testing.T, pool *pgxpool.Pool, userIDs ...string) {
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

func serverIntegrationMigrationsDir() string {
	if _, err := os.Stat("migrations"); err == nil {
		return "migrations"
	}
	return "../../migrations"
}
