package server

import (
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
)

func TestWalletRejectsPublicGoalWalletWrites(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "wallet-validation-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "wallet-validation")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	assertAPIStatus(t, router, token, http.MethodPost, "/api/v1/wallets", `{
		"name": "Standalone goal wallet",
		"type": "goal",
		"currency_code": "IDR",
		"balance_minor": 0
	}`, http.StatusBadRequest)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Regular wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 0
	}`)

	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/wallets/"+walletID, `{
		"name": "Promoted goal wallet",
		"type": "goal",
		"currency_code": "IDR"
	}`, http.StatusBadRequest)
}
