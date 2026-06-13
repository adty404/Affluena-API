package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena/internal/config"
)

func TestListEndpointsSupportPaginationAndSorting(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "pagination-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "pagination")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	alphaWalletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Alpha Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 1000000
	}`)
	bravoWalletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Bravo Wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 0
	}`)
	createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Charlie Wallet",
		"type": "e_wallet",
		"currency_code": "IDR",
		"balance_minor": 0
	}`)
	incomeCategoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Pagination Income",
		"type": "income"
	}`)
	expenseCategoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Pagination Expense",
		"type": "expense"
	}`)

	createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+alphaWalletID+`",
		"category_id": "`+incomeCategoryID+`",
		"amount_minor": 500000,
		"transaction_at": "2026-06-01T08:00:00Z"
	}`)
	createAPIResource(t, router, token, "/api/v1/quick-entry-templates", `{
		"name": "Pagination Quick Entry",
		"type": "income",
		"wallet_id": "`+alphaWalletID+`",
		"category_id": "`+incomeCategoryID+`",
		"amount_minor": 75000
	}`)
	createAPIResource(t, router, token, "/api/v1/category-budgets", `{
		"category_id": "`+expenseCategoryID+`",
		"month": "2026-06",
		"limit_minor": 200000
	}`)
	createAPIResource(t, router, token, "/api/v1/debts", `{
		"type": "receivable",
		"counterparty_name": "Pagination Friend",
		"wallet_id": "`+alphaWalletID+`",
		"disbursement_category_id": "`+expenseCategoryID+`",
		"payment_category_id": "`+incomeCategoryID+`",
		"principal_amount_minor": 100000,
		"opened_at": "2026-06-13T08:00:00Z",
		"due_date": "2026-07-01"
	}`)
	createAPIResource(t, router, token, "/api/v1/installments", `{
		"name": "Pagination Installment",
		"wallet_id": "`+alphaWalletID+`",
		"category_id": "`+expenseCategoryID+`",
		"total_amount_minor": 600000,
		"monthly_amount_minor": 200000,
		"tenor_months": 3,
		"start_date": "2026-06-01",
		"due_day": 5
	}`)
	createAPIResource(t, router, token, "/api/v1/subscriptions", `{
		"name": "Pagination Subscription",
		"account_detail": "pagination@example.test",
		"wallet_id": "`+alphaWalletID+`",
		"category_id": "`+expenseCategoryID+`",
		"amount_minor": 250000,
		"billing_cycle": "monthly",
		"next_due_date": "2026-07-01"
	}`)
	createAPIResource(t, router, token, "/api/v1/recurring-transactions", `{
		"name": "Pagination Recurring",
		"type": "transfer",
		"wallet_id": "`+alphaWalletID+`",
		"to_wallet_id": "`+bravoWalletID+`",
		"amount_minor": 10000,
		"frequency": "monthly",
		"interval_count": 1,
		"next_run_at": "2030-01-01T00:00:00Z"
	}`)

	walletsResponse := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/wallets?limit=2&offset=1&sort=name_asc", "", http.StatusOK)
	assertWalletNames(t, walletsResponse, []string{"Bravo Wallet", "Charlie Wallet"})
	assertPagination(t, walletsResponse, "wallets", 2, 1, 3)

	listCases := []struct {
		name string
		path string
		key  string
	}{
		{name: "categories", path: "/api/v1/categories?limit=1&offset=0&sort=name_asc", key: "categories"},
		{name: "transactions", path: "/api/v1/transactions?limit=1&offset=0&sort=transaction_at_desc", key: "transactions"},
		{name: "quick entries", path: "/api/v1/quick-entry-templates?limit=1&offset=0&sort=name_asc", key: "templates"},
		{name: "budgets", path: "/api/v1/category-budgets?month=2026-06&limit=1&offset=0&sort=created_at_desc", key: "budgets"},
		{name: "debts", path: "/api/v1/debts?limit=1&offset=0&sort=opened_at_desc", key: "debts"},
		{name: "installments", path: "/api/v1/installments?limit=1&offset=0&sort=created_at_desc", key: "installments"},
		{name: "subscriptions", path: "/api/v1/subscriptions?limit=1&offset=0&sort=next_due_date_asc", key: "subscriptions"},
		{name: "recurring transactions", path: "/api/v1/recurring-transactions?limit=1&offset=0&sort=next_run_at_asc", key: "recurring_transactions"},
	}
	for _, tc := range listCases {
		t.Run(tc.name, func(t *testing.T) {
			response := performAPIRequest(t, router, token, http.MethodGet, tc.path, "", http.StatusOK)
			assertPagination(t, response, tc.key, 1, 0, 1)
		})
	}

	assertAPIStatus(t, router, token, http.MethodGet, "/api/v1/wallets?limit=0", "", http.StatusBadRequest)
	assertAPIStatus(t, router, token, http.MethodGet, "/api/v1/wallets?sort=unknown", "", http.StatusBadRequest)
}

func assertWalletNames(t *testing.T, response []byte, want []string) {
	t.Helper()

	var parsed struct {
		Wallets []struct {
			Name string `json:"name"`
		} `json:"wallets"`
	}
	if err := json.Unmarshal(response, &parsed); err != nil {
		t.Fatalf("parse wallet list response: %v", err)
	}
	if len(parsed.Wallets) != len(want) {
		t.Fatalf("expected %d wallets, got %d: %s", len(want), len(parsed.Wallets), string(response))
	}
	for i, expectedName := range want {
		if parsed.Wallets[i].Name != expectedName {
			t.Fatalf("expected wallet %d name %q, got %q", i, expectedName, parsed.Wallets[i].Name)
		}
	}
}

func assertPagination(t *testing.T, response []byte, itemKey string, wantLimit int, wantOffset int, minTotal int) {
	t.Helper()

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(response, &parsed); err != nil {
		t.Fatalf("parse paginated list response: %v", err)
	}
	if _, ok := parsed[itemKey]; !ok {
		t.Fatalf("response missing %q collection: %s", itemKey, string(response))
	}
	var items []json.RawMessage
	if err := json.Unmarshal(parsed[itemKey], &items); err != nil {
		t.Fatalf("parse %q collection: %v", itemKey, err)
	}
	if len(items) > wantLimit {
		t.Fatalf("expected at most %d %s items, got %d", wantLimit, itemKey, len(items))
	}

	var pagination struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
		Total  int `json:"total"`
	}
	rawPagination, ok := parsed["pagination"]
	if !ok {
		t.Fatalf("response missing pagination: %s", string(response))
	}
	if err := json.Unmarshal(rawPagination, &pagination); err != nil {
		t.Fatalf("parse pagination: %v", err)
	}
	if pagination.Limit != wantLimit || pagination.Offset != wantOffset {
		t.Fatalf("expected pagination limit/offset %d/%d, got %+v", wantLimit, wantOffset, pagination)
	}
	if pagination.Total < minTotal {
		t.Fatalf("expected pagination total >= %d, got %+v", minTotal, pagination)
	}
}
