package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
	"affluena-api/internal/transaction"
)

func TestSharedWalletMemberCanCreateAndPayDebt(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "shared-debt-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	ownerID, ownerToken := registerIntegrationAPIUser(t, router, "shared-debt-owner")
	memberID, memberToken := registerIntegrationAPIUser(t, router, "shared-debt-member")
	defer cleanupServerIntegrationUsers(t, pool, ownerID)
	defer cleanupServerIntegrationUsers(t, pool, memberID)

	sharedWalletID := createSharedWalletForMember(t, router, ownerToken, memberToken, memberID, "Shared Debt Wallet", 300000)
	disbursementCategoryID := createAPIResource(t, router, memberToken, "/api/v1/categories", `{
		"name": "Shared Debt Disbursement",
		"type": "expense"
	}`)
	paymentCategoryID := createAPIResource(t, router, memberToken, "/api/v1/categories", `{
		"name": "Shared Debt Payment",
		"type": "income"
	}`)
	debtID := createAPIResource(t, router, memberToken, "/api/v1/debts", `{
		"type": "receivable",
		"counterparty_name": "Shared friend",
		"wallet_id": "`+sharedWalletID+`",
		"disbursement_category_id": "`+disbursementCategoryID+`",
		"payment_category_id": "`+paymentCategoryID+`",
		"principal_amount_minor": 50000,
		"opened_at": "2026-06-01T08:00:00Z",
		"due_date": "2026-07-01"
	}`)
	assertWalletBalance(t, router, ownerToken, sharedWalletID, 250000)

	body := performAPIRequest(t, router, memberToken, http.MethodPost, "/api/v1/debts/"+debtID+"/pay", `{
		"amount_minor": 20000,
		"paid_at": "2026-06-20T08:00:00Z"
	}`, http.StatusCreated)
	var payment struct {
		Transaction transaction.Transaction `json:"transaction"`
	}
	if err := json.Unmarshal(body, &payment); err != nil {
		t.Fatalf("parse shared debt payment response: %v", err)
	}
	if payment.Transaction.WalletID != sharedWalletID || payment.Transaction.AmountMinor != 20000 {
		t.Fatalf("expected shared-wallet debt payment transaction, got %+v", payment.Transaction)
	}
	assertWalletBalance(t, router, ownerToken, sharedWalletID, 270000)
}
