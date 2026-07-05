package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// onboardingDefaultCategoryCount is the number of default categories POST
// /auth/register seeds for every new account (see internal/auth/onboarding.go).
const onboardingDefaultCategoryCount = 8

// removeOnboardingDefaults deletes a test user's register-seeded default
// categories and starter wallet, for tests that assert exact category
// positions/counts or wallet counts and want a clean slate. The onboarding
// defaults themselves are covered by TestRegisterCreatesOnboardingDefaults.
func removeOnboardingDefaults(t *testing.T, pool *pgxpool.Pool, userIDs ...string) {
	t.Helper()
	for _, userID := range userIDs {
		if _, err := pool.Exec(context.Background(), `DELETE FROM categories WHERE user_id = $1`, userID); err != nil {
			t.Fatalf("remove onboarding default categories: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM wallets WHERE user_id = $1 AND name = 'Dompet Utama'`, userID); err != nil {
			t.Fatalf("remove onboarding starter wallet: %v", err)
		}
	}
}

// TestRegisterCreatesOnboardingDefaults proves POST /auth/register leaves a
// fresh account fully usable: 8 default Bahasa Indonesia categories (position
// ASC, icon+color from the shared client catalogs) and one starter cash
// wallet, while the register response shape stays `{ user, tokens }` (checked
// by registerIntegrationAPIUser itself).
func TestRegisterCreatesOnboardingDefaults(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "register-onboarding-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "register-onboarding")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	// Categories: exactly the 8 defaults, in position order (the list default
	// sort is position ASC).
	response := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/categories", "", http.StatusOK)
	var categoriesParsed struct {
		Categories []struct {
			Name     string `json:"name"`
			Type     string `json:"type"`
			Icon     string `json:"icon"`
			Color    string `json:"color"`
			Position int    `json:"position"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(response, &categoriesParsed); err != nil {
		t.Fatalf("parse categories response: %v", err)
	}

	want := []struct {
		name, categoryType, icon, color string
	}{
		{"Makanan & Minuman", "expense", "food", "#C2553F"},
		{"Transportasi", "expense", "transport", "#E0A23B"},
		{"Belanja", "expense", "shopping", "#C2588A"},
		{"Tagihan & Utilitas", "expense", "bills", "#4256B8"},
		{"Hiburan", "expense", "entertainment", "#7C5BC2"},
		{"Kesehatan", "expense", "health", "#2BB3A3"},
		{"Gaji", "income", "salary", "#2E8B57"},
		{"Penghasilan Lain", "income", "misc", "#9E7B4F"},
	}
	if len(categoriesParsed.Categories) != len(want) {
		t.Fatalf("expected %d default categories, got %d: %+v", len(want), len(categoriesParsed.Categories), categoriesParsed.Categories)
	}
	for i, got := range categoriesParsed.Categories {
		if got.Name != want[i].name || got.Type != want[i].categoryType || got.Icon != want[i].icon || got.Color != want[i].color {
			t.Fatalf("default category %d mismatch: got %+v, want %+v", i, got, want[i])
		}
		if got.Position != i {
			t.Fatalf("default category %q expected position %d, got %d", got.Name, i, got.Position)
		}
	}

	// Wallet: exactly the starter "Dompet Utama".
	response = performAPIRequest(t, router, token, http.MethodGet, "/api/v1/wallets", "", http.StatusOK)
	var walletsParsed struct {
		Wallets []struct {
			Name         string `json:"name"`
			Type         string `json:"type"`
			CurrencyCode string `json:"currency_code"`
			BalanceMinor int64  `json:"balance_minor"`
		} `json:"wallets"`
	}
	if err := json.Unmarshal(response, &walletsParsed); err != nil {
		t.Fatalf("parse wallets response: %v", err)
	}
	if len(walletsParsed.Wallets) != 1 {
		t.Fatalf("expected exactly the starter wallet, got %+v", walletsParsed.Wallets)
	}
	starter := walletsParsed.Wallets[0]
	if starter.Name != "Dompet Utama" || starter.Type != "cash" || starter.CurrencyCode != "IDR" || starter.BalanceMinor != 0 {
		t.Fatalf("unexpected starter wallet: %+v", starter)
	}
}

// TestRegisterDuplicateEmailCreatesNothing proves a failed register writes no
// rows at all: re-registering an existing email conflicts, and the original
// account keeps exactly its own defaults (no stray categories/wallets appear
// anywhere).
func TestRegisterDuplicateEmailCreatesNothing(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "register-duplicate-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, _ := registerIntegrationAPIUser(t, router, "register-duplicate")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	var email string
	if err := pool.QueryRow(t.Context(), `SELECT email FROM users WHERE id = $1`, userID).Scan(&email); err != nil {
		t.Fatalf("read registered email: %v", err)
	}

	categoriesBefore := countServerIntegrationRows(t, pool, `SELECT COUNT(*) FROM categories`)
	walletsBefore := countServerIntegrationRows(t, pool, `SELECT COUNT(*) FROM wallets`)

	body := `{"email":"` + email + `","password":"super-secret-password"}`
	assertAPIStatus(t, router, "", http.MethodPost, "/api/v1/auth/register", body, http.StatusConflict)

	if got := countServerIntegrationRows(t, pool, `SELECT COUNT(*) FROM categories`); got != categoriesBefore {
		t.Fatalf("failed register changed categories count: %d -> %d", categoriesBefore, got)
	}
	if got := countServerIntegrationRows(t, pool, `SELECT COUNT(*) FROM wallets`); got != walletsBefore {
		t.Fatalf("failed register changed wallets count: %d -> %d", walletsBefore, got)
	}
	var usersForEmail int
	if err := pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM users WHERE email = $1`, email).Scan(&usersForEmail); err != nil {
		t.Fatalf("count users for email: %v", err)
	}
	if usersForEmail != 1 {
		t.Fatalf("expected exactly one user for %s, got %d", email, usersForEmail)
	}
}

func countServerIntegrationRows(t *testing.T, pool *pgxpool.Pool, query string) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(t.Context(), query).Scan(&count); err != nil {
		t.Fatalf("count query %q: %v", query, err)
	}
	return count
}
