package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"affluena-api/internal/config"
	"affluena-api/internal/splitbill"
)

func TestSplitBillIntegration(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "split-integration-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	user, token := registerIntegrationAPIUser(t, router, "splituser")
	defer cleanupServerIntegrationUsers(t, pool, user)

	// Create wallet
	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "BCA",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 10000000
	}`)

	// Create categories
	foodCat := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Food",
		"type": "expense"
	}`)
	piutangDisbCat := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Memberi Pinjaman",
		"type": "expense"
	}`)
	piutangPayCat := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Pinjaman Dibayar",
		"type": "income"
	}`)

	// Payload for split bill
	splitPayload := `{
		"wallet_id": "` + walletID + `",
		"category_id": "` + foodCat + `",
		"total_amount_minor": 300000,
		"note": "Dinner with friends",
		"splits": [
			{
				"counterparty_name": "Budi",
				"amount_minor": 100000,
				"disbursement_category_id": "` + piutangDisbCat + `",
				"payment_category_id": "` + piutangPayCat + `"
			},
			{
				"counterparty_name": "Citra",
				"amount_minor": 100000,
				"disbursement_category_id": "` + piutangDisbCat + `",
				"payment_category_id": "` + piutangPayCat + `"
			}
		]
	}`

	respBody := performAPIRequest(t, router, token, "POST", "/api/v1/transactions/split", splitPayload, http.StatusCreated)

	var splitResp splitbill.SplitTransactionResponse
	json.Unmarshal([]byte(respBody), &splitResp)

	if splitResp.TransactionID == "" {
		t.Errorf("expected transaction_id to be populated")
	}
	if len(splitResp.DebtIDs) != 2 {
		t.Errorf("expected 2 debt IDs, got %d", len(splitResp.DebtIDs))
	}

	// Verify wallet balance
	wBody := performAPIRequest(t, router, token, "GET", "/api/v1/wallets/"+walletID, "", http.StatusOK)
	// Expected balance: 10M - 300k = 9,700,000
	var walletResp struct {
		BalanceMinor int64 `json:"balance_minor"`
	}
	json.Unmarshal([]byte(wBody), &walletResp)
	if walletResp.BalanceMinor != 9700000 {
		t.Errorf("expected wallet balance 9,700,000, got %d", walletResp.BalanceMinor)
	}
}

func TestSplitBillRollsBackAllWritesWhenDebtCreationFails(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "split-rollback-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	user, token := registerIntegrationAPIUser(t, router, "split-rollback")
	defer cleanupServerIntegrationUsers(t, pool, user)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Split rollback wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 1000000
	}`)
	foodCat := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Split rollback food",
		"type": "expense"
	}`)
	piutangDisbCat := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Split rollback disbursement",
		"type": "expense"
	}`)
	piutangPayCat := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Split rollback payment",
		"type": "income"
	}`)

	payload := `{
		"wallet_id": "` + walletID + `",
		"category_id": "` + foodCat + `",
		"total_amount_minor": 300000,
		"transaction_at": "2026-06-13T12:00:00Z",
		"note": "Rollback dinner",
		"splits": [
			{
				"counterparty_name": "Budi",
				"amount_minor": 100000,
				"disbursement_category_id": "` + piutangDisbCat + `",
				"payment_category_id": "` + piutangPayCat + `"
			},
			{
				"counterparty_name": "Citra",
				"amount_minor": 100000,
				"disbursement_category_id": "` + piutangDisbCat + `",
				"payment_category_id": "00000000-0000-0000-0000-000000000000"
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/split", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNotFound {
		t.Errorf("expected missing split debt reference to return 404, got %d: %s", recorder.Code, recorder.Body.String())
	}

	assertWalletBalance(t, router, token, walletID, 1000000)
	assertListCount(t, router, token, "/api/v1/transactions", "transactions", 0)
	assertListCount(t, router, token, "/api/v1/debts", "debts", 0)
}

func TestSplitBillFullSplit(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "split-full-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	user, token := registerIntegrationAPIUser(t, router, "splitfulluser")
	defer cleanupServerIntegrationUsers(t, pool, user)

	// Create wallet
	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "BCA Full Split",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 10000000
	}`)

	// Create categories
	foodCat := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Food",
		"type": "expense"
	}`)
	piutangDisbCat := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Memberi Pinjaman",
		"type": "expense"
	}`)
	piutangPayCat := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Pinjaman Dibayar",
		"type": "income"
	}`)

	// Payload for full split
	splitPayload := `{
		"wallet_id": "` + walletID + `",
		"category_id": "` + foodCat + `",
		"total_amount_minor": 200000,
		"note": "Full dinner split",
		"splits": [
			{
				"counterparty_name": "Budi",
				"amount_minor": 100000,
				"disbursement_category_id": "` + piutangDisbCat + `",
				"payment_category_id": "` + piutangPayCat + `"
			},
			{
				"counterparty_name": "Citra",
				"amount_minor": 100000,
				"disbursement_category_id": "` + piutangDisbCat + `",
				"payment_category_id": "` + piutangPayCat + `"
			}
		]
	}`

	respBody := performAPIRequest(t, router, token, "POST", "/api/v1/transactions/split", splitPayload, http.StatusCreated)

	var splitResp splitbill.SplitTransactionResponse
	json.Unmarshal([]byte(respBody), &splitResp)

	if splitResp.TransactionID == "" {
		t.Errorf("expected transaction_id to be populated")
	}

	// Verify wallet balance
	wBody := performAPIRequest(t, router, token, "GET", "/api/v1/wallets/"+walletID, "", http.StatusOK)
	// Expected balance: 10M - 200k = 9,800,000
	var walletResp struct {
		BalanceMinor int64 `json:"balance_minor"`
	}
	json.Unmarshal([]byte(wBody), &walletResp)
	if walletResp.BalanceMinor != 9800000 {
		t.Errorf("expected wallet balance 9,800,000, got %d", walletResp.BalanceMinor)
	}

	// Verify the original transaction amount is not 0
	txBody := performAPIRequest(t, router, token, "GET", "/api/v1/transactions/"+splitResp.TransactionID, "", http.StatusOK)
	var txResp struct {
		AmountMinor int64 `json:"amount_minor"`
	}
	json.Unmarshal([]byte(txBody), &txResp)
	if txResp.AmountMinor != 200000 {
		t.Errorf("expected transaction amount to remain 200000 (total bill), got %d", txResp.AmountMinor)
	}
}

func assertListCount(t *testing.T, router http.Handler, token string, path string, key string, wantCount int) {
	t.Helper()

	body := performAPIRequest(t, router, token, http.MethodGet, path, "", http.StatusOK)
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse list response for %s: %v", path, err)
	}
	items, ok := parsed[key]
	if !ok {
		t.Fatalf("list response for %s missing %q: %s", path, key, string(body))
	}
	var collection []json.RawMessage
	if err := json.Unmarshal(items, &collection); err != nil {
		t.Fatalf("parse %s collection for %s: %v", key, path, err)
	}
	if len(collection) != wantCount {
		t.Fatalf("expected %s count %d, got %d: %s", key, wantCount, len(collection), string(items))
	}
}
