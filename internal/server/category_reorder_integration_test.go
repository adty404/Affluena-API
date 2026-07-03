package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
)

// TestCategoryAppearanceAndReorder proves category icon/color round-trip,
// the default position-based list ordering, the reorder endpoint (including
// partial reorders), and that reorder cannot touch another user's categories.
func TestCategoryAppearanceAndReorder(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "integration-test-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "api-category-reorder-a")
	otherID, otherToken := registerIntegrationAPIUser(t, router, "api-category-reorder-b")
	defer cleanupServerIntegrationUsers(t, pool, userID, otherID)

	groceriesID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Groceries",
		"type": "expense",
		"icon": "cart",
		"color": "#FF8800"
	}`)
	transportID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Transport",
		"type": "expense"
	}`)
	utilitiesID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Utilities",
		"type": "expense",
		"icon": "bolt",
		"color": "#0044FF"
	}`)
	otherCategoryID := createAPIResource(t, router, otherToken, "/api/v1/categories", `{
		"name": "Other User Category",
		"type": "expense"
	}`)

	// Icon/color round-trip on GET; position defaults to a per-user sequence.
	groceries := getReorderTestCategory(t, router, token, groceriesID)
	if groceries.Icon != "cart" || groceries.Color != "#FF8800" || groceries.Position != 0 {
		t.Fatalf("unexpected first category appearance/position: %+v", groceries)
	}
	utilities := getReorderTestCategory(t, router, token, utilitiesID)
	if utilities.Icon != "bolt" || utilities.Color != "#0044FF" || utilities.Position != 2 {
		t.Fatalf("unexpected third category appearance/position: %+v", utilities)
	}
	if transport := getReorderTestCategory(t, router, token, transportID); transport.Icon != "" || transport.Color != "" || transport.Position != 1 {
		t.Fatalf("expected empty icon/color and position 1 for second category, got %+v", transport)
	}

	// Default list order is position ascending (creation order for now).
	assertCategoryOrder(t, router, token, "/api/v1/categories", []string{groceriesID, transportID, utilitiesID})

	// Full reorder sets position = array index inside one transaction.
	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/categories/reorder", `{
		"ids": ["`+utilitiesID+`", "`+groceriesID+`", "`+transportID+`"]
	}`, http.StatusNoContent)
	assertCategoryOrder(t, router, token, "/api/v1/categories", []string{utilitiesID, groceriesID, transportID})
	if got := getReorderTestCategory(t, router, token, utilitiesID); got.Position != 0 {
		t.Fatalf("expected reordered category position 0, got %+v", got)
	}
	if got := getReorderTestCategory(t, router, token, transportID); got.Position != 2 {
		t.Fatalf("expected reordered category position 2, got %+v", got)
	}

	// Partial reorder: omitted ids keep their position; ties break by name.
	// transport -> 0, utilities -> 1, groceries keeps 1 ("Groceries" < "Utilities").
	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/categories/reorder", `{
		"ids": ["`+transportID+`", "`+utilitiesID+`"]
	}`, http.StatusNoContent)
	assertCategoryOrder(t, router, token, "/api/v1/categories", []string{transportID, groceriesID, utilitiesID})
	if got := getReorderTestCategory(t, router, token, groceriesID); got.Position != 1 {
		t.Fatalf("expected omitted category to keep position 1, got %+v", got)
	}

	// Explicit sort keys still work.
	assertCategoryOrder(t, router, token, "/api/v1/categories?sort=name_asc", []string{groceriesID, transportID, utilitiesID})

	// Update replaces icon/color and preserves position.
	updated := performAPIRequest(t, router, token, http.MethodPut, "/api/v1/categories/"+groceriesID, `{
		"name": "Groceries",
		"type": "expense",
		"icon": "basket",
		"color": "#00CC66"
	}`, http.StatusOK)
	var updatedCategory reorderTestCategory
	if err := json.Unmarshal(updated, &updatedCategory); err != nil {
		t.Fatalf("parse update response: %v", err)
	}
	if updatedCategory.Icon != "basket" || updatedCategory.Color != "#00CC66" || updatedCategory.Position != 1 {
		t.Fatalf("expected update to replace icon/color and keep position, got %+v", updatedCategory)
	}

	// Red flows: another user's id (alone or mixed in) is 404 and rolls the
	// whole reorder back; malformed payloads are 400.
	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/categories/reorder", `{
		"ids": ["`+otherCategoryID+`"]
	}`, http.StatusNotFound)
	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/categories/reorder", `{
		"ids": ["`+utilitiesID+`", "`+otherCategoryID+`", "`+groceriesID+`"]
	}`, http.StatusNotFound)
	assertCategoryOrder(t, router, token, "/api/v1/categories", []string{transportID, groceriesID, utilitiesID})
	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/categories/reorder", `{
		"ids": ["00000000-0000-0000-0000-000000000001"]
	}`, http.StatusNotFound)
	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/categories/reorder", `{
		"ids": ["`+groceriesID+`", "`+groceriesID+`"]
	}`, http.StatusBadRequest)
	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/categories/reorder", `{
		"ids": ["not-a-uuid"]
	}`, http.StatusBadRequest)
	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/categories/reorder", `{
		"ids": []
	}`, http.StatusBadRequest)

	// The other user's own reorder works and never touches this user's rows.
	assertAPIStatus(t, router, otherToken, http.MethodPut, "/api/v1/categories/reorder", `{
		"ids": ["`+otherCategoryID+`"]
	}`, http.StatusNoContent)
	assertCategoryOrder(t, router, token, "/api/v1/categories", []string{transportID, groceriesID, utilitiesID})
}

type reorderTestCategory struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	Color    string `json:"color"`
	Position int    `json:"position"`
}

func getReorderTestCategory(t *testing.T, router http.Handler, token string, id string) reorderTestCategory {
	t.Helper()

	response := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/categories/"+id, "", http.StatusOK)
	var category reorderTestCategory
	if err := json.Unmarshal(response, &category); err != nil {
		t.Fatalf("parse category response: %v", err)
	}
	return category
}

func assertCategoryOrder(t *testing.T, router http.Handler, token string, path string, wantIDs []string) {
	t.Helper()

	response := performAPIRequest(t, router, token, http.MethodGet, path, "", http.StatusOK)
	var parsed struct {
		Categories []reorderTestCategory `json:"categories"`
	}
	if err := json.Unmarshal(response, &parsed); err != nil {
		t.Fatalf("parse category list response: %v", err)
	}
	if len(parsed.Categories) != len(wantIDs) {
		t.Fatalf("expected %d categories, got %d: %s", len(wantIDs), len(parsed.Categories), string(response))
	}
	for i, want := range wantIDs {
		if parsed.Categories[i].ID != want {
			t.Fatalf("expected category %s at index %d, got %+v", want, i, parsed.Categories)
		}
	}
}
