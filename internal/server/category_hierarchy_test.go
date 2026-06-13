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
