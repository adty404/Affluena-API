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

func TestTransactionTagsRejectCrossUserAndInvalidTagsAtomically(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "tag-isolation-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userA, tokenA := registerIntegrationAPIUser(t, router, "tag-owner-a")
	userB, tokenB := registerIntegrationAPIUser(t, router, "tag-owner-b")
	defer cleanupServerIntegrationUsers(t, pool, userA, userB)

	walletA := createAPIResource(t, router, tokenA, "/api/v1/wallets", `{
		"name": "Owner A wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 100000
	}`)
	categoryA := createAPIResource(t, router, tokenA, "/api/v1/categories", `{
		"name": "Owner A salary",
		"type": "income"
	}`)
	tagA := createAPIResource(t, router, tokenA, "/api/v1/tags", `{"name": "OwnerATag"}`)
	tagB := createAPIResource(t, router, tokenB, "/api/v1/tags", `{"name": "OwnerBTag"}`)

	assertAPIStatus(t, router, tokenA, http.MethodPost, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+walletA+`",
		"category_id": "`+categoryA+`",
		"amount_minor": 50000,
		"tag_ids": ["`+tagB+`"],
		"transaction_at": "2026-06-13T08:00:00Z"
	}`, http.StatusNotFound)
	assertWalletBalance(t, router, tokenA, walletA, 100000)

	transactionID := createAPIResource(t, router, tokenA, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+walletA+`",
		"category_id": "`+categoryA+`",
		"amount_minor": 25000,
		"tag_ids": ["`+tagA+`"],
		"transaction_at": "2026-06-13T09:00:00Z"
	}`)
	assertWalletBalance(t, router, tokenA, walletA, 125000)

	assertAPIStatus(t, router, tokenA, http.MethodPut, "/api/v1/transactions/"+transactionID, `{
		"type": "income",
		"wallet_id": "`+walletA+`",
		"category_id": "`+categoryA+`",
		"amount_minor": 30000,
		"tag_ids": ["`+tagB+`"],
		"transaction_at": "2026-06-13T09:00:00Z"
	}`, http.StatusNotFound)
	assertWalletBalance(t, router, tokenA, walletA, 125000)

	resTx := performAPIRequest(t, router, tokenA, http.MethodGet, "/api/v1/transactions/"+transactionID, "", http.StatusOK)
	var tx transaction.Transaction
	json.Unmarshal(resTx, &tx)
	if tx.AmountMinor != 25000 || len(tx.TagIDs) != 1 || tx.TagIDs[0] != tagA {
		t.Fatalf("expected original transaction to remain unchanged, got %+v", tx)
	}

	assertAPIStatus(t, router, tokenA, http.MethodPost, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+walletA+`",
		"category_id": "`+categoryA+`",
		"amount_minor": 50000,
		"tag_ids": ["00000000-0000-0000-0000-000000000000"],
		"transaction_at": "2026-06-13T10:00:00Z"
	}`, http.StatusNotFound)
	assertWalletBalance(t, router, tokenA, walletA, 125000)
}

func TestTransactionTagsDeduplicateAndValidateFilter(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "tag-edge-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	user, token := registerIntegrationAPIUser(t, router, "tag-edge")
	defer cleanupServerIntegrationUsers(t, pool, user)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Tag edge wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 0
	}`)
	categoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Tag edge salary",
		"type": "income"
	}`)
	tagID := createAPIResource(t, router, token, "/api/v1/tags", `{"name": "DedupedTag"}`)

	transactionID := createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 1000,
		"tag_ids": ["`+tagID+`", "`+tagID+`"],
		"transaction_at": "2026-06-13T08:00:00Z"
	}`)

	resTx := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/transactions/"+transactionID, "", http.StatusOK)
	var tx transaction.Transaction
	json.Unmarshal(resTx, &tx)
	if len(tx.TagIDs) != 1 || tx.TagIDs[0] != tagID {
		t.Fatalf("expected duplicate tag IDs to be stored once, got %+v", tx.TagIDs)
	}

	assertAPIStatus(t, router, token, http.MethodGet, "/api/v1/transactions?tag_id=not-a-uuid", "", http.StatusBadRequest)
}

func TestSharedWalletTagFilterDoesNotUseOtherUsersTags(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "tag-shared-filter-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	ownerID, ownerToken := registerIntegrationAPIUser(t, router, "tag-shared-owner")
	memberID, memberToken := registerIntegrationAPIUser(t, router, "tag-shared-member")
	defer cleanupServerIntegrationUsers(t, pool, ownerID)
	defer cleanupServerIntegrationUsers(t, pool, memberID)

	sharedWalletID := createAPIResource(t, router, ownerToken, "/api/v1/wallets", `{
		"name": "Tagged shared wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 100000
	}`)
	memberEmail := getIntegrationUserEmail(t, router, memberToken)
	assertAPIStatus(t, router, ownerToken, http.MethodPost, "/api/v1/wallets/"+sharedWalletID+"/invites", `{
		"email": "`+memberEmail+`"
	}`, http.StatusCreated)
	assertAPIStatus(t, router, memberToken, http.MethodPatch, "/api/v1/wallets/"+sharedWalletID+"/members/"+memberID, `{
		"status": "joined"
	}`, http.StatusOK)

	memberTagID := createAPIResource(t, router, memberToken, "/api/v1/tags", `{"name": "MemberPrivateTag"}`)
	memberCategoryID := createAPIResource(t, router, memberToken, "/api/v1/categories", `{
		"name": "Member tagged expense",
		"type": "expense"
	}`)
	transactionID := createAPIResource(t, router, memberToken, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+sharedWalletID+`",
		"category_id": "`+memberCategoryID+`",
		"amount_minor": 10000,
		"tag_ids": ["`+memberTagID+`"],
		"transaction_at": "2026-06-14T08:00:00Z"
	}`)

	ownerListBody := performAPIRequest(t, router, ownerToken, http.MethodGet, "/api/v1/transactions?wallet_id="+sharedWalletID, "", http.StatusOK)
	var ownerList struct {
		Transactions []transaction.Transaction `json:"transactions"`
	}
	if err := json.Unmarshal(ownerListBody, &ownerList); err != nil {
		t.Fatalf("parse owner shared wallet transactions: %v", err)
	}
	if countTransactionID(ownerList.Transactions, transactionID) != 1 {
		t.Fatalf("expected owner to see shared wallet transaction %s, got %+v", transactionID, ownerList.Transactions)
	}

	ownerFilteredBody := performAPIRequest(t, router, ownerToken, http.MethodGet, "/api/v1/transactions?tag_id="+memberTagID, "", http.StatusOK)
	var ownerFiltered struct {
		Transactions []transaction.Transaction `json:"transactions"`
	}
	if err := json.Unmarshal(ownerFilteredBody, &ownerFiltered); err != nil {
		t.Fatalf("parse owner filtered transactions: %v", err)
	}
	if len(ownerFiltered.Transactions) != 0 {
		t.Fatalf("expected owner filter with member tag to return no transactions, got %+v", ownerFiltered.Transactions)
	}
}
