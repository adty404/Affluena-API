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
	catIncome := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Salary",
		"type": "income"
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
		"category_id": "`+catIncome+`",
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

func TestDashboardAnalyticsEdgeCases(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "dash-edge-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userA, tokenA := registerIntegrationAPIUser(t, router, "dash-edge-a")
	userB, tokenB := registerIntegrationAPIUser(t, router, "dash-edge-b")
	defer cleanupServerIntegrationUsers(t, pool, userA, userB)

	for _, query := range []string{"months=0", "months=13", "months=abc"} {
		performAPIRequest(t, router, tokenA, http.MethodGet, "/api/v1/dashboard/cashflow-trend?"+query, "", http.StatusBadRequest)
	}

	resTrend := performAPIRequest(t, router, tokenA, http.MethodGet, "/api/v1/dashboard/cashflow-trend?months=3", "", http.StatusOK)
	var trendResp struct {
		Trend []dashboard.CashflowTrend `json:"trend"`
	}
	if err := json.Unmarshal(resTrend, &trendResp); err != nil {
		t.Fatalf("parse trend response: %v", err)
	}
	if len(trendResp.Trend) != 3 {
		t.Fatalf("expected 3 empty trend months, got %d", len(trendResp.Trend))
	}
	for _, month := range trendResp.Trend {
		if month.IncomeMinor != 0 || month.ExpenseMinor != 0 || month.CashflowMinor != 0 {
			t.Fatalf("expected empty trend month, got %+v", month)
		}
	}

	currentMonth := time.Now().UTC().Format("2006-01")
	resDist := performAPIRequest(t, router, tokenA, http.MethodGet, "/api/v1/dashboard/expense-distribution?month="+currentMonth, "", http.StatusOK)
	var distResp struct {
		Distribution []dashboard.ExpenseDistribution `json:"distribution"`
	}
	if err := json.Unmarshal(resDist, &distResp); err != nil {
		t.Fatalf("parse distribution response: %v", err)
	}
	if len(distResp.Distribution) != 0 {
		t.Fatalf("expected empty distribution, got %+v", distResp.Distribution)
	}

	resForecast := performAPIRequest(t, router, tokenA, http.MethodGet, "/api/v1/dashboard/forecast?month="+currentMonth, "", http.StatusOK)
	var forecast dashboard.Forecast
	if err := json.Unmarshal(resForecast, &forecast); err != nil {
		t.Fatalf("parse forecast response: %v", err)
	}
	if forecast.CurrentExpenseMinor != 0 || forecast.DailyAverageMinor != 0 || forecast.ForecastedExpenseMinor != 0 || forecast.BudgetLimitMinor != 0 || forecast.Status != "safe" {
		t.Fatalf("expected empty safe forecast, got %+v", forecast)
	}

	walletA := createAPIResource(t, router, tokenA, "/api/v1/wallets", `{
		"name": "Analytics wallet A",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 1000000
	}`)
	parentCategory := createAPIResource(t, router, tokenA, "/api/v1/categories", `{
		"name": "Living",
		"type": "expense"
	}`)
	childCategory := createAPIResource(t, router, tokenA, "/api/v1/categories", `{
		"name": "Groceries",
		"type": "expense",
		"parent_id": "`+parentCategory+`"
	}`)
	createAPIResource(t, router, tokenA, "/api/v1/category-budgets", `{
		"category_id": "`+parentCategory+`",
		"month": "`+currentMonth+`",
		"limit_minor": 100000
	}`)
	createAPIResource(t, router, tokenA, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletA+`",
		"category_id": "`+childCategory+`",
		"amount_minor": 20000,
		"transaction_at": "`+time.Now().UTC().Format(time.RFC3339)+`"
	}`)

	walletB := createAPIResource(t, router, tokenB, "/api/v1/wallets", `{
		"name": "Analytics wallet B",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 1000000
	}`)
	categoryB := createAPIResource(t, router, tokenB, "/api/v1/categories", `{
		"name": "Other Living",
		"type": "expense"
	}`)
	createAPIResource(t, router, tokenB, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletB+`",
		"category_id": "`+categoryB+`",
		"amount_minor": 999999,
		"transaction_at": "`+time.Now().UTC().Format(time.RFC3339)+`"
	}`)

	resDist = performAPIRequest(t, router, tokenA, http.MethodGet, "/api/v1/dashboard/expense-distribution?month="+currentMonth, "", http.StatusOK)
	if err := json.Unmarshal(resDist, &distResp); err != nil {
		t.Fatalf("parse distribution response: %v", err)
	}
	if len(distResp.Distribution) != 1 {
		t.Fatalf("expected one rolled-up distribution row, got %+v", distResp.Distribution)
	}
	if distResp.Distribution[0].CategoryID != parentCategory || distResp.Distribution[0].CategoryName != "Living" || distResp.Distribution[0].AmountMinor != 20000 {
		t.Fatalf("expected child expense rolled up to parent only, got %+v", distResp.Distribution[0])
	}

	resForecast = performAPIRequest(t, router, tokenA, http.MethodGet, "/api/v1/dashboard/forecast?month="+currentMonth, "", http.StatusOK)
	if err := json.Unmarshal(resForecast, &forecast); err != nil {
		t.Fatalf("parse forecast response: %v", err)
	}
	if forecast.CurrentExpenseMinor != 20000 || forecast.BudgetLimitMinor != 100000 || forecast.Status != "safe" {
		t.Fatalf("expected isolated parent-budget forecast, got %+v", forecast)
	}
}

func TestDashboardForecastWithoutBudgetDoesNotOverbudget(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "dash-no-budget-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	user, token := registerIntegrationAPIUser(t, router, "dash-no-budget")
	defer cleanupServerIntegrationUsers(t, pool, user)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Cash Wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 1000000
	}`)
	categoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Food",
		"type": "expense"
	}`)
	now := time.Now().UTC()
	createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 50000,
		"transaction_at": "`+now.Format(time.RFC3339)+`"
	}`)

	resForecast := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/dashboard/forecast?month="+now.Format("2006-01"), "", http.StatusOK)
	var forecast dashboard.Forecast
	if err := json.Unmarshal(resForecast, &forecast); err != nil {
		t.Fatalf("parse forecast response: %v", err)
	}
	if forecast.BudgetLimitMinor != 0 || forecast.CurrentExpenseMinor != 50000 || forecast.Status != "safe" {
		t.Fatalf("expected no-budget spending to remain safe, got %+v", forecast)
	}
}

func TestSharedWalletMemberTransactionsDoNotAffectOwnerPersonalBudgetButAffectDashboard(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "dash-shared-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	owner, tokenOwner := registerIntegrationAPIUser(t, router, "dash-shared-owner")
	member, tokenMember := registerIntegrationAPIUser(t, router, "dash-shared-member")
	defer cleanupServerIntegrationUsers(t, pool, owner, member)

	// Owner creates wallet
	walletID := createAPIResource(t, router, tokenOwner, "/api/v1/wallets", `{
		"name": "Shared Expense Wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 10000000
	}`)

	// Fetch member email
	resMe := performAPIRequest(t, router, tokenMember, http.MethodGet, "/api/v1/auth/me", "", http.StatusOK)
	var me struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	json.Unmarshal(resMe, &me)

	// Owner shares wallet with Member
	performAPIRequest(t, router, tokenOwner, http.MethodPost, "/api/v1/wallets/"+walletID+"/invites", `{
		"email": "`+me.User.Email+`"
	}`, http.StatusCreated)

	// Member accepts
	performAPIRequest(t, router, tokenMember, http.MethodPatch, "/api/v1/wallets/"+walletID+"/members/"+member, `{"status":"joined"}`, http.StatusOK)

	// Owner creates a category and a budget
	ownerCategory := createAPIResource(t, router, tokenOwner, "/api/v1/categories", `{
		"name": "Owner Food",
		"type": "expense"
	}`)
	now := time.Now().UTC()
	currentMonthStr := now.Format("2006-01")
	createAPIResource(t, router, tokenOwner, "/api/v1/category-budgets", `{
		"category_id": "`+ownerCategory+`",
		"month": "`+currentMonthStr+`",
		"limit_minor": 1000000
	}`)

	// Member creates their own category
	memberCategory := createAPIResource(t, router, tokenMember, "/api/v1/categories", `{
		"name": "Member Food",
		"type": "expense"
	}`)

	// Member creates an expense transaction in the shared wallet using their category
	createAPIResource(t, router, tokenMember, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+memberCategory+`",
		"amount_minor": 300000,
		"transaction_at": "`+now.Format(time.RFC3339)+`"
	}`)

	// Owner checks Dashboard Expense Distribution
	resDist := performAPIRequest(t, router, tokenOwner, http.MethodGet, "/api/v1/dashboard/expense-distribution?month="+currentMonthStr, "", http.StatusOK)
	var distResp struct {
		Distribution []dashboard.ExpenseDistribution `json:"distribution"`
	}
	if err := json.Unmarshal(resDist, &distResp); err != nil {
		t.Fatalf("parse distribution response: %v", err)
	}
	if len(distResp.Distribution) != 1 {
		t.Fatalf("expected 1 distribution row, got %d", len(distResp.Distribution))
	}
	// Fallback to Uncategorized since the category belongs to the member
	if distResp.Distribution[0].CategoryName != "Uncategorized" {
		t.Fatalf("expected Uncategorized, got %s", distResp.Distribution[0].CategoryName)
	}
	if distResp.Distribution[0].AmountMinor != 300000 {
		t.Fatalf("expected amount 300000, got %d", distResp.Distribution[0].AmountMinor)
	}

	// Owner checks Dashboard Forecast
	resForecast := performAPIRequest(t, router, tokenOwner, http.MethodGet, "/api/v1/dashboard/forecast?month="+currentMonthStr, "", http.StatusOK)
	var forecast dashboard.Forecast
	if err := json.Unmarshal(resForecast, &forecast); err != nil {
		t.Fatalf("parse forecast response: %v", err)
	}
	// The current expense minor in the forecast should include the member's expense
	if forecast.CurrentExpenseMinor != 300000 {
		t.Fatalf("expected current expense to be 300000, got %d", forecast.CurrentExpenseMinor)
	}
	// The budget limit minor should be 1000000 (Owner's budget)
	if forecast.BudgetLimitMinor != 1000000 {
		t.Fatalf("expected budget limit to be 1000000, got %d", forecast.BudgetLimitMinor)
	}

	// Owner checks personal Budget List (from /api/v1/category-budgets)
	resBudgets := performAPIRequest(t, router, tokenOwner, http.MethodGet, "/api/v1/category-budgets?month="+currentMonthStr, "", http.StatusOK)
	var budgetsResp struct {
		Budgets []struct {
			LimitMinor     int64 `json:"limit_minor"`
			SpentMinor     int64 `json:"spent_minor"`
			RemainingMinor int64 `json:"remaining_minor"`
		} `json:"budgets"`
	}
	if err := json.Unmarshal(resBudgets, &budgetsResp); err != nil {
		t.Fatalf("parse budgets response: %v", err)
	}
	if len(budgetsResp.Budgets) != 1 {
		t.Fatalf("expected 1 budget, got %d", len(budgetsResp.Budgets))
	}
	// The member's transaction should NOT affect the owner's budget spent amount
	if budgetsResp.Budgets[0].SpentMinor != 0 {
		t.Fatalf("expected owner budget spent minor to be 0, got %d", budgetsResp.Budgets[0].SpentMinor)
	}
}
