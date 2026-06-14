package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"affluena-api/internal/config"
	"affluena-api/internal/transaction"
)

func TestSharedWalletMemberCanPaySubscription(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "shared-subscription-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	ownerID, ownerToken := registerIntegrationAPIUser(t, router, "shared-subscription-owner")
	memberID, memberToken := registerIntegrationAPIUser(t, router, "shared-subscription-member")
	defer cleanupServerIntegrationUsers(t, pool, ownerID)
	defer cleanupServerIntegrationUsers(t, pool, memberID)

	sharedWalletID := createSharedWalletForMember(t, router, ownerToken, memberToken, memberID, "Shared Subscription Wallet", 300000)
	categoryID := createAPIResource(t, router, memberToken, "/api/v1/categories", `{
		"name": "Shared Subscription",
		"type": "expense"
	}`)
	subscriptionID := createAPIResource(t, router, memberToken, "/api/v1/subscriptions", `{
		"name": "Shared Google One",
		"wallet_id": "`+sharedWalletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 50000,
		"billing_cycle": "monthly",
		"next_due_date": "2026-06-20"
	}`)

	body := performAPIRequest(t, router, memberToken, http.MethodPost, "/api/v1/subscriptions/"+subscriptionID+"/pay", `{
		"paid_at": "2026-06-20T08:00:00Z"
	}`, http.StatusCreated)
	var payment struct {
		Transaction transaction.Transaction `json:"transaction"`
	}
	if err := json.Unmarshal(body, &payment); err != nil {
		t.Fatalf("parse shared subscription payment response: %v", err)
	}
	if payment.Transaction.WalletID != sharedWalletID || payment.Transaction.AmountMinor != 50000 {
		t.Fatalf("expected shared-wallet subscription expense, got %+v", payment.Transaction)
	}
	assertWalletBalance(t, router, ownerToken, sharedWalletID, 250000)
}

func TestSharedWalletMemberCanPayInstallment(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "shared-installment-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	ownerID, ownerToken := registerIntegrationAPIUser(t, router, "shared-installment-owner")
	memberID, memberToken := registerIntegrationAPIUser(t, router, "shared-installment-member")
	defer cleanupServerIntegrationUsers(t, pool, ownerID)
	defer cleanupServerIntegrationUsers(t, pool, memberID)

	sharedWalletID := createSharedWalletForMember(t, router, ownerToken, memberToken, memberID, "Shared Installment Wallet", 300000)
	categoryID := createAPIResource(t, router, memberToken, "/api/v1/categories", `{
		"name": "Shared Installment",
		"type": "expense"
	}`)
	installmentID := createAPIResource(t, router, memberToken, "/api/v1/installments", `{
		"name": "Shared Laptop",
		"wallet_id": "`+sharedWalletID+`",
		"category_id": "`+categoryID+`",
		"total_amount_minor": 100000,
		"monthly_amount_minor": 50000,
		"tenor_months": 2,
		"start_date": "2026-06-01",
		"due_day": 5
	}`)

	body := performAPIRequest(t, router, memberToken, http.MethodPost, "/api/v1/installments/"+installmentID+"/pay", `{
		"paid_at": "2026-06-05T08:00:00Z"
	}`, http.StatusCreated)
	var payment struct {
		Transaction transaction.Transaction `json:"transaction"`
	}
	if err := json.Unmarshal(body, &payment); err != nil {
		t.Fatalf("parse shared installment payment response: %v", err)
	}
	if payment.Transaction.WalletID != sharedWalletID || payment.Transaction.AmountMinor != 50000 {
		t.Fatalf("expected shared-wallet installment expense, got %+v", payment.Transaction)
	}
	assertWalletBalance(t, router, ownerToken, sharedWalletID, 250000)
}

func createSharedWalletForMember(t *testing.T, router http.Handler, ownerToken string, memberToken string, memberID string, name string, balance int64) string {
	t.Helper()

	sharedWalletID := createAPIResource(t, router, ownerToken, "/api/v1/wallets", `{
		"name": "`+name+`",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": `+strconv.FormatInt(balance, 10)+`
	}`)
	memberEmail := getIntegrationUserEmail(t, router, memberToken)
	assertAPIStatus(t, router, ownerToken, http.MethodPost, "/api/v1/wallets/"+sharedWalletID+"/invites", `{
		"email": "`+memberEmail+`"
	}`, http.StatusCreated)
	assertAPIStatus(t, router, memberToken, http.MethodPatch, "/api/v1/wallets/"+sharedWalletID+"/members/"+memberID, `{
		"status": "joined"
	}`, http.StatusOK)
	return sharedWalletID
}
