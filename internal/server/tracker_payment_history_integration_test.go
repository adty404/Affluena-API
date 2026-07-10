package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"affluena-api/internal/config"
	"affluena-api/internal/transaction"
)

type trackerPaymentHistoryEntry struct {
	ID             string    `json:"id"`
	InstallmentID  string    `json:"installment_id"`
	SubscriptionID string    `json:"subscription_id"`
	AmountMinor    int64     `json:"amount_minor"`
	PaidAt         time.Time `json:"paid_at"`
	TransactionID  string    `json:"transaction_id"`
	Note           string    `json:"note"`
}

func TestInstallmentPaymentHistory(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "installment-payment-history-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "installment-payments")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Installment Payment Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 300000
	}`)
	categoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Installment History",
		"type": "expense"
	}`)
	installmentID := createAPIResource(t, router, token, "/api/v1/installments", `{
		"name": "History Laptop",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"total_amount_minor": 150000,
		"monthly_amount_minor": 50000,
		"tenor_months": 3,
		"start_date": "2026-07-01",
		"due_day": 5
	}`)

	emptyBody := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/installments/"+installmentID+"/payments", "", http.StatusOK)
	if !strings.Contains(string(emptyBody), `"payments":[]`) {
		t.Fatalf("expected empty payments array before any payment, got %s", string(emptyBody))
	}

	firstTxID := payTrackerForHistory(t, router, token, "/api/v1/installments/"+installmentID+"/pay", `{
		"paid_at": "2026-07-05T08:00:00Z"
	}`)
	secondTxID := payTrackerForHistory(t, router, token, "/api/v1/installments/"+installmentID+"/pay", `{
		"paid_at": "2026-08-05T08:00:00Z",
		"note": "bulan kedua"
	}`)

	body := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/installments/"+installmentID+"/payments", "", http.StatusOK)
	var parsed struct {
		Payments []trackerPaymentHistoryEntry `json:"payments"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse installment payments response: %v", err)
	}
	if len(parsed.Payments) != 2 {
		t.Fatalf("expected 2 installment payments, got %d: %s", len(parsed.Payments), string(body))
	}
	newest, oldest := parsed.Payments[0], parsed.Payments[1]
	if !newest.PaidAt.After(oldest.PaidAt) {
		t.Fatalf("expected payments ordered paid_at DESC, got %s then %s", newest.PaidAt, oldest.PaidAt)
	}
	if newest.TransactionID != secondTxID || oldest.TransactionID != firstTxID {
		t.Fatalf("expected transaction linkage newest=%s oldest=%s, got newest=%s oldest=%s",
			secondTxID, firstTxID, newest.TransactionID, oldest.TransactionID)
	}
	for _, payment := range parsed.Payments {
		if payment.AmountMinor != 50000 {
			t.Fatalf("expected amount_minor 50000, got %+v", payment)
		}
		if payment.InstallmentID != installmentID {
			t.Fatalf("expected installment_id %s, got %+v", installmentID, payment)
		}
		if payment.ID == "" {
			t.Fatalf("expected payment id, got %+v", payment)
		}
	}
	if newest.Note != "bulan kedua" || oldest.Note != "" {
		t.Fatalf("expected notes to round-trip, got newest=%q oldest=%q", newest.Note, oldest.Note)
	}
}

func TestSubscriptionPaymentHistory(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "subscription-payment-history-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "subscription-payments")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Subscription Payment Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 300000
	}`)
	categoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Subscription History",
		"type": "expense"
	}`)
	subscriptionID := createAPIResource(t, router, token, "/api/v1/subscriptions", `{
		"name": "History Streaming",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 30000,
		"billing_cycle": "monthly",
		"next_due_date": "2026-07-20"
	}`)

	firstTxID := payTrackerForHistory(t, router, token, "/api/v1/subscriptions/"+subscriptionID+"/pay", `{
		"paid_at": "2026-07-20T08:00:00Z"
	}`)
	secondTxID := payTrackerForHistory(t, router, token, "/api/v1/subscriptions/"+subscriptionID+"/pay", `{
		"paid_at": "2026-08-20T08:00:00Z"
	}`)

	body := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/subscriptions/"+subscriptionID+"/payments", "", http.StatusOK)
	var parsed struct {
		Payments []trackerPaymentHistoryEntry `json:"payments"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse subscription payments response: %v", err)
	}
	if len(parsed.Payments) != 2 {
		t.Fatalf("expected 2 subscription payments, got %d: %s", len(parsed.Payments), string(body))
	}
	newest, oldest := parsed.Payments[0], parsed.Payments[1]
	if !newest.PaidAt.After(oldest.PaidAt) {
		t.Fatalf("expected payments ordered paid_at DESC, got %s then %s", newest.PaidAt, oldest.PaidAt)
	}
	if newest.TransactionID != secondTxID || oldest.TransactionID != firstTxID {
		t.Fatalf("expected transaction linkage newest=%s oldest=%s, got newest=%s oldest=%s",
			secondTxID, firstTxID, newest.TransactionID, oldest.TransactionID)
	}
	for _, payment := range parsed.Payments {
		if payment.AmountMinor != 30000 {
			t.Fatalf("expected amount_minor 30000, got %+v", payment)
		}
		if payment.SubscriptionID != subscriptionID {
			t.Fatalf("expected subscription_id %s, got %+v", subscriptionID, payment)
		}
	}
}

func TestTrackerPaymentHistoryIsolation(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "tracker-payment-isolation-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	ownerID, ownerToken := registerIntegrationAPIUser(t, router, "tracker-payments-owner")
	otherID, otherToken := registerIntegrationAPIUser(t, router, "tracker-payments-other")
	defer cleanupServerIntegrationUsers(t, pool, ownerID)
	defer cleanupServerIntegrationUsers(t, pool, otherID)

	walletID := createAPIResource(t, router, ownerToken, "/api/v1/wallets", `{
		"name": "Isolated Tracker Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 300000
	}`)
	categoryID := createAPIResource(t, router, ownerToken, "/api/v1/categories", `{
		"name": "Isolated Tracker",
		"type": "expense"
	}`)
	installmentID := createAPIResource(t, router, ownerToken, "/api/v1/installments", `{
		"name": "Isolated Laptop",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"total_amount_minor": 100000,
		"monthly_amount_minor": 50000,
		"tenor_months": 2,
		"start_date": "2026-07-01",
		"due_day": 5
	}`)
	subscriptionID := createAPIResource(t, router, ownerToken, "/api/v1/subscriptions", `{
		"name": "Isolated Streaming",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 30000,
		"billing_cycle": "monthly",
		"next_due_date": "2026-07-20"
	}`)

	installmentTxID := payTrackerForHistory(t, router, ownerToken, "/api/v1/installments/"+installmentID+"/pay", `{
		"paid_at": "2026-07-05T08:00:00Z"
	}`)
	subscriptionTxID := payTrackerForHistory(t, router, ownerToken, "/api/v1/subscriptions/"+subscriptionID+"/pay", `{
		"paid_at": "2026-07-20T08:00:00Z"
	}`)

	installmentBody := performAPIRequest(t, router, otherToken, http.MethodGet, "/api/v1/installments/"+installmentID+"/payments", "", http.StatusNotFound)
	if strings.Contains(string(installmentBody), installmentTxID) {
		t.Fatalf("cross-user installment payments response leaked transaction id: %s", string(installmentBody))
	}
	subscriptionBody := performAPIRequest(t, router, otherToken, http.MethodGet, "/api/v1/subscriptions/"+subscriptionID+"/payments", "", http.StatusNotFound)
	if strings.Contains(string(subscriptionBody), subscriptionTxID) {
		t.Fatalf("cross-user subscription payments response leaked transaction id: %s", string(subscriptionBody))
	}

	assertAPIStatus(t, router, ownerToken, http.MethodGet, "/api/v1/installments/"+installmentID+"/payments", "", http.StatusOK)
	assertAPIStatus(t, router, ownerToken, http.MethodGet, "/api/v1/subscriptions/"+subscriptionID+"/payments", "", http.StatusOK)
}

func payTrackerForHistory(t *testing.T, router http.Handler, token string, path string, body string) string {
	t.Helper()

	response := performAPIRequest(t, router, token, http.MethodPost, path, body, http.StatusCreated)
	var payment struct {
		Transaction transaction.Transaction `json:"transaction"`
	}
	if err := json.Unmarshal(response, &payment); err != nil {
		t.Fatalf("parse pay response for %s: %v", path, err)
	}
	if payment.Transaction.ID == "" {
		t.Fatalf("pay response for %s missing transaction id: %s", path, string(response))
	}
	return payment.Transaction.ID
}
