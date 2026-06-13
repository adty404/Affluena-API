package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
)

func TestSubscriptionAccountDetailLifecycle(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "subscription-account-detail-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "subscription-account-detail")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Subscription wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 1000000
	}`)
	categoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Subscriptions",
		"type": "expense"
	}`)

	subscriptionID := createAPIResource(t, router, token, "/api/v1/subscriptions", `{
		"name": "Google One",
		"account_detail": "personal@example.com",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 26900,
		"billing_cycle": "monthly",
		"next_due_date": "2026-06-20",
		"note": "Personal Google One"
	}`)

	created := getSubscriptionResponse(t, router, token, subscriptionID)
	if created.AccountDetail != "personal@example.com" {
		t.Fatalf("expected created subscription account detail, got %+v", created)
	}

	listed := listSubscriptionAccountDetailsByID(t, performAPIRequest(t, router, token, http.MethodGet, "/api/v1/subscriptions", "", http.StatusOK))
	if listed[subscriptionID] != "personal@example.com" {
		t.Fatalf("expected listed subscription account detail, got %+v", listed)
	}

	performAPIRequest(t, router, token, http.MethodPut, "/api/v1/subscriptions/"+subscriptionID, `{
		"name": "Google One",
		"account_detail": "work@example.com",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 26900,
		"billing_cycle": "monthly",
		"next_due_date": "2026-06-20",
		"status": "active",
		"note": "Work Google One"
	}`, http.StatusOK)

	updated := getSubscriptionResponse(t, router, token, subscriptionID)
	if updated.AccountDetail != "work@example.com" {
		t.Fatalf("expected updated subscription account detail, got %+v", updated)
	}

	paymentResponse := performAPIRequest(t, router, token, http.MethodPost, "/api/v1/subscriptions/"+subscriptionID+"/pay", `{
		"paid_at": "2026-06-20T08:00:00Z",
		"note": "Paid Google One"
	}`, http.StatusCreated)
	var payment struct {
		Subscription subscriptionAccountDetailResponse `json:"subscription"`
	}
	if err := json.Unmarshal(paymentResponse, &payment); err != nil {
		t.Fatalf("parse subscription payment response: %v", err)
	}
	if payment.Subscription.AccountDetail != "work@example.com" {
		t.Fatalf("expected paid subscription account detail, got %+v", payment.Subscription)
	}
}

type subscriptionAccountDetailResponse struct {
	ID            string `json:"id"`
	AccountDetail string `json:"account_detail"`
}

func getSubscriptionResponse(t *testing.T, router http.Handler, token string, subscriptionID string) subscriptionAccountDetailResponse {
	t.Helper()

	response := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/subscriptions/"+subscriptionID, "", http.StatusOK)
	var subscription subscriptionAccountDetailResponse
	if err := json.Unmarshal(response, &subscription); err != nil {
		t.Fatalf("parse subscription response: %v", err)
	}
	if subscription.ID != subscriptionID {
		t.Fatalf("expected subscription %s, got %+v", subscriptionID, subscription)
	}
	return subscription
}

func listSubscriptionAccountDetailsByID(t *testing.T, response []byte) map[string]string {
	t.Helper()

	var parsed struct {
		Subscriptions []subscriptionAccountDetailResponse `json:"subscriptions"`
	}
	if err := json.Unmarshal(response, &parsed); err != nil {
		t.Fatalf("parse subscription list response: %v", err)
	}

	accountDetailsByID := make(map[string]string, len(parsed.Subscriptions))
	for _, subscription := range parsed.Subscriptions {
		accountDetailsByID[subscription.ID] = subscription.AccountDetail
	}
	return accountDetailsByID
}
