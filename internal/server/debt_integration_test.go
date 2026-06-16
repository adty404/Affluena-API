package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"affluena-api/internal/config"
	"affluena-api/internal/debt"
)

func TestDebtCancellation(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "debt-cancel-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "debt-cancel-user")
	otherUserID, otherToken := registerIntegrationAPIUser(t, router, "debt-other-user")
	defer cleanupServerIntegrationUsers(t, pool, userID)
	defer cleanupServerIntegrationUsers(t, pool, otherUserID)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Debt Wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 1000000
	}`)
	disburseCat := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Lent Money",
		"type": "expense"
	}`)
	paymentCat := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Returned Money",
		"type": "income"
	}`)

	// 1. Create a debt
	debtID := createAPIResource(t, router, token, "/api/v1/debts", `{
		"type": "receivable",
		"counterparty_name": "John Doe",
		"wallet_id": "`+walletID+`",
		"disbursement_category_id": "`+disburseCat+`",
		"payment_category_id": "`+paymentCat+`",
		"principal_amount_minor": 500000
	}`)

	// 2. Unrelated user tries to cancel it (should fail)
	assertAPIStatus(t, router, otherToken, http.MethodDelete, "/api/v1/debts/"+debtID, "", http.StatusNotFound)

	// 3. User cancels the debt
	assertAPIStatus(t, router, token, http.MethodDelete, "/api/v1/debts/"+debtID, "", http.StatusNoContent)

	// 4. Verify it's cancelled
	resBody := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/debts/"+debtID, "", http.StatusOK)
	var getDebt debt.Debt
	if err := json.Unmarshal(resBody, &getDebt); err != nil {
		t.Fatalf("parse debt: %v", err)
	}
	if getDebt.Status != debt.DebtStatusCancelled {
		t.Fatalf("expected status cancelled, got %s", getDebt.Status)
	}

	// 5. Try to pay the cancelled debt (should fail)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/debts/"+debtID+"/pay", strings.NewReader(`{"amount_minor": 100000}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code == http.StatusCreated || recorder.Code == http.StatusOK {
		t.Fatalf("expected failure when paying cancelled debt, got success %d", recorder.Code)
	}

	// 6. Verify it's not active in Upcoming Debts dashboard
	now := time.Now().UTC()
	currentMonthStr := now.Format("2006-01")
	dashRes := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/dashboard/summary?month="+currentMonthStr, "", http.StatusOK)
	var dashResp struct {
		UpcomingDebts []interface{} `json:"upcoming_debts"`
	}
	if err := json.Unmarshal(dashRes, &dashResp); err != nil {
		t.Fatalf("parse dash: %v", err)
	}
	if len(dashResp.UpcomingDebts) != 0 {
		t.Fatalf("expected 0 upcoming debts, got %d", len(dashResp.UpcomingDebts))
	}
}
