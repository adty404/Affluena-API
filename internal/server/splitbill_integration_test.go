package server

import (
	"encoding/json"
	"net/http"
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
