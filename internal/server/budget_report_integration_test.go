package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/budget"
	"affluena-api/internal/config"
)

func TestBudgetAlertsAndReportIntegration(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "budget-integration-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	user, token := registerIntegrationAPIUser(t, router, "budgetreportuser")
	defer cleanupServerIntegrationUsers(t, pool, user)

	// Create a wallet
	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Cash Wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 10000000
	}`)

	// Create categories
	catSafe := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Safe Category",
		"type": "expense"
	}`)
	catWarning := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Warning Category",
		"type": "expense"
	}`)
	catDanger := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Danger Category",
		"type": "expense"
	}`)

	// Create budgets for current month
	now := time.Now().UTC()
	monthValue := now.Format("2006-01")

	createAPIResource(t, router, token, "/api/v1/category-budgets", `{
		"category_id": "`+catSafe+`",
		"month": "`+monthValue+`",
		"limit_minor": 100000
	}`)
	createAPIResource(t, router, token, "/api/v1/category-budgets", `{
		"category_id": "`+catWarning+`",
		"month": "`+monthValue+`",
		"limit_minor": 100000
	}`)
	createAPIResource(t, router, token, "/api/v1/category-budgets", `{
		"category_id": "`+catDanger+`",
		"month": "`+monthValue+`",
		"limit_minor": 100000
	}`)

	// Add transactions for current month
	// Safe: 79% (79,000)
	createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+catSafe+`",
		"amount_minor": 79000,
		"transaction_at": "`+now.Format(time.RFC3339)+`"
	}`)

	// Warning: 80% (80,000)
	createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+catWarning+`",
		"amount_minor": 80000,
		"transaction_at": "`+now.Format(time.RFC3339)+`"
	}`)

	// Danger: 100% (100,000)
	createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+catDanger+`",
		"amount_minor": 100000,
		"transaction_at": "`+now.Format(time.RFC3339)+`"
	}`)

	// 1. Test Alerts endpoint
	resAlerts := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/category-budgets/alerts?month="+monthValue, "", http.StatusOK)
	var alertsResp struct {
		Alerts []budget.BudgetAlert `json:"alerts"`
	}
	json.Unmarshal(resAlerts, &alertsResp)

	if len(alertsResp.Alerts) != 2 {
		t.Fatalf("Expected 2 alerts, got %d", len(alertsResp.Alerts))
	}

	var hasWarning, hasDanger bool
	for _, a := range alertsResp.Alerts {
		if a.CategoryID == catWarning {
			hasWarning = true
			if a.Severity != "warning" || a.Threshold != 80 {
				t.Errorf("expected warning severity and 80 threshold, got %s %d", a.Severity, a.Threshold)
			}
		}
		if a.CategoryID == catDanger {
			hasDanger = true
			if a.Severity != "danger" || a.Threshold != 100 {
				t.Errorf("expected danger severity and 100 threshold, got %s %d", a.Severity, a.Threshold)
			}
		}
	}
	if !hasWarning || !hasDanger {
		t.Errorf("missing expected alerts: warning=%v, danger=%v", hasWarning, hasDanger)
	}

	// 2. Test Report endpoint
	resReport := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/category-budgets/report?month="+monthValue, "", http.StatusOK)
	var reportResp struct {
		Report  []budget.BudgetReportItem  `json:"report"`
		Summary budget.BudgetReportSummary `json:"summary"`
	}
	json.Unmarshal(resReport, &reportResp)

	if len(reportResp.Report) != 3 {
		t.Fatalf("Expected 3 report items, got %d", len(reportResp.Report))
	}

	if reportResp.Summary.TotalLimitMinor != 300000 {
		t.Errorf("expected total limit 300,000, got %d", reportResp.Summary.TotalLimitMinor)
	}
	if reportResp.Summary.TotalSpentMinor != 259000 {
		t.Errorf("expected total spent 259,000, got %d", reportResp.Summary.TotalSpentMinor)
	}
	if reportResp.Summary.SafeCount != 1 || reportResp.Summary.WarningCount != 1 || reportResp.Summary.ExceededCount != 1 {
		t.Errorf("expected counts 1/1/1, got %d/%d/%d", reportResp.Summary.SafeCount, reportResp.Summary.WarningCount, reportResp.Summary.ExceededCount)
	}

	for _, item := range reportResp.Report {
		if item.CategoryID == catSafe && item.Recommendation != "safe" {
			t.Errorf("expected safe recommendation, got %s", item.Recommendation)
		}
		if item.CategoryID == catWarning && item.Recommendation != "warning" {
			t.Errorf("expected warning recommendation, got %s", item.Recommendation)
		}
		if item.CategoryID == catDanger && item.Recommendation != "exceeded" {
			t.Errorf("expected exceeded recommendation, got %s", item.Recommendation)
		}
	}

	// 3. Test isolation
	user2, token2 := registerIntegrationAPIUser(t, router, "budgetreportuser2")
	defer cleanupServerIntegrationUsers(t, pool, user2)

	resAlerts2 := performAPIRequest(t, router, token2, http.MethodGet, "/api/v1/category-budgets/alerts?month="+monthValue, "", http.StatusOK)
	var alertsResp2 struct {
		Alerts []budget.BudgetAlert `json:"alerts"`
	}
	json.Unmarshal(resAlerts2, &alertsResp2)
	if len(alertsResp2.Alerts) != 0 {
		t.Errorf("expected 0 alerts for new user, got %d", len(alertsResp2.Alerts))
	}
}
