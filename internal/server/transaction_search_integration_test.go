package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"affluena-api/internal/config"
)

// TestListTransactionsSearchFilter proves the free-text `search` param on
// GET /api/v1/transactions: case-insensitive substring over the transaction
// note, its category name, or its source wallet name; LIKE metacharacters
// match literally; results stay scoped to the caller; and the filter composes
// with the existing `type` filter while keeping pagination totals in sync.
func TestListTransactionsSearchFilter(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "transaction-search-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userAID, tokenA := registerIntegrationAPIUser(t, router, "tx-search-a")
	userBID, tokenB := registerIntegrationAPIUser(t, router, "tx-search-b")
	defer cleanupServerIntegrationUsers(t, pool, userAID, userBID)

	bankWalletID := createAPIResource(t, router, tokenA, "/api/v1/wallets", `{
		"name": "Rekening Gajian",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 1000000
	}`)
	cashWalletID := createAPIResource(t, router, tokenA, "/api/v1/wallets", `{
		"name": "Kantong Harian",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 500000
	}`)
	coffeeCategoryID := createAPIResource(t, router, tokenA, "/api/v1/categories", `{
		"name": "Ngopi Cantik",
		"type": "expense"
	}`)
	miscExpenseCategoryID := createAPIResource(t, router, tokenA, "/api/v1/categories", `{
		"name": "Serba Serbi",
		"type": "expense"
	}`)
	miscIncomeCategoryID := createAPIResource(t, router, tokenA, "/api/v1/categories", `{
		"name": "Rejeki Nomplok",
		"type": "income"
	}`)

	txByNote := createAPIResource(t, router, tokenA, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+bankWalletID+`",
		"category_id": "`+miscExpenseCategoryID+`",
		"amount_minor": 42000,
		"note": "beli gula aren premium"
	}`)
	txByCategory := createAPIResource(t, router, tokenA, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+bankWalletID+`",
		"category_id": "`+coffeeCategoryID+`",
		"amount_minor": 30000,
		"note": "es susu favorit"
	}`)
	txByWallet := createAPIResource(t, router, tokenA, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+cashWalletID+`",
		"category_id": "`+miscExpenseCategoryID+`",
		"amount_minor": 15000,
		"note": "jajan sore"
	}`)
	txPercentExpense := createAPIResource(t, router, tokenA, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+bankWalletID+`",
		"category_id": "`+miscExpenseCategoryID+`",
		"amount_minor": 99000,
		"note": "promo 50% akhir pekan"
	}`)
	txPercentDecoyIncome := createAPIResource(t, router, tokenA, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+bankWalletID+`",
		"category_id": "`+miscIncomeCategoryID+`",
		"amount_minor": 50500,
		"note": "promo 505 akhir pekan"
	}`)

	// Another user with a similar note: their rows must never surface for A.
	walletBID := createAPIResource(t, router, tokenB, "/api/v1/wallets", `{
		"name": "Dompet B",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 100000
	}`)
	categoryBID := createAPIResource(t, router, tokenB, "/api/v1/categories", `{
		"name": "Dapur B",
		"type": "expense"
	}`)
	txUserB := createAPIResource(t, router, tokenB, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletBID+`",
		"category_id": "`+categoryBID+`",
		"amount_minor": 12000,
		"note": "beli gula merah"
	}`)

	searchURL := func(query string, extra string) string {
		return "/api/v1/transactions?search=" + url.QueryEscape(query) + extra
	}

	// Match by note, case-insensitive, trimmed.
	byNote, total := listSearchTransactionIDs(t, router, tokenA, searchURL("  GULA AREN ", ""))
	assertContainsTransactionIDs(t, byNote, txByNote)
	assertMissingTransactionIDs(t, byNote, txByCategory, txByWallet, txPercentExpense, txPercentDecoyIncome, txUserB)
	if total != 1 {
		t.Fatalf("expected search pagination total 1, got %d", total)
	}

	// Match by category name (the note does not contain the term).
	byCategory, _ := listSearchTransactionIDs(t, router, tokenA, searchURL("ngopi", ""))
	assertContainsTransactionIDs(t, byCategory, txByCategory)
	assertMissingTransactionIDs(t, byCategory, txByNote, txByWallet, txPercentExpense, txPercentDecoyIncome)

	// Match by source wallet name.
	byWallet, _ := listSearchTransactionIDs(t, router, tokenA, searchURL("kantong", ""))
	assertContainsTransactionIDs(t, byWallet, txByWallet)
	assertMissingTransactionIDs(t, byWallet, txByNote, txByCategory, txPercentExpense, txPercentDecoyIncome)

	// A literal % must not act as a wildcard: "50%" only matches the note
	// containing the actual percent sign, not "505...".
	byPercent, percentTotal := listSearchTransactionIDs(t, router, tokenA, searchURL("50%", ""))
	assertContainsTransactionIDs(t, byPercent, txPercentExpense)
	assertMissingTransactionIDs(t, byPercent, txPercentDecoyIncome)
	if percentTotal != 1 {
		t.Fatalf("expected literal-percent search total 1, got %d", percentTotal)
	}

	// Composes with the type filter: "promo" alone matches both types, adding
	// type=income narrows to the income row (list and total in sync).
	promoAll, promoAllTotal := listSearchTransactionIDs(t, router, tokenA, searchURL("promo", ""))
	assertContainsTransactionIDs(t, promoAll, txPercentExpense, txPercentDecoyIncome)
	if promoAllTotal != 2 {
		t.Fatalf("expected promo search total 2, got %d", promoAllTotal)
	}
	promoIncome, promoIncomeTotal := listSearchTransactionIDs(t, router, tokenA, searchURL("promo", "&type=income"))
	assertContainsTransactionIDs(t, promoIncome, txPercentDecoyIncome)
	assertMissingTransactionIDs(t, promoIncome, txPercentExpense)
	if promoIncomeTotal != 1 {
		t.Fatalf("expected promo+income search total 1, got %d", promoIncomeTotal)
	}

	// Isolation both ways: B only sees their own "gula" row.
	gulaForB, _ := listSearchTransactionIDs(t, router, tokenB, searchURL("gula", ""))
	assertContainsTransactionIDs(t, gulaForB, txUserB)
	assertMissingTransactionIDs(t, gulaForB, txByNote)

	// Over the 100-character cap -> 400.
	assertAPIStatus(t, router, tokenA, http.MethodGet, searchURL(strings.Repeat("a", 101), ""), "", http.StatusBadRequest)
}

func listSearchTransactionIDs(t *testing.T, router http.Handler, token string, path string) (map[string]struct{}, int) {
	t.Helper()

	response := performAPIRequest(t, router, token, http.MethodGet, path, "", http.StatusOK)
	var parsed struct {
		Transactions []struct {
			ID string `json:"id"`
		} `json:"transactions"`
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(response, &parsed); err != nil {
		t.Fatalf("parse transaction search response: %v", err)
	}

	ids := make(map[string]struct{}, len(parsed.Transactions))
	for _, transaction := range parsed.Transactions {
		ids[transaction.ID] = struct{}{}
	}
	return ids, parsed.Pagination.Total
}
