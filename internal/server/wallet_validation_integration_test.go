package server

import (
	"encoding/json"
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

func TestWalletRejectsDirectMutationOfManagedGoalWallet(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "wallet-goal-mutation-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "wallet-goal-mutation")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	goalID := createAPIResource(t, router, token, "/api/v1/goals", `{
		"name": "Emergency Fund",
		"target_amount_minor": 5000000,
		"deadline": "2026-12-31T00:00:00Z"
	}`)
	goalWalletID := findGoalWallet(t, router, token, goalID)
	if goalWalletID == "" {
		t.Fatalf("expected financial goal workflow to create a goal wallet")
	}

	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/wallets/"+goalWalletID, `{
		"name": "Converted goal wallet",
		"type": "bank",
		"currency_code": "IDR"
	}`, http.StatusBadRequest)
	assertAPIStatus(t, router, token, http.MethodDelete, "/api/v1/wallets/"+goalWalletID, "", http.StatusBadRequest)

	if got := findGoalWallet(t, router, token, goalID); got != goalWalletID {
		t.Fatalf("expected managed goal wallet %q to remain intact, got %q", goalWalletID, got)
	}
}

func TestWalletColorAndIconRoundTrip(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "wallet-color-icon-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "wallet-color-icon")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Dompet warna",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 0,
		"color": "#3E72B8",
		"icon": "bank"
	}`)

	var created struct {
		Color string `json:"color"`
		Icon  string `json:"icon"`
	}
	body := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/wallets/"+walletID, "", http.StatusOK)
	if err := json.Unmarshal([]byte(body), &created); err != nil {
		t.Fatalf("decode wallet: %v", err)
	}
	if created.Color != "#3E72B8" || created.Icon != "bank" {
		t.Fatalf("color/icon did not round-trip on create: got color=%q icon=%q", created.Color, created.Icon)
	}

	// Update should persist new color + icon.
	performAPIRequest(t, router, token, http.MethodPut, "/api/v1/wallets/"+walletID, `{
		"name": "Dompet warna",
		"type": "bank",
		"currency_code": "IDR",
		"color": "#2E8B57",
		"icon": "savings"
	}`, http.StatusOK)

	var updated struct {
		Color string `json:"color"`
		Icon  string `json:"icon"`
	}
	body = performAPIRequest(t, router, token, http.MethodGet, "/api/v1/wallets/"+walletID, "", http.StatusOK)
	if err := json.Unmarshal([]byte(body), &updated); err != nil {
		t.Fatalf("decode updated wallet: %v", err)
	}
	if updated.Color != "#2E8B57" || updated.Icon != "savings" {
		t.Fatalf("color/icon did not round-trip on update: got color=%q icon=%q", updated.Color, updated.Icon)
	}
}
