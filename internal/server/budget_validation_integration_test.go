package server

import (
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
)

func TestCategoryBudgetListRejectsInvalidMonth(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "budget-validation-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "budget-validation")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	assertAPIStatus(t, router, token, http.MethodGet, "/api/v1/category-budgets?month=2026/06", "", http.StatusBadRequest)
}
