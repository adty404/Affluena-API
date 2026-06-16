package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"affluena-api/internal/config"
	"affluena-api/internal/goal"
	"affluena-api/internal/wallet"
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

	// --- Edge Case Red: User A invites non-existent user -> 404 Not Found
	assertAPIStatus(t, router, tokenA, http.MethodPost, "/api/v1/goals/"+goalID+"/members", `{
		"email": "notfound@example.com"
	}`, http.StatusNotFound)

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

func TestGoalDuplicateNamesAndInviteEdgeCases(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "goal-edge-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userA, tokenA := registerIntegrationAPIUser(t, router, "goal-edge-a")
	userB, tokenB := registerIntegrationAPIUser(t, router, "goal-edge-b")
	defer cleanupServerIntegrationUsers(t, pool, userA, userB)

	resB := performAPIRequest(t, router, tokenB, http.MethodGet, "/api/v1/auth/me", "", http.StatusOK)
	var userBData struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(resB, &userBData); err != nil {
		t.Fatalf("parse user B response: %v", err)
	}

	goalA := createAPIResource(t, router, tokenA, "/api/v1/goals", `{
		"name": "Wedding",
		"target_amount_minor": 10000000,
		"deadline": "2026-12-31T00:00:00Z"
	}`)
	goalB := createAPIResource(t, router, tokenA, "/api/v1/goals", `{
		"name": "Wedding",
		"target_amount_minor": 20000000,
		"deadline": "2027-12-31T00:00:00Z"
	}`)

	walletA := findGoalWallet(t, router, tokenA, goalA)
	walletB := findGoalWallet(t, router, tokenA, goalB)
	if walletA == "" || walletB == "" || walletA == walletB {
		t.Fatalf("expected duplicate goal names to create distinct goal wallets, got %q and %q", walletA, walletB)
	}

	assertAPIStatus(t, router, tokenA, http.MethodPost, "/api/v1/goals/"+goalA+"/members", `{
		"email": "`+userBData.User.Email+`"
	}`, http.StatusOK)

	assertAPIStatus(t, router, tokenB, http.MethodPut, "/api/v1/goals/"+goalA+"/members/"+userA+"/respond", `{
		"status": "joined"
	}`, http.StatusNotFound)
	if got := findGoalWallet(t, router, tokenB, goalA); got != "" {
		t.Fatalf("expected mismatched member response to avoid creating goal wallet, got %q", got)
	}

	assertAPIStatus(t, router, tokenB, http.MethodPut, "/api/v1/goals/"+goalA+"/members/"+userB+"/respond", `{
		"status": "rejected"
	}`, http.StatusOK)
	assertAPIStatus(t, router, tokenB, http.MethodGet, "/api/v1/goals/"+goalA, "", http.StatusNotFound)
	assertGoalListContains(t, router, tokenB, goalA, false)
	if got := findGoalWallet(t, router, tokenB, goalA); got != "" {
		t.Fatalf("expected rejected invite to avoid creating goal wallet, got %q", got)
	}

	assertAPIStatus(t, router, tokenA, http.MethodPost, "/api/v1/goals/"+goalA+"/members", `{
		"email": "`+userBData.User.Email+`"
	}`, http.StatusOK)
	assertAPIStatus(t, router, tokenB, http.MethodPut, "/api/v1/goals/"+goalA+"/members/"+userB+"/respond", `{
		"status": "joined"
	}`, http.StatusOK)
	if got := findGoalWallet(t, router, tokenB, goalA); got == "" {
		t.Fatalf("expected re-invited member to receive goal wallet after joining")
	}
	if got := countGoalWallets(t, router, tokenB, goalA); got != 1 {
		t.Fatalf("expected exactly one goal wallet after joining, got %d", got)
	}
	assertAPIStatus(t, router, tokenB, http.MethodPut, "/api/v1/goals/"+goalA+"/members/"+userB+"/respond", `{
		"status": "rejected"
	}`, http.StatusBadRequest)
	assertGoalListContains(t, router, tokenB, goalA, true)
	if got := countGoalWallets(t, router, tokenB, goalA); got != 1 {
		t.Fatalf("expected joined member rejection attempt to keep exactly one goal wallet, got %d", got)
	}

	assertAPIStatus(t, router, tokenA, http.MethodPut, "/api/v1/goals/"+goalA+"/members/"+userA+"/respond", `{
		"status": "rejected"
	}`, http.StatusForbidden)
}

func findGoalWallet(t *testing.T, router http.Handler, token string, goalID string) string {
	t.Helper()
	wallets := listGoalWalletIDs(t, router, token, goalID)
	if len(wallets) == 0 {
		return ""
	}
	return wallets[0]
}

func countGoalWallets(t *testing.T, router http.Handler, token string, goalID string) int {
	t.Helper()
	return len(listGoalWalletIDs(t, router, token, goalID))
}

func listGoalWalletIDs(t *testing.T, router http.Handler, token string, goalID string) []string {
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

	var ids []string
	for _, w := range wallets.Wallets {
		if w.Type == "goal" && w.GoalID != nil && *w.GoalID == goalID {
			ids = append(ids, w.ID)
		}
	}
	return ids
}

func assertGoalListContains(t *testing.T, router http.Handler, token string, goalID string, want bool) {
	t.Helper()

	res := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/goals", "", http.StatusOK)
	var goals []goal.Goal
	if err := json.Unmarshal(res, &goals); err != nil {
		t.Fatalf("parse goals response: %v", err)
	}
	for _, g := range goals {
		if g.ID == goalID {
			if !want {
				t.Fatalf("did not expect goal %s in list: %+v", goalID, goals)
			}
			return
		}
	}
	if want {
		t.Fatalf("expected goal %s in list: %+v", goalID, goals)
	}
}

func TestGoalContributionViaTransfer(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "goal-contrib-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "goal-contrib")
	unauthID, unauthToken := registerIntegrationAPIUser(t, router, "goal-contrib-unauth")
	defer cleanupServerIntegrationUsers(t, pool, userID, unauthID)

	sourceWalletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Main Wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 10000000
	}`)

	goalID := createAPIResource(t, router, token, "/api/v1/goals", `{
		"name": "Buy Car",
		"target_amount_minor": 50000000,
		"deadline": "2027-01-01T00:00:00Z"
	}`)

	// Get goal to find its wallet
	resBody := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/goals/"+goalID, "", http.StatusOK)
	var getGoal struct {
		CollectedAmountMinor int64 `json:"collected_amount_minor"`
	}
	if err := json.Unmarshal(resBody, &getGoal); err != nil {
		t.Fatalf("parse goal: %v", err)
	}
	if getGoal.CollectedAmountMinor != 0 {
		t.Fatalf("expected 0 collected amount, got %d", getGoal.CollectedAmountMinor)
	}

	// We need to find the goal wallet ID. It's returned in /api/v1/wallets
	walletsRes := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/wallets", "", http.StatusOK)
	var walletsResp struct {
		Wallets []wallet.Wallet `json:"wallets"`
	}
	if err := json.Unmarshal(walletsRes, &walletsResp); err != nil {
		t.Fatalf("parse wallets: %v", err)
	}
	var goalWalletID string
	for _, w := range walletsResp.Wallets {
		if w.Type == "goal" && strings.Contains(w.Name, "Buy Car") {
			goalWalletID = w.ID
			break
		}
	}
	if goalWalletID == "" {
		t.Fatalf("goal wallet not found")
	}

	// Unauth user tries to transfer to the goal wallet
	unauthSourceWallet := createAPIResource(t, router, unauthToken, "/api/v1/wallets", `{
		"name": "Unauth Wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 10000000
	}`)
	assertAPIStatus(t, router, unauthToken, http.MethodPost, "/api/v1/transactions", `{
		"type": "transfer",
		"wallet_id": "`+unauthSourceWallet+`",
		"to_wallet_id": "`+goalWalletID+`",
		"amount_minor": 500000,
		"transaction_at": "`+time.Now().Format(time.RFC3339)+`"
	}`, http.StatusNotFound) // Should fail because goal wallet doesn't belong to unauth

	// Owner contributes 2,000,000
	assertAPIStatus(t, router, token, http.MethodPost, "/api/v1/transactions", `{
		"type": "transfer",
		"wallet_id": "`+sourceWalletID+`",
		"to_wallet_id": "`+goalWalletID+`",
		"amount_minor": 2000000,
		"transaction_at": "`+time.Now().Format(time.RFC3339)+`"
	}`, http.StatusCreated)

	// Check balances
	assertWalletBalance(t, router, token, sourceWalletID, 8000000)
	assertWalletBalance(t, router, token, goalWalletID, 2000000)

	// Check goal collected amount
	resBodyAfter := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/goals/"+goalID, "", http.StatusOK)
	if err := json.Unmarshal(resBodyAfter, &getGoal); err != nil {
		t.Fatalf("parse goal after: %v", err)
	}
	if getGoal.CollectedAmountMinor != 2000000 {
		t.Fatalf("expected 2000000 collected amount, got %d", getGoal.CollectedAmountMinor)
	}
}
