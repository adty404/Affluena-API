package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
)

func TestListTransactionsSupportsFilters(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "transaction-filter-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "transaction-filter")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	mainWalletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Filter main wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 1000000
	}`)
	cashWalletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Filter cash wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 0
	}`)
	incomeCategoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Filter salary",
		"type": "income"
	}`)
	expenseCategoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Filter food",
		"type": "expense"
	}`)

	incomeID := createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+mainWalletID+`",
		"category_id": "`+incomeCategoryID+`",
		"amount_minor": 500000,
		"transaction_at": "2026-06-01T08:00:00Z"
	}`)
	mainExpenseID := createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+mainWalletID+`",
		"category_id": "`+expenseCategoryID+`",
		"amount_minor": 120000,
		"transaction_at": "2026-06-02T08:00:00Z"
	}`)
	cashExpenseID := createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+cashWalletID+`",
		"category_id": "`+expenseCategoryID+`",
		"amount_minor": 30000,
		"transaction_at": "2026-06-03T08:00:00Z"
	}`)
	transferID := createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "transfer",
		"wallet_id": "`+mainWalletID+`",
		"to_wallet_id": "`+cashWalletID+`",
		"amount_minor": 10000,
		"transaction_at": "2026-06-04T08:00:00Z"
	}`)

	expenseFiltered := listTransactionIDs(t, performAPIRequest(t, router, token, http.MethodGet, "/api/v1/transactions?type=expense", "", http.StatusOK))
	assertContainsTransactionIDs(t, expenseFiltered, mainExpenseID, cashExpenseID)
	assertMissingTransactionIDs(t, expenseFiltered, incomeID, transferID)

	cashFiltered := listTransactionIDs(t, performAPIRequest(t, router, token, http.MethodGet, "/api/v1/transactions?wallet_id="+cashWalletID, "", http.StatusOK))
	assertContainsTransactionIDs(t, cashFiltered, cashExpenseID, transferID)
	assertMissingTransactionIDs(t, cashFiltered, incomeID, mainExpenseID)

	categoryFiltered := listTransactionIDs(t, performAPIRequest(t, router, token, http.MethodGet, "/api/v1/transactions?category_id="+expenseCategoryID, "", http.StatusOK))
	assertContainsTransactionIDs(t, categoryFiltered, mainExpenseID, cashExpenseID)
	assertMissingTransactionIDs(t, categoryFiltered, incomeID, transferID)

	dateFiltered := listTransactionIDs(t, performAPIRequest(t, router, token, http.MethodGet, "/api/v1/transactions?from=2026-06-02&to=2026-06-03", "", http.StatusOK))
	assertContainsTransactionIDs(t, dateFiltered, mainExpenseID, cashExpenseID)
	assertMissingTransactionIDs(t, dateFiltered, incomeID, transferID)

	assertAPIStatus(t, router, token, http.MethodGet, "/api/v1/transactions?type=payday", "", http.StatusBadRequest)
	assertAPIStatus(t, router, token, http.MethodGet, "/api/v1/transactions?from=2026/06/01", "", http.StatusBadRequest)
}

func listTransactionIDs(t *testing.T, response []byte) map[string]struct{} {
	t.Helper()

	var parsed struct {
		Transactions []struct {
			ID string `json:"id"`
		} `json:"transactions"`
	}
	if err := json.Unmarshal(response, &parsed); err != nil {
		t.Fatalf("parse transaction list response: %v", err)
	}

	ids := make(map[string]struct{}, len(parsed.Transactions))
	for _, transaction := range parsed.Transactions {
		ids[transaction.ID] = struct{}{}
	}
	return ids
}

func assertContainsTransactionIDs(t *testing.T, ids map[string]struct{}, expected ...string) {
	t.Helper()
	for _, id := range expected {
		if _, ok := ids[id]; !ok {
			t.Fatalf("expected transaction %s in response, got %+v", id, ids)
		}
	}
}

func assertMissingTransactionIDs(t *testing.T, ids map[string]struct{}, forbidden ...string) {
	t.Helper()
	for _, id := range forbidden {
		if _, ok := ids[id]; ok {
			t.Fatalf("did not expect transaction %s in response: %+v", id, ids)
		}
	}
}
