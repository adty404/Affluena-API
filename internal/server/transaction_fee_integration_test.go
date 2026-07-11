package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
)

// TestTransferAdminFeeMaintainsBalances proves the optional per-transfer bank
// admin fee (fee_minor) is charged atomically to the source wallet on top of
// the amount, is net-worth-correct (sum of balances drops by exactly the fee),
// and stays exact across create, edit (fee, amount, and source-wallet changes),
// and delete. It also guards the validation scope: fee is transfer-only and
// never negative.
func TestTransferAdminFeeMaintainsBalances(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "transfer-fee-integration-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "fee-owner")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	walletA := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Wallet A", "type": "bank", "currency_code": "IDR", "balance_minor": 500000
	}`)
	walletB := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Wallet B", "type": "cash", "currency_code": "IDR", "balance_minor": 0
	}`)
	walletC := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Wallet C", "type": "bank", "currency_code": "IDR", "balance_minor": 300000
	}`)
	expenseCategory := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Food", "type": "expense"
	}`)

	// Scenario 1: transfer amount=100000 fee=2500 A -> B.
	// A drops 102500, B rises 100000, sum-of-balances drops 2500.
	createResponse := performAPIRequest(t, router, token, http.MethodPost, "/api/v1/transactions", `{
		"type": "transfer",
		"wallet_id": "`+walletA+`",
		"to_wallet_id": "`+walletB+`",
		"amount_minor": 100000,
		"fee_minor": 2500,
		"transaction_at": "2026-06-13T10:00:00Z"
	}`, http.StatusCreated)
	var created struct {
		ID       string `json:"id"`
		FeeMinor int64  `json:"fee_minor"`
	}
	if err := json.Unmarshal(createResponse, &created); err != nil {
		t.Fatalf("parse transfer create response: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("transfer create response missing id: %s", string(createResponse))
	}
	if created.FeeMinor != 2500 {
		t.Fatalf("expected create response fee_minor 2500, got %d", created.FeeMinor)
	}
	transferID := created.ID

	assertWalletBalance(t, router, token, walletA, 397500) // 500000 - 102500
	assertWalletBalance(t, router, token, walletB, 100000)
	assertWalletBalance(t, router, token, walletC, 300000)
	assertSumOfBalances(t, router, token, []string{walletA, walletB, walletC}, 797500) // 800000 - 2500

	// The detail response must also echo fee_minor.
	assertTransactionFee(t, router, token, transferID, 2500)

	// Scenario 2a: edit fee 2500 -> 5000 (amount unchanged). Only A adjusts by
	// the 2500 diff; B is untouched.
	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/transactions/"+transferID, `{
		"type": "transfer",
		"wallet_id": "`+walletA+`",
		"to_wallet_id": "`+walletB+`",
		"amount_minor": 100000,
		"fee_minor": 5000,
		"transaction_at": "2026-06-13T10:00:00Z"
	}`, http.StatusOK)
	assertWalletBalance(t, router, token, walletA, 395000) // 500000 - 105000
	assertWalletBalance(t, router, token, walletB, 100000)
	assertTransactionFee(t, router, token, transferID, 5000)

	// Scenario 2b: edit amount 100000 -> 80000 (fee unchanged at 5000).
	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/transactions/"+transferID, `{
		"type": "transfer",
		"wallet_id": "`+walletA+`",
		"to_wallet_id": "`+walletB+`",
		"amount_minor": 80000,
		"fee_minor": 5000,
		"transaction_at": "2026-06-13T10:00:00Z"
	}`, http.StatusOK)
	assertWalletBalance(t, router, token, walletA, 415000) // 500000 - 85000
	assertWalletBalance(t, router, token, walletB, 80000)

	// Scenario 2c: change the source wallet A -> C (amount 80000, fee 5000).
	// Old source A is fully refunded amount+fee; new source C is charged
	// amount+fee; B unchanged.
	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/transactions/"+transferID, `{
		"type": "transfer",
		"wallet_id": "`+walletC+`",
		"to_wallet_id": "`+walletB+`",
		"amount_minor": 80000,
		"fee_minor": 5000,
		"transaction_at": "2026-06-13T10:00:00Z"
	}`, http.StatusOK)
	assertWalletBalance(t, router, token, walletA, 500000) // fully restored
	assertWalletBalance(t, router, token, walletC, 215000) // 300000 - 85000
	assertWalletBalance(t, router, token, walletB, 80000)

	// Scenario 3: delete the transfer -> source C and dest B restored to their
	// pre-transfer balances exactly.
	assertAPIStatus(t, router, token, http.MethodDelete, "/api/v1/transactions/"+transferID, "", http.StatusNoContent)
	assertWalletBalance(t, router, token, walletA, 500000)
	assertWalletBalance(t, router, token, walletB, 0)
	assertWalletBalance(t, router, token, walletC, 300000)
	assertSumOfBalances(t, router, token, []string{walletA, walletB, walletC}, 800000)

	// Scenario 4: fee_minor on a non-transfer type -> 400; negative fee -> 400.
	assertAPIStatus(t, router, token, http.MethodPost, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletA+`",
		"category_id": "`+expenseCategory+`",
		"amount_minor": 10000,
		"fee_minor": 500,
		"transaction_at": "2026-06-13T11:00:00Z"
	}`, http.StatusBadRequest)
	assertAPIStatus(t, router, token, http.MethodPost, "/api/v1/transactions", `{
		"type": "transfer",
		"wallet_id": "`+walletA+`",
		"to_wallet_id": "`+walletB+`",
		"amount_minor": 10000,
		"fee_minor": -1,
		"transaction_at": "2026-06-13T11:00:00Z"
	}`, http.StatusBadRequest)
	// Neither rejected request touched any balance.
	assertWalletBalance(t, router, token, walletA, 500000)
	assertWalletBalance(t, router, token, walletB, 0)

	// Scenario 5 (regression): a transfer with fee omitted behaves exactly as
	// before and reports fee_minor=0.
	plainTransferID := createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "transfer",
		"wallet_id": "`+walletA+`",
		"to_wallet_id": "`+walletB+`",
		"amount_minor": 40000,
		"transaction_at": "2026-06-13T12:00:00Z"
	}`)
	assertWalletBalance(t, router, token, walletA, 460000)
	assertWalletBalance(t, router, token, walletB, 40000)
	assertSumOfBalances(t, router, token, []string{walletA, walletB, walletC}, 800000) // no fee lost
	assertTransactionFee(t, router, token, plainTransferID, 0)

	// A non-transfer with fee omitted (the default 0) is accepted.
	assertAPIStatus(t, router, token, http.MethodPost, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletA+`",
		"category_id": "`+expenseCategory+`",
		"amount_minor": 10000,
		"transaction_at": "2026-06-13T13:00:00Z"
	}`, http.StatusCreated)
}

func assertTransactionFee(t *testing.T, router http.Handler, token string, transactionID string, wantFee int64) {
	t.Helper()

	response := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/transactions/"+transactionID, "", http.StatusOK)
	var parsed struct {
		ID       string `json:"id"`
		FeeMinor int64  `json:"fee_minor"`
	}
	if err := json.Unmarshal(response, &parsed); err != nil {
		t.Fatalf("parse transaction response: %v", err)
	}
	if parsed.FeeMinor != wantFee {
		t.Fatalf("expected transaction %s fee_minor %d, got %d", transactionID, wantFee, parsed.FeeMinor)
	}
}

func assertSumOfBalances(t *testing.T, router http.Handler, token string, walletIDs []string, wantSum int64) {
	t.Helper()

	var sum int64
	for _, walletID := range walletIDs {
		response := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/wallets/"+walletID, "", http.StatusOK)
		var wallet struct {
			BalanceMinor int64 `json:"balance_minor"`
		}
		if err := json.Unmarshal(response, &wallet); err != nil {
			t.Fatalf("parse wallet response: %v", err)
		}
		sum += wallet.BalanceMinor
	}
	if sum != wantSum {
		t.Fatalf("expected sum-of-balances %d, got %d", wantSum, sum)
	}
}
