package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
	"affluena-api/internal/wallet"
)

func TestWalletShareLifecycle(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "wallet-share-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userA, tokenA := registerIntegrationAPIUser(t, router, "usera-wallet-share")
	defer cleanupServerIntegrationUsers(t, pool, userA)

	userB, tokenB := registerIntegrationAPIUser(t, router, "userb-wallet-share")
	defer cleanupServerIntegrationUsers(t, pool, userB)

	userC, tokenC := registerIntegrationAPIUser(t, router, "userc-wallet-share")
	defer cleanupServerIntegrationUsers(t, pool, userC)

	// Fetch User B's email
	resB := performAPIRequest(t, router, tokenB, http.MethodGet, "/api/v1/auth/me", "", http.StatusOK)
	var userBData struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	json.Unmarshal(resB, &userBData)

	// User A creates a Wallet
	walletID := createAPIResource(t, router, tokenA, "/api/v1/wallets", `{
		"name": "Family Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 0
	}`)

	// A invites B
	assertAPIStatus(t, router, tokenA, http.MethodPost, "/api/v1/wallets/"+walletID+"/invites", `{
		"email": "`+userBData.User.Email+`"
	}`, http.StatusCreated)

	// User B checks their wallets, should see it as pending
	resWalletsB := performAPIRequest(t, router, tokenB, http.MethodGet, "/api/v1/wallets", "", http.StatusOK)
	var listResp struct {
		Wallets []wallet.Wallet `json:"wallets"`
	}
	json.Unmarshal(resWalletsB, &listResp)
	if len(listResp.Wallets) != 1 || listResp.Wallets[0].ShareStatus != "pending" {
		t.Fatalf("expected 1 pending wallet, got %v", listResp.Wallets)
	}

	// B responds with joined
	assertAPIStatus(t, router, tokenB, http.MethodPatch, "/api/v1/wallets/"+walletID+"/members/"+userB, `{
		"status": "joined"
	}`, http.StatusOK)

	// Now B should see share_status = joined
	resWalletsB2 := performAPIRequest(t, router, tokenB, http.MethodGet, "/api/v1/wallets", "", http.StatusOK)
	json.Unmarshal(resWalletsB2, &listResp)
	if len(listResp.Wallets) != 1 || listResp.Wallets[0].ShareStatus != "joined" {
		t.Fatalf("expected joined wallet, got %v", listResp.Wallets)
	}

	// C tries to invite themselves (fails, C is not owner and cannot see wallet)
	assertAPIStatus(t, router, tokenC, http.MethodPost, "/api/v1/wallets/"+walletID+"/invites", `{
		"email": "doesn-matter@example.com"
	}`, http.StatusNotFound)

	// B logs an expense on the shared wallet
	catID := createAPIResource(t, router, tokenB, "/api/v1/categories", `{
		"name": "Groceries (User B)",
		"type": "expense"
	}`)

	// Insert transaction by B
	createAPIResource(t, router, tokenB, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+catID+`",
		"amount_minor": 50000,
		"transaction_at": "`+time.Now().Format(time.RFC3339)+`"
	}`)

	// User A views dashboard, should see Expense of 50000
	resDash := performAPIRequest(t, router, tokenA, http.MethodGet, "/api/v1/dashboard/summary?month="+time.Now().Format("2006-01"), "", http.StatusOK)
	var summary struct {
		MonthlyExpenseMinor int64 `json:"monthly_expense_minor"`
		NetWorthMinor       int64 `json:"net_worth_minor"`
	}
	json.Unmarshal(resDash, &summary)
	if summary.MonthlyExpenseMinor != 50000 {
		t.Fatalf("expected monthly expense 50000 on owner's dashboard, got %d", summary.MonthlyExpenseMinor)
	}
	if summary.NetWorthMinor != -50000 {
		t.Fatalf("expected net worth -50000, got %d", summary.NetWorthMinor)
	}
}
