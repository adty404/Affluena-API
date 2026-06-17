package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
)

func TestWalletColorDescriptionRoundTrip(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "wallet-color-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "wallet-color")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Color Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 0,
		"color": "purple",
		"description": "Test description"
	}`)

	body := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/wallets/"+walletID, "", http.StatusOK)
	var wallet struct {
		Color       string `json:"color"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(body), &wallet); err != nil {
		t.Fatalf("unmarshal wallet: %v", err)
	}
	if wallet.Color != "purple" {
		t.Fatalf("expected color purple, got %q", wallet.Color)
	}
	if wallet.Description != "Test description" {
		t.Fatalf("expected description, got %q", wallet.Description)
	}

	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/wallets/"+walletID, `{
		"name": "Color Wallet Updated",
		"type": "bank",
		"currency_code": "IDR",
		"color": "orange",
		"description": "Updated"
	}`, http.StatusOK)

	body = performAPIRequest(t, router, token, http.MethodGet, "/api/v1/wallets/"+walletID, "", http.StatusOK)
	if err := json.Unmarshal([]byte(body), &wallet); err != nil {
		t.Fatalf("unmarshal after update: %v", err)
	}
	if wallet.Color != "orange" {
		t.Fatalf("expected updated color orange, got %q", wallet.Color)
	}
	if wallet.Description != "Updated" {
		t.Fatalf("expected updated description, got %q", wallet.Description)
	}
}

func TestWalletMembersListedIncludesOwner(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "wallet-members-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "wallet-members-owner")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Members Test Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 0
	}`)

	body := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/wallets/"+walletID+"/members", "", http.StatusOK)
	var resp struct {
		Members []struct {
			UserID string `json:"user_id"`
			Email  string `json:"email"`
			Role   string `json:"role"`
			Status string `json:"status"`
		} `json:"members"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal members: %v", err)
	}
	if len(resp.Members) != 1 {
		t.Fatalf("expected 1 member (owner), got %d", len(resp.Members))
	}
	if resp.Members[0].Role != "owner" {
		t.Fatalf("expected role owner, got %q", resp.Members[0].Role)
	}
	if resp.Members[0].Status != "joined" {
		t.Fatalf("expected status joined, got %q", resp.Members[0].Status)
	}
	if resp.Members[0].UserID != userID {
		t.Fatalf("expected user_id %s, got %s", userID, resp.Members[0].UserID)
	}
}

func TestWalletMembersInaccessibleReturnsNotFound(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "wallet-members-404-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	ownerID, ownerToken := registerIntegrationAPIUser(t, router, "wallet-members-owner-2")
	otherID, otherToken := registerIntegrationAPIUser(t, router, "wallet-members-other")
	defer cleanupServerIntegrationUsers(t, pool, ownerID, otherID)

	walletID := createAPIResource(t, router, ownerToken, "/api/v1/wallets", `{
		"name": "Private Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 0
	}`)

	assertAPIStatus(t, router, otherToken, http.MethodGet, "/api/v1/wallets/"+walletID+"/members", "", http.StatusNotFound)
}

func TestWalletAnalyticsShape(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "wallet-analytics-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "wallet-analytics")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Analytics Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 0
	}`)

	body := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/wallets/"+walletID+"/analytics?month=2026-06", "", http.StatusOK)
	var analytics struct {
		WalletID         string  `json:"wallet_id"`
		Month            string  `json:"month"`
		InflowMinor      int64   `json:"inflow_minor"`
		OutflowMinor     int64   `json:"outflow_minor"`
		TransactionCount int64   `json:"transaction_count"`
		LastActivityAt   *string `json:"last_activity_at"`
	}
	if err := json.Unmarshal([]byte(body), &analytics); err != nil {
		t.Fatalf("unmarshal analytics: %v", err)
	}
	if analytics.WalletID != walletID {
		t.Fatalf("expected wallet_id %s, got %s", walletID, analytics.WalletID)
	}
	if analytics.Month != "2026-06" {
		t.Fatalf("expected month 2026-06, got %s", analytics.Month)
	}
	if analytics.InflowMinor != 0 || analytics.OutflowMinor != 0 {
		t.Fatalf("expected zero inflow/outflow for new wallet, got in=%d out=%d", analytics.InflowMinor, analytics.OutflowMinor)
	}
}

func TestWalletAnalyticsInvalidMonth(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "wallet-analytics-invalid-month",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "wallet-analytics-invalid")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Invalid Month Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 0
	}`)

	assertAPIStatus(t, router, token, http.MethodGet, "/api/v1/wallets/"+walletID+"/analytics?month=invalid", "", http.StatusBadRequest)
	assertAPIStatus(t, router, token, http.MethodGet, "/api/v1/wallets/"+walletID+"/analytics?month=2026-13", "", http.StatusOK)
}

func TestWalletAnalyticsCrossUserNotFound(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "wallet-analytics-xuser",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	ownerID, ownerToken := registerIntegrationAPIUser(t, router, "wallet-analytics-owner")
	otherID, otherToken := registerIntegrationAPIUser(t, router, "wallet-analytics-other")
	defer cleanupServerIntegrationUsers(t, pool, ownerID, otherID)

	walletID := createAPIResource(t, router, ownerToken, "/api/v1/wallets", `{
		"name": "Private Analytics Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 0
	}`)

	assertAPIStatus(t, router, otherToken, http.MethodGet, "/api/v1/wallets/"+walletID+"/analytics?month=2026-06", "", http.StatusNotFound)
}
