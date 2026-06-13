package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
)

func TestListCategoriesCanFilterByType(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "integration-test-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "api-category-filter")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	incomeID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Filter Income",
		"type": "income"
	}`)
	expenseID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Filter Expense",
		"type": "expense"
	}`)

	allCategories := listCategoryTypesByID(t, performAPIRequest(t, router, token, http.MethodGet, "/api/v1/categories", "", http.StatusOK))
	if allCategories[incomeID] != "income" || allCategories[expenseID] != "expense" {
		t.Fatalf("expected unfiltered list to include both categories, got %+v", allCategories)
	}

	incomeCategories := listCategoryTypesByID(t, performAPIRequest(t, router, token, http.MethodGet, "/api/v1/categories?type=income", "", http.StatusOK))
	if incomeCategories[incomeID] != "income" {
		t.Fatalf("expected income category %s in filtered response, got %+v", incomeID, incomeCategories)
	}
	if _, ok := incomeCategories[expenseID]; ok {
		t.Fatalf("did not expect expense category %s in income response: %+v", expenseID, incomeCategories)
	}

	expenseCategories := listCategoryTypesByID(t, performAPIRequest(t, router, token, http.MethodGet, "/api/v1/categories?type=expense", "", http.StatusOK))
	if expenseCategories[expenseID] != "expense" {
		t.Fatalf("expected expense category %s in filtered response, got %+v", expenseID, expenseCategories)
	}
	if _, ok := expenseCategories[incomeID]; ok {
		t.Fatalf("did not expect income category %s in expense response: %+v", incomeID, expenseCategories)
	}

	assertAPIStatus(t, router, token, http.MethodGet, "/api/v1/categories?type=transfer", "", http.StatusBadRequest)
}

func listCategoryTypesByID(t *testing.T, response []byte) map[string]string {
	t.Helper()

	var parsed struct {
		Categories []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(response, &parsed); err != nil {
		t.Fatalf("parse category list response: %v", err)
	}

	typesByID := make(map[string]string, len(parsed.Categories))
	for _, category := range parsed.Categories {
		typesByID[category.ID] = category.Type
	}
	return typesByID
}
