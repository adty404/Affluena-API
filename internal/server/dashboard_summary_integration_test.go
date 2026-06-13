package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
)

func TestDashboardSummaryAggregatesUserMonth(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "dashboard-summary-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userAID, userAToken := registerIntegrationAPIUser(t, router, "dashboard-owner-a")
	userBID, userBToken := registerIntegrationAPIUser(t, router, "dashboard-owner-b")
	defer cleanupServerIntegrationUsers(t, pool, userAID, userBID)

	walletID := createAPIResource(t, router, userAToken, "/api/v1/wallets", `{
		"name": "Dashboard wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 1000000
	}`)
	incomeCategoryID := createAPIResource(t, router, userAToken, "/api/v1/categories", `{
		"name": "Dashboard salary",
		"type": "income"
	}`)
	expenseCategoryID := createAPIResource(t, router, userAToken, "/api/v1/categories", `{
		"name": "Dashboard food",
		"type": "expense"
	}`)
	loanCategoryID := createAPIResource(t, router, userAToken, "/api/v1/categories", `{
		"name": "Dashboard loans",
		"type": "expense"
	}`)

	createAPIResource(t, router, userAToken, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+walletID+`",
		"category_id": "`+incomeCategoryID+`",
		"amount_minor": 500000,
		"transaction_at": "2026-06-05T08:00:00Z"
	}`)
	createAPIResource(t, router, userAToken, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+expenseCategoryID+`",
		"amount_minor": 120000,
		"transaction_at": "2026-06-06T08:00:00Z"
	}`)
	createAPIResource(t, router, userAToken, "/api/v1/category-budgets", `{
		"category_id": "`+expenseCategoryID+`",
		"month": "2026-06",
		"limit_minor": 200000
	}`)
	createAPIResource(t, router, userAToken, "/api/v1/subscriptions", `{
		"name": "Google One",
		"account_detail": "personal@example.com",
		"wallet_id": "`+walletID+`",
		"category_id": "`+expenseCategoryID+`",
		"amount_minor": 26900,
		"billing_cycle": "monthly",
		"next_due_date": "2026-06-20",
		"note": "Dashboard subscription"
	}`)
	createAPIResource(t, router, userAToken, "/api/v1/installments", `{
		"name": "Laptop",
		"wallet_id": "`+walletID+`",
		"category_id": "`+expenseCategoryID+`",
		"total_amount_minor": 300000,
		"monthly_amount_minor": 100000,
		"tenor_months": 3,
		"start_date": "2026-06-01",
		"due_day": 15
	}`)
	createAPIResource(t, router, userAToken, "/api/v1/debts", `{
		"type": "receivable",
		"counterparty_name": "Friend",
		"wallet_id": "`+walletID+`",
		"disbursement_category_id": "`+loanCategoryID+`",
		"payment_category_id": "`+incomeCategoryID+`",
		"principal_amount_minor": 50000,
		"opened_at": "2026-06-07T08:00:00Z",
		"due_date": "2026-06-25"
	}`)

	otherWalletID := createAPIResource(t, router, userBToken, "/api/v1/wallets", `{
		"name": "Other dashboard wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 999999
	}`)
	otherCategoryID := createAPIResource(t, router, userBToken, "/api/v1/categories", `{
		"name": "Other dashboard salary",
		"type": "income"
	}`)
	createAPIResource(t, router, userBToken, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+otherWalletID+`",
		"category_id": "`+otherCategoryID+`",
		"amount_minor": 999999,
		"transaction_at": "2026-06-05T08:00:00Z"
	}`)

	response := performAPIRequest(t, router, userAToken, http.MethodGet, "/api/v1/dashboard/summary?month=2026-06", "", http.StatusOK)
	var summary dashboardSummaryResponse
	if err := json.Unmarshal(response, &summary); err != nil {
		t.Fatalf("parse dashboard summary response: %v", err)
	}

	if summary.Month != "2026-06" {
		t.Fatalf("expected month 2026-06, got %q", summary.Month)
	}
	if summary.NetWorthMinor != 1330000 {
		t.Fatalf("expected net worth 1330000, got %d", summary.NetWorthMinor)
	}
	if summary.MonthlyIncomeMinor != 500000 {
		t.Fatalf("expected monthly income 500000, got %d", summary.MonthlyIncomeMinor)
	}
	if summary.MonthlyExpenseMinor != 170000 {
		t.Fatalf("expected monthly expense 170000, got %d", summary.MonthlyExpenseMinor)
	}
	if summary.MonthlyCashflowMinor != 330000 {
		t.Fatalf("expected monthly cashflow 330000, got %d", summary.MonthlyCashflowMinor)
	}
	if summary.Budget.LimitMinor != 200000 || summary.Budget.SpentMinor != 120000 || summary.Budget.RemainingMinor != 80000 {
		t.Fatalf("unexpected budget summary %+v", summary.Budget)
	}
	if len(summary.UpcomingSubscriptions) != 1 || summary.UpcomingSubscriptions[0].Name != "Google One" || summary.UpcomingSubscriptions[0].AccountDetail != "personal@example.com" {
		t.Fatalf("unexpected upcoming subscriptions %+v", summary.UpcomingSubscriptions)
	}
	if len(summary.UpcomingInstallments) != 1 || summary.UpcomingInstallments[0].Name != "Laptop" || summary.UpcomingInstallments[0].MonthlyAmountMinor != 100000 {
		t.Fatalf("unexpected upcoming installments %+v", summary.UpcomingInstallments)
	}
	if len(summary.UpcomingDebts) != 1 || summary.UpcomingDebts[0].CounterpartyName != "Friend" || summary.UpcomingDebts[0].RemainingAmountMinor != 50000 {
		t.Fatalf("unexpected upcoming debts %+v", summary.UpcomingDebts)
	}

	otherSummaryResponse := performAPIRequest(t, router, userBToken, http.MethodGet, "/api/v1/dashboard/summary?month=2026-06", "", http.StatusOK)
	var otherSummary dashboardSummaryResponse
	if err := json.Unmarshal(otherSummaryResponse, &otherSummary); err != nil {
		t.Fatalf("parse other dashboard summary response: %v", err)
	}
	if otherSummary.MonthlyIncomeMinor != 999999 || otherSummary.MonthlyExpenseMinor != 0 {
		t.Fatalf("expected dashboard to be scoped to other user data, got %+v", otherSummary)
	}

	assertAPIStatus(t, router, userAToken, http.MethodGet, "/api/v1/dashboard/summary?month=2026/06", "", http.StatusBadRequest)
}

type dashboardSummaryResponse struct {
	Month                string `json:"month"`
	NetWorthMinor        int64  `json:"net_worth_minor"`
	MonthlyIncomeMinor   int64  `json:"monthly_income_minor"`
	MonthlyExpenseMinor  int64  `json:"monthly_expense_minor"`
	MonthlyCashflowMinor int64  `json:"monthly_cashflow_minor"`
	Budget               struct {
		LimitMinor     int64   `json:"limit_minor"`
		SpentMinor     int64   `json:"spent_minor"`
		RemainingMinor int64   `json:"remaining_minor"`
		UsagePercent   float64 `json:"usage_percent"`
	} `json:"budget"`
	UpcomingSubscriptions []struct {
		Name          string `json:"name"`
		AccountDetail string `json:"account_detail"`
	} `json:"upcoming_subscriptions"`
	UpcomingInstallments []struct {
		Name               string `json:"name"`
		MonthlyAmountMinor int64  `json:"monthly_amount_minor"`
	} `json:"upcoming_installments"`
	UpcomingDebts []struct {
		CounterpartyName     string `json:"counterparty_name"`
		RemainingAmountMinor int64  `json:"remaining_amount_minor"`
	} `json:"upcoming_debts"`
}
