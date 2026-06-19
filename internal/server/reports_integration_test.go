package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
	"affluena-api/internal/report"
)

func TestReportsIntegration(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "reports-integration-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	user, token := registerIntegrationAPIUser(t, router, "reportsuser")
	emptyUser, emptyToken := registerIntegrationAPIUser(t, router, "emptyreportsuser")
	defer cleanupServerIntegrationUsers(t, pool, user, emptyUser)

	// Create a wallet
	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Main Wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 10000000
	}`)

	// Create categories
	catIncome := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Salary",
		"type": "income"
	}`)
	catExpense := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Food",
		"type": "expense"
	}`)

	now := time.Now().UTC()
	currMonth := now.Format("2006-01")

	// Current month transactions
	createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+walletID+`",
		"category_id": "`+catIncome+`",
		"amount_minor": 500000,
		"transaction_at": "`+now.Format(time.RFC3339)+`"
	}`)

	createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+catExpense+`",
		"amount_minor": 200000,
		"transaction_at": "`+now.Format(time.RFC3339)+`"
	}`)

	// Previous month transactions
	prevMonth := now.AddDate(0, -1, 0)
	createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+walletID+`",
		"category_id": "`+catIncome+`",
		"amount_minor": 400000,
		"transaction_at": "`+prevMonth.Format(time.RFC3339)+`"
	}`)

	createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+catExpense+`",
		"amount_minor": 100000,
		"transaction_at": "`+prevMonth.Format(time.RFC3339)+`"
	}`)

	// Create debt
	createAPIResource(t, router, token, "/api/v1/debts", `{
		"type": "payable",
		"counterparty_name": "John Doe",
		"principal_amount_minor": 100000,
		"wallet_id": "`+walletID+`",
		"disbursement_category_id": "`+catIncome+`",
		"payment_category_id": "`+catExpense+`",
		"opened_at": "`+now.Format(time.RFC3339)+`"
	}`)

	goalID := createAPIResource(t, router, token, "/api/v1/goals", `{
		"name": "Vacation",
		"target_amount_minor": 200000,
		"deadline": "`+now.AddDate(1, 0, 0).Format(time.RFC3339)+`"
	}`)

	_, err := pool.Exec(context.Background(), `UPDATE wallets SET balance_minor = 50000 WHERE goal_id = $1`, goalID)
	if err != nil {
		t.Fatalf("Failed to update goal wallet balance: %v", err)
	}

	// 1. Test Income Report
	resIncome := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/reports/income?month="+currMonth, "", http.StatusOK)
	var incomeResp report.ReportResponse
	json.Unmarshal(resIncome, &incomeResp)

	if len(incomeResp.Metrics) == 0 || len(incomeResp.Rows) == 0 {
		t.Fatalf("Expected income metrics and rows")
	}

	var totalIncome int64
	for _, m := range incomeResp.Metrics {
		if m.ID == "total_income" {
			totalIncome = m.ValueMinor
		}
	}
	if totalIncome != 600000 {
		t.Fatalf("Expected total income 600000, got %d", totalIncome)
	}
	if incomeResp.Rows[0].Name != "Salary" || incomeResp.Rows[0].AmountMinor != 600000 || incomeResp.Rows[0].PreviousAmountMinor != 400000 {
		t.Fatalf("Income row mismatch: %+v", incomeResp.Rows[0])
	}

	// 2. Test Expense Report
	resExpense := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/reports/expense?month="+currMonth, "", http.StatusOK)
	var expenseResp report.ReportResponse
	json.Unmarshal(resExpense, &expenseResp)

	if len(expenseResp.Metrics) == 0 || len(expenseResp.Rows) == 0 {
		t.Fatalf("Expected expense metrics and rows")
	}
	if expenseResp.Rows[0].Name != "Food" || expenseResp.Rows[0].AmountMinor != 200000 || expenseResp.Rows[0].PreviousAmountMinor != 100000 {
		t.Fatalf("Expense row mismatch: %+v", expenseResp.Rows[0])
	}

	// 3. Test Cashflow Report
	resCashflow := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/reports/cashflow?month="+currMonth, "", http.StatusOK)
	var cashflowResp report.ReportResponse
	json.Unmarshal(resCashflow, &cashflowResp)

	if len(cashflowResp.Metrics) == 0 || len(cashflowResp.Rows) != 4 {
		t.Fatalf("Expected cashflow metrics and 4 rows")
	}
	var netCashflow int64
	for _, m := range cashflowResp.Metrics {
		if m.ID == "net_cashflow" {
			netCashflow = m.ValueMinor
		}
	}
	if netCashflow != 400000 {
		t.Fatalf("Expected net cashflow 400000, got %d", netCashflow)
	}

	// 4. Test Debt Report
	resDebt := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/reports/debt?month="+currMonth, "", http.StatusOK)
	var debtResp report.ReportResponse
	json.Unmarshal(resDebt, &debtResp)

	if len(debtResp.Metrics) == 0 || len(debtResp.Rows) == 0 {
		t.Fatalf("Expected debt metrics and rows")
	}
	if debtResp.Rows[0].Name != "John Doe" || debtResp.Rows[0].AmountMinor != 100000 {
		t.Fatalf("Debt row mismatch: %+v", debtResp.Rows[0])
	}

	// 5. Test Goal Report
	resGoal := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/reports/goal?month="+currMonth, "", http.StatusOK)
	var goalResp report.ReportResponse
	json.Unmarshal(resGoal, &goalResp)

	if len(goalResp.Metrics) == 0 || len(goalResp.Rows) == 0 {
		t.Fatalf("Expected goal metrics and rows")
	}
	if goalResp.Rows[0].Name != "Vacation" || goalResp.Rows[0].AmountMinor != 50000 {
		t.Fatalf("Goal row mismatch: %+v", goalResp.Rows[0])
	}

	// 6. Test Overview Report
	resOverview := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/reports/overview?month="+currMonth, "", http.StatusOK)
	var overviewResp report.ReportResponse
	json.Unmarshal(resOverview, &overviewResp)

	if len(overviewResp.Metrics) == 0 || len(overviewResp.Rows) == 0 {
		t.Fatalf("Expected overview metrics and rows")
	}

	// 7. Test Empty User Isolation
	resEmpty := performAPIRequest(t, router, emptyToken, http.MethodGet, "/api/v1/reports/income?month="+currMonth, "", http.StatusOK)
	var emptyResp report.ReportResponse
	json.Unmarshal(resEmpty, &emptyResp)

	if len(emptyResp.Metrics) == 0 {
		t.Fatalf("Expected default metrics for empty user")
	}
	if len(emptyResp.Rows) != 0 {
		t.Fatalf("Expected 0 rows for empty user, got %d", len(emptyResp.Rows))
	}

	var emptyTotal int64
	for _, m := range emptyResp.Metrics {
		if m.ID == "total_income" {
			emptyTotal = m.ValueMinor
		}
	}
	if emptyTotal != 0 {
		t.Fatalf("Expected 0 total income for empty user, got %d", emptyTotal)
	}
}
