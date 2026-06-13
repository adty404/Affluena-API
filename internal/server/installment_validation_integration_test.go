package server

import (
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
)

func TestInstallmentRejectsPlanThatDoesNotMatchTotal(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "installment-validation-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "installment-validation")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Installment validation wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 1000000
	}`)
	categoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Installment validation expense",
		"type": "expense"
	}`)

	assertAPIStatus(t, router, token, http.MethodPost, "/api/v1/installments", `{
		"name": "Inconsistent laptop",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"total_amount_minor": 100000,
		"monthly_amount_minor": 60000,
		"tenor_months": 2,
		"start_date": "2026-06-01",
		"due_day": 5
	}`, http.StatusBadRequest)
}
