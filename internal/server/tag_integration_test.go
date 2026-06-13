package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
	"affluena-api/internal/tag"
	"affluena-api/internal/transaction"
)

func TestTagLifecycleAndTransactionIntegration(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "tag-integration-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	user, token := registerIntegrationAPIUser(t, router, "taguser")
	defer cleanupServerIntegrationUsers(t, pool, user)

	// --- 1. Create Tags ---
	tagA := createAPIResource(t, router, token, "/api/v1/tags", `{
		"name": "Vacation2026"
	}`)
	tagB := createAPIResource(t, router, token, "/api/v1/tags", `{
		"name": "Bali Trip"
	}`)

	// --- 2. List Tags ---
	resTags := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/tags", "", http.StatusOK)
	var listResp struct {
		Tags []tag.Tag `json:"tags"`
	}
	json.Unmarshal(resTags, &listResp)
	if len(listResp.Tags) != 2 {
		t.Fatalf("Expected 2 tags, got %d", len(listResp.Tags))
	}

	// --- 3. Create Prerequisites for Transaction ---
	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Cash Wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 0
	}`)
	categoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Salary",
		"type": "income"
	}`)

	// --- 4. Create Transaction with Tags ---
	transactionID := createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 10000000,
		"tag_ids": ["`+tagA+`", "`+tagB+`"]
	}`)

	// --- 5. Verify Transaction contains Tags ---
	resTx := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/transactions/"+transactionID, "", http.StatusOK)
	var tx transaction.Transaction
	json.Unmarshal(resTx, &tx)
	if len(tx.TagIDs) != 2 {
		t.Fatalf("Expected transaction to have 2 tags, got %d", len(tx.TagIDs))
	}

	// --- 6. Filter Transactions by Tag ---
	resFilter := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/transactions?tag_id="+tagA, "", http.StatusOK)
	var txListResp struct {
		Transactions []transaction.Transaction `json:"transactions"`
	}
	json.Unmarshal(resFilter, &txListResp)
	if len(txListResp.Transactions) != 1 {
		t.Fatalf("Expected 1 transaction matching tag A, got %d", len(txListResp.Transactions))
	}

	resFilterMiss := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/transactions?tag_id=00000000-0000-0000-0000-000000000000", "", http.StatusOK)
	json.Unmarshal(resFilterMiss, &txListResp)
	if len(txListResp.Transactions) != 0 {
		t.Fatalf("Expected 0 transaction matching invalid tag, got %d", len(txListResp.Transactions))
	}

	// --- 7. Update Transaction (Remove one tag) ---
	performAPIRequest(t, router, token, http.MethodPut, "/api/v1/transactions/"+transactionID, `{
		"type": "income",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 10000000,
		"tag_ids": ["`+tagB+`"]
	}`, http.StatusOK)

	resTxUpdated := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/transactions/"+transactionID, "", http.StatusOK)
	json.Unmarshal(resTxUpdated, &tx)
	if len(tx.TagIDs) != 1 || tx.TagIDs[0] != tagB {
		t.Fatalf("Expected transaction to have 1 tag (tagB)")
	}

	// --- 8. Delete Tag (Should Cascade) ---
	performAPIRequest(t, router, token, http.MethodDelete, "/api/v1/tags/"+tagB, "", http.StatusNoContent)

	resTxCascade := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/transactions/"+transactionID, "", http.StatusOK)
	json.Unmarshal(resTxCascade, &tx)
	if len(tx.TagIDs) != 0 {
		t.Fatalf("Expected transaction to have 0 tags after tag deletion due to cascade")
	}
}
