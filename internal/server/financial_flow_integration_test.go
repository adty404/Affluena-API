package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
)

func TestTransactionLifecycleMaintainsWalletBalances(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "financial-flow-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "flow-owner")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	mainWalletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Main wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 100000
	}`)
	cashWalletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Cash wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 0
	}`)
	incomeCategoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Salary",
		"type": "income"
	}`)
	expenseCategoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Food",
		"type": "expense"
	}`)

	assertWalletBalance(t, router, token, mainWalletID, 100000)
	assertWalletBalance(t, router, token, cashWalletID, 0)

	incomeID := createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+mainWalletID+`",
		"category_id": "`+incomeCategoryID+`",
		"amount_minor": 50000,
		"transaction_at": "2026-06-13T08:00:00Z"
	}`)
	assertWalletBalance(t, router, token, mainWalletID, 150000)

	createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+mainWalletID+`",
		"category_id": "`+expenseCategoryID+`",
		"amount_minor": 20000,
		"transaction_at": "2026-06-13T09:00:00Z"
	}`)
	assertWalletBalance(t, router, token, mainWalletID, 130000)

	transferID := createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "transfer",
		"wallet_id": "`+mainWalletID+`",
		"to_wallet_id": "`+cashWalletID+`",
		"amount_minor": 30000,
		"transaction_at": "2026-06-13T10:00:00Z"
	}`)
	assertWalletBalance(t, router, token, mainWalletID, 100000)
	assertWalletBalance(t, router, token, cashWalletID, 30000)

	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/transactions/"+transferID, `{
		"type": "transfer",
		"wallet_id": "`+mainWalletID+`",
		"to_wallet_id": "`+cashWalletID+`",
		"amount_minor": 10000,
		"transaction_at": "2026-06-13T10:00:00Z"
	}`, http.StatusOK)
	assertWalletBalance(t, router, token, mainWalletID, 120000)
	assertWalletBalance(t, router, token, cashWalletID, 10000)

	assertAPIStatus(t, router, token, http.MethodPost, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+mainWalletID+`",
		"category_id": "00000000-0000-0000-0000-000000000000",
		"amount_minor": 999999,
		"transaction_at": "2026-06-13T11:00:00Z"
	}`, http.StatusNotFound)
	assertWalletBalance(t, router, token, mainWalletID, 120000)
	assertWalletBalance(t, router, token, cashWalletID, 10000)

	assertAPIStatus(t, router, token, http.MethodDelete, "/api/v1/transactions/"+incomeID, "", http.StatusNoContent)
	assertWalletBalance(t, router, token, mainWalletID, 70000)
	assertWalletBalance(t, router, token, cashWalletID, 10000)
}

func assertWalletBalance(t *testing.T, router http.Handler, token string, walletID string, wantBalance int64) {
	t.Helper()

	response := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/wallets/"+walletID, "", http.StatusOK)
	var wallet struct {
		ID           string `json:"id"`
		BalanceMinor int64  `json:"balance_minor"`
	}
	if err := json.Unmarshal(response, &wallet); err != nil {
		t.Fatalf("parse wallet response: %v", err)
	}
	if wallet.ID != walletID {
		t.Fatalf("expected wallet %s, got %s", walletID, wallet.ID)
	}
	if wallet.BalanceMinor != wantBalance {
		t.Fatalf("expected wallet %s balance %d, got %d", walletID, wantBalance, wallet.BalanceMinor)
	}
}
