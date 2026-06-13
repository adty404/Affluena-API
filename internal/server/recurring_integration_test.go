package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
	"affluena-api/internal/recurring"
)

func TestRecurringManualRunCreatesTransactionAndUpdatesWallet(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "recurring-manual-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "recurring-manual")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Recurring wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 1000000
	}`)
	categoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Internet",
		"type": "expense"
	}`)
	ruleID := createAPIResource(t, router, token, "/api/v1/recurring-transactions", `{
		"name": "Monthly internet",
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 250000,
		"frequency": "monthly",
		"interval_count": 1,
		"next_run_at": "2026-06-01T00:00:00Z"
	}`)

	body := performAPIRequest(t, router, token, http.MethodPost, "/api/v1/recurring-transactions/"+ruleID+"/run", `{
		"now": "2026-06-13T00:00:00Z"
	}`, http.StatusCreated)

	var run recurring.Run
	if err := json.Unmarshal(body, &run); err != nil {
		t.Fatalf("parse recurring run response: %v", err)
	}
	if run.ID == "" || run.TransactionID == "" || run.Transaction.ID == "" {
		t.Fatalf("expected run and transaction IDs to be populated: %s", string(body))
	}
	if run.Rule.ID != ruleID {
		t.Fatalf("expected updated rule %s, got %s", ruleID, run.Rule.ID)
	}
	if got := run.Rule.NextRunAt.UTC().Format(time.RFC3339); got != "2026-07-01T00:00:00Z" {
		t.Fatalf("expected next_run_at to advance to 2026-07-01T00:00:00Z, got %s", got)
	}
	assertWalletBalance(t, router, token, walletID, 750000)
}
