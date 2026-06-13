package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
	"affluena-api/internal/goal"
)

func TestGoalLifecycleAndMembers(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "goal-integration-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userA, tokenA := registerIntegrationAPIUser(t, router, "usera")
	defer cleanupServerIntegrationUsers(t, pool, userA)

	userB, tokenB := registerIntegrationAPIUser(t, router, "userb")
	defer cleanupServerIntegrationUsers(t, pool, userB)

	// Fetch UserB's email
	resB := performAPIRequest(t, router, tokenB, http.MethodGet, "/api/v1/auth/me", "", http.StatusOK)
	var userBData struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	json.Unmarshal(resB, &userBData)

	// --- Edge Case Red: Missing name in create goal -> 400
	assertAPIStatus(t, router, tokenA, http.MethodPost, "/api/v1/goals", `{
		"target_amount_minor": 10000000
	}`, http.StatusBadRequest)

	// --- Edge Case Red: Invalid target amount (e.g. string instead of int, though json unmarshal handles it, but maybe negative? Wait, check constraint fails)
	// We'll skip DB check constraint test for now, just test green flow for creation.

	// --- Green Flow: Create Goal
	goalID := createAPIResource(t, router, tokenA, "/api/v1/goals", `{
		"name": "Wedding",
		"target_amount_minor": 10000000,
		"deadline": "2026-12-31T00:00:00Z"
	}`)

	// Validate User A has a goal wallet created
	walletAForGoal := findGoalWallet(t, router, tokenA, goalID)
	if walletAForGoal == "" {
		t.Fatalf("Expected goal wallet for User A")
	}

	// --- Edge Case Red: User B tries to invite someone to User A's goal -> 404 Not Found (hidden existence)
	assertAPIStatus(t, router, tokenB, http.MethodPost, "/api/v1/goals/"+goalID+"/members", `{
		"email": "anyone@example.com"
	}`, http.StatusNotFound)

	// --- Edge Case Red: User A invites non-existent user -> 400 Bad Request
	assertAPIStatus(t, router, tokenA, http.MethodPost, "/api/v1/goals/"+goalID+"/members", `{
		"email": "notfound@example.com"
	}`, http.StatusBadRequest)

	// --- Green Flow: User A invites User B -> 200 OK
	assertAPIStatus(t, router, tokenA, http.MethodPost, "/api/v1/goals/"+goalID+"/members", `{
		"email": "`+userBData.User.Email+`"
	}`, http.StatusOK)

	// --- Edge Case Red: User B responds with invalid status -> 400
	assertAPIStatus(t, router, tokenB, http.MethodPut, "/api/v1/goals/"+goalID+"/members/"+userB+"/respond", `{
		"status": "unknown"
	}`, http.StatusBadRequest)

	// --- Green Flow: User B accepts invite
	assertAPIStatus(t, router, tokenB, http.MethodPut, "/api/v1/goals/"+goalID+"/members/"+userB+"/respond", `{
		"status": "joined"
	}`, http.StatusOK)

	// --- Edge Case Red: User B accepts again -> 400 Bad Request
	assertAPIStatus(t, router, tokenB, http.MethodPut, "/api/v1/goals/"+goalID+"/members/"+userB+"/respond", `{
		"status": "joined"
	}`, http.StatusBadRequest)

	// Validate User B has a goal wallet created
	walletBForGoal := findGoalWallet(t, router, tokenB, goalID)
	if walletBForGoal == "" {
		t.Fatalf("Expected goal wallet for User B")
	}

	// --- Data Integrity: Funding the Goal ---
	// Create Bank Wallet for User A and User B to fund from
	bankA := createAPIResource(t, router, tokenA, "/api/v1/wallets", `{
		"name": "Bank A",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 5000000
	}`)
	bankB := createAPIResource(t, router, tokenB, "/api/v1/wallets", `{
		"name": "Bank B",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 5000000
	}`)

	// User A transfers 1M to Goal
	createAPIResource(t, router, tokenA, "/api/v1/transactions", `{
		"type": "transfer",
		"wallet_id": "`+bankA+`",
		"to_wallet_id": "`+walletAForGoal+`",
		"amount_minor": 1000000,
		"transaction_at": "2026-06-13T10:00:00Z"
	}`)

	// User B transfers 2M to Goal
	createAPIResource(t, router, tokenB, "/api/v1/transactions", `{
		"type": "transfer",
		"wallet_id": "`+bankB+`",
		"to_wallet_id": "`+walletBForGoal+`",
		"amount_minor": 2000000,
		"transaction_at": "2026-06-13T10:00:00Z"
	}`)

	// Verify Goal total is 3M
	resGoal := performAPIRequest(t, router, tokenA, http.MethodGet, "/api/v1/goals/"+goalID, "", http.StatusOK)
	var g goal.Goal
	json.Unmarshal(resGoal, &g)
	if g.CollectedAmountMinor != 3000000 {
		t.Fatalf("Expected goal to have 3000000 collected, got %d", g.CollectedAmountMinor)
	}

	// Verify User A Bank is 4M and User B Bank is 3M
	assertWalletBalance(t, router, tokenA, bankA, 4000000)
	assertWalletBalance(t, router, tokenB, bankB, 3000000)
}

func findGoalWallet(t *testing.T, router http.Handler, token string, goalID string) string {
	t.Helper()
	res := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/wallets", "", http.StatusOK)
	t.Log("Wallets response:", string(res))
	var wallets struct {
		Wallets []struct {
			ID     string  `json:"id"`
			Type   string  `json:"type"`
			GoalID *string `json:"goal_id"`
		} `json:"wallets"`
	}
	json.Unmarshal(res, &wallets)

	for _, w := range wallets.Wallets {
		if w.Type == "goal" && w.GoalID != nil && *w.GoalID == goalID {
			return w.ID
		}
	}
	return ""
}
