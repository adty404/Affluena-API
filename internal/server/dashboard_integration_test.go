package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
	"affluena-api/internal/dashboard"
)

func TestDashboardAnalyticsIntegration(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "dash-integration-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	user, token := registerIntegrationAPIUser(t, router, "dashuser")
	defer cleanupServerIntegrationUsers(t, pool, user)

	// Create a wallet
	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Cash Wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 10000000
	}`)

	// Create categories
	catFood := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Food",
		"type": "expense"
	}`)
	catTransport := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Transport",
		"type": "expense"
	}`)

	// Create category budget for Food for current month
	now := time.Now().UTC()
	currentMonthStr := now.Format("2006-01")

	createAPIResource(t, router, token, "/api/v1/category-budgets", `{
		"category_id": "`+catFood+`",
		"month": "`+currentMonthStr+`",
		"limit_minor": 5000000
	}`)

	// Add transactions for current month
	// Food: 200,000
	createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+catFood+`",
		"amount_minor": 20000000,
		"transaction_at": "`+now.Format(time.RFC3339)+`"
	}`)

	// Transport: 100,000
	createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+catTransport+`",
		"amount_minor": 10000000,
		"transaction_at": "`+now.Format(time.RFC3339)+`"
	}`)

	// Add income transaction 3 months ago to test trend
	threeMonthsAgo := now.AddDate(0, -3, 0)
	createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+walletID+`",
		"amount_minor": 50000000,
		"transaction_at": "`+threeMonthsAgo.Format(time.RFC3339)+`"
	}`)

	// 1. Test Cashflow Trend
	resTrend := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/dashboard/cashflow-trend?months=4", "", http.StatusOK)
	var trendResp struct {
		Trend []dashboard.CashflowTrend `json:"trend"`
	}
	json.Unmarshal(resTrend, &trendResp)
	if len(trendResp.Trend) != 4 {
		t.Fatalf("Expected 4 months of trend, got %d", len(trendResp.Trend))
	}

	// 2. Test Expense Distribution
	resDist := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/dashboard/expense-distribution?month="+currentMonthStr, "", http.StatusOK)
	var distResp struct {
		Distribution []dashboard.ExpenseDistribution `json:"distribution"`
	}
	json.Unmarshal(resDist, &distResp)
	if len(distResp.Distribution) != 2 {
		t.Fatalf("Expected 2 categories in distribution, got %d", len(distResp.Distribution))
	}
	// Verify percentages (Total is 30M, Food is 20M = 66.6%, Transport is 10M = 33.3%)
	if distResp.Distribution[0].CategoryID == catFood {
		if distResp.Distribution[0].AmountMinor != 20000000 {
			t.Fatalf("Expected 20M for food")
		}
	}

	// 3. Test Forecast
	resForecast := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/dashboard/forecast?month="+currentMonthStr, "", http.StatusOK)
	var forecast dashboard.Forecast
	json.Unmarshal(resForecast, &forecast)

	if forecast.CurrentExpenseMinor != 30000000 {
		t.Fatalf("Expected current expense 30M, got %d", forecast.CurrentExpenseMinor)
	}
	if forecast.BudgetLimitMinor != 5000000 {
		t.Fatalf("Expected budget limit 5M, got %d", forecast.BudgetLimitMinor)
	}
	if forecast.Status != "overbudget" {
		t.Fatalf("Expected status overbudget, got %s", forecast.Status)
	}
}
