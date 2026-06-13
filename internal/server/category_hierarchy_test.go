package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
)

func TestCategoryHierarchyIntegration(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "cat-integration-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	user, token := registerIntegrationAPIUser(t, router, "hierarchy_user")
	defer cleanupServerIntegrationUsers(t, pool, user)

	// Create Level 1 Category (Food)
	l1ID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Food",
		"type": "expense"
	}`)

	// Create Level 2 Category (Restaurant)
	l2ID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Restaurant",
		"type": "expense",
		"parent_id": "`+l1ID+`"
	}`)

	// Create Level 3 Category (FastFood)
	l3ID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "FastFood",
		"type": "expense",
		"parent_id": "`+l2ID+`"
	}`)

	// Attempt to create Level 4 Category (Should Fail)
	resL4 := performAPIRequest(t, router, token, http.MethodPost, "/api/v1/categories", `{
		"name": "Burger",
		"type": "expense",
		"parent_id": "`+l3ID+`"
	}`, http.StatusBadRequest)

	// Ensure depth error
	if string(resL4) == "" {
		t.Fatalf("expected depth exceeded error")
	}

	// Test Cyclic Reference
	// Try updating Level 1 to have Level 3 as parent
	resCycle := performAPIRequest(t, router, token, http.MethodPut, "/api/v1/categories/"+l1ID, `{
		"name": "Food (Cycle)",
		"type": "expense",
		"parent_id": "`+l3ID+`"
	}`, http.StatusBadRequest)

	if string(resCycle) == "" {
		t.Fatalf("expected cyclic reference error")
	}

	// Test GET Categories (Verify parent_id is returned)
	resList := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/categories", "", http.StatusOK)
	var listResp struct {
		Categories []struct {
			ID       string  `json:"id"`
			Name     string  `json:"name"`
			ParentID *string `json:"parent_id"`
		} `json:"categories"`
	}
	json.Unmarshal(resList, &listResp)

	if len(listResp.Categories) != 3 {
		t.Fatalf("Expected 3 categories, got %d", len(listResp.Categories))
	}

	// Delete Level 1 category should fail (RESTRICT)
	performAPIRequest(t, router, token, http.MethodDelete, "/api/v1/categories/"+l1ID, "", http.StatusBadRequest)
}

func TestCategoryHierarchyRejectsCrossUserAndTypeMismatchParents(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "cat-parent-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userA, tokenA := registerIntegrationAPIUser(t, router, "hierarchy-parent-a")
	userB, tokenB := registerIntegrationAPIUser(t, router, "hierarchy-parent-b")
	defer cleanupServerIntegrationUsers(t, pool, userA, userB)

	otherUserParent := createAPIResource(t, router, tokenB, "/api/v1/categories", `{
		"name": "Other User Parent",
		"type": "expense"
	}`)
	incomeParent := createAPIResource(t, router, tokenA, "/api/v1/categories", `{
		"name": "Income Parent",
		"type": "income"
	}`)
	expenseChild := createAPIResource(t, router, tokenA, "/api/v1/categories", `{
		"name": "Expense Child",
		"type": "expense"
	}`)

	assertAPIStatus(t, router, tokenA, http.MethodPost, "/api/v1/categories", `{
		"name": "Cross User Child",
		"type": "expense",
		"parent_id": "`+otherUserParent+`"
	}`, http.StatusBadRequest)

	assertAPIStatus(t, router, tokenA, http.MethodPost, "/api/v1/categories", `{
		"name": "Mixed Type Child",
		"type": "expense",
		"parent_id": "`+incomeParent+`"
	}`, http.StatusBadRequest)

	assertAPIStatus(t, router, tokenA, http.MethodPut, "/api/v1/categories/"+expenseChild, `{
		"name": "Expense Child Updated",
		"type": "expense",
		"parent_id": "`+incomeParent+`"
	}`, http.StatusBadRequest)

	resList := performAPIRequest(t, router, tokenA, http.MethodGet, "/api/v1/categories", "", http.StatusOK)
	var listResp struct {
		Categories []struct {
			ID       string  `json:"id"`
			ParentID *string `json:"parent_id"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(resList, &listResp); err != nil {
		t.Fatalf("parse categories response: %v", err)
	}
	if len(listResp.Categories) != 2 {
		t.Fatalf("expected only valid user A categories, got %+v", listResp.Categories)
	}
	for _, category := range listResp.Categories {
		if category.ParentID != nil {
			t.Fatalf("expected invalid parent writes to be rejected, got %+v", category)
		}
	}
}
