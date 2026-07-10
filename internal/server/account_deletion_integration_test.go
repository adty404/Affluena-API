package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"affluena-api/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The account-deletion tests use the same password as
// registerIntegrationAPIUser so wrong-password cases are explicit.
const accountDeletionPassword = "super-secret-password"

func newAccountDeletionRouter(t *testing.T, pool *pgxpool.Pool) http.Handler {
	t.Helper()
	return NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "integration-test-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)
}

// registerAccountDeletionUser mirrors registerIntegrationAPIUser but also
// returns the generated email so tests can attempt logins after deletion.
func registerAccountDeletionUser(t *testing.T, router http.Handler, label string) (string, string, string) {
	t.Helper()

	email := label + "-" + time.Now().UTC().Format("20060102150405.000000000") + "@example.test"
	body := `{"email":"` + email + `","password":"` + accountDeletionPassword + `"}`
	response := performAPIRequest(t, router, "", http.MethodPost, "/api/v1/auth/register", body, http.StatusCreated)

	var parsed struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(response, &parsed); err != nil {
		t.Fatalf("parse register response: %v", err)
	}
	if parsed.User.ID == "" || parsed.Tokens.AccessToken == "" {
		t.Fatalf("register response missing user id or access token: %s", string(response))
	}
	return parsed.User.ID, parsed.Tokens.AccessToken, email
}

func countAccountDeletionRows(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&count); err != nil {
		t.Fatalf("count query %q: %v", query, err)
	}
	return count
}

func TestDeleteAccountWipesAllUserData(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := newAccountDeletionRouter(t, pool)

	userID, token, email := registerAccountDeletionUser(t, router, "delete-account-full")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Delete me wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 100000
	}`)
	categoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Delete me income",
		"type": "income"
	}`)
	createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 50000,
		"transaction_at": "2026-07-01T08:00:00Z"
	}`)

	assertAPIStatus(t, router, token, http.MethodDelete, "/api/v1/auth/account", `{
		"password": "`+accountDeletionPassword+`"
	}`, http.StatusNoContent)

	// Credentials are gone: login must fail with 401.
	assertAPIStatus(t, router, "", http.MethodPost, "/api/v1/auth/login", `{
		"email": "`+email+`",
		"password": "`+accountDeletionPassword+`"
	}`, http.StatusUnauthorized)

	// Direct SQL: the user row and every owned row are gone (including the 8
	// onboarding categories and the starter wallet seeded at registration).
	for _, check := range []struct {
		name  string
		query string
	}{
		{"users", `SELECT COUNT(*) FROM users WHERE id = $1`},
		{"wallets", `SELECT COUNT(*) FROM wallets WHERE user_id = $1`},
		{"categories", `SELECT COUNT(*) FROM categories WHERE user_id = $1`},
		{"transactions", `SELECT COUNT(*) FROM transactions WHERE user_id = $1`},
		{"refresh_tokens", `SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1`},
	} {
		if got := countAccountDeletionRows(t, pool, check.query, userID); got != 0 {
			t.Fatalf("expected 0 %s rows after account deletion, got %d", check.name, got)
		}
	}
}

func TestDeleteAccountWrongPasswordKeepsEverything(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := newAccountDeletionRouter(t, pool)

	userID, token, email := registerAccountDeletionUser(t, router, "delete-account-wrongpw")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Survivor wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 5000
	}`)

	assertAPIStatus(t, router, token, http.MethodDelete, "/api/v1/auth/account", `{
		"password": "definitely-not-the-password"
	}`, http.StatusUnauthorized)

	if got := countAccountDeletionRows(t, pool, `SELECT COUNT(*) FROM users WHERE id = $1`, userID); got != 1 {
		t.Fatalf("expected user row to survive wrong-password delete, got %d rows", got)
	}
	// Starter wallet + the one above; 8 onboarding categories.
	if got := countAccountDeletionRows(t, pool, `SELECT COUNT(*) FROM wallets WHERE user_id = $1`, userID); got != 2 {
		t.Fatalf("expected 2 wallets to survive, got %d", got)
	}
	if got := countAccountDeletionRows(t, pool, `SELECT COUNT(*) FROM categories WHERE user_id = $1`, userID); got != 8 {
		t.Fatalf("expected 8 categories to survive, got %d", got)
	}

	assertAPIStatus(t, router, "", http.MethodPost, "/api/v1/auth/login", `{
		"email": "`+email+`",
		"password": "`+accountDeletionPassword+`"
	}`, http.StatusOK)
}

func TestDeleteAccountMissingPasswordIsBadRequest(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := newAccountDeletionRouter(t, pool)

	userID, token, _ := registerAccountDeletionUser(t, router, "delete-account-nopw")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	assertAPIStatus(t, router, token, http.MethodDelete, "/api/v1/auth/account", `{}`, http.StatusBadRequest)
	assertAPIStatus(t, router, token, http.MethodDelete, "/api/v1/auth/account", ``, http.StatusBadRequest)

	if got := countAccountDeletionRows(t, pool, `SELECT COUNT(*) FROM users WHERE id = $1`, userID); got != 1 {
		t.Fatalf("expected user row to survive missing-password delete, got %d rows", got)
	}
}

func TestDeleteAccountRemovesSharedWalletLinkVisibility(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := newAccountDeletionRouter(t, pool)

	ownerID, ownerToken, _ := registerAccountDeletionUser(t, router, "delete-account-share-owner")
	defer cleanupServerIntegrationUsers(t, pool, ownerID)
	viewerID, viewerToken, viewerEmail := registerAccountDeletionUser(t, router, "delete-account-share-viewer")
	defer cleanupServerIntegrationUsers(t, pool, viewerID)

	walletID := createAPIResource(t, router, ownerToken, "/api/v1/wallets", `{
		"name": "Owner shared wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 250000
	}`)

	// Owner shares ALL wallets with the viewer (Berbagi Dompet) and the viewer
	// accepts the link.
	assertAPIStatus(t, router, ownerToken, http.MethodPost, "/api/v1/partners/invites", `{
		"email": "`+viewerEmail+`"
	}`, http.StatusCreated)

	linksResponse := performAPIRequest(t, router, viewerToken, http.MethodGet, "/api/v1/partners", "", http.StatusOK)
	var links struct {
		Partners []struct {
			ID        string `json:"id"`
			Direction string `json:"direction"`
			Status    string `json:"status"`
		} `json:"partners"`
	}
	if err := json.Unmarshal(linksResponse, &links); err != nil {
		t.Fatalf("parse partners response: %v", err)
	}
	linkID := ""
	for _, link := range links.Partners {
		if link.Direction == "incoming" && link.Status == "pending" {
			linkID = link.ID
		}
	}
	if linkID == "" {
		t.Fatalf("viewer has no pending incoming share link: %s", string(linksResponse))
	}
	assertAPIStatus(t, router, viewerToken, http.MethodPatch, "/api/v1/partners/"+linkID, `{
		"status": "joined"
	}`, http.StatusOK)

	viewerWallets := performAPIRequest(t, router, viewerToken, http.MethodGet, "/api/v1/wallets?limit=100", "", http.StatusOK)
	if !strings.Contains(string(viewerWallets), walletID) {
		t.Fatalf("expected viewer to see owner's wallet %s before deletion: %s", walletID, string(viewerWallets))
	}

	assertAPIStatus(t, router, ownerToken, http.MethodDelete, "/api/v1/auth/account", `{
		"password": "`+accountDeletionPassword+`"
	}`, http.StatusNoContent)

	// The viewer's wallet list no longer includes the deleted owner's wallets.
	viewerWallets = performAPIRequest(t, router, viewerToken, http.MethodGet, "/api/v1/wallets?limit=100", "", http.StatusOK)
	if strings.Contains(string(viewerWallets), walletID) {
		t.Fatalf("viewer still sees deleted owner's wallet %s: %s", walletID, string(viewerWallets))
	}

	if got := countAccountDeletionRows(t, pool, `SELECT COUNT(*) FROM wallet_share_links WHERE owner_id = $1 OR viewer_id = $1`, ownerID); got != 0 {
		t.Fatalf("expected owner's wallet_share_links rows gone, got %d", got)
	}
	if got := countAccountDeletionRows(t, pool, `SELECT COUNT(*) FROM wallet_shares WHERE user_id = $1`, viewerID); got != 0 {
		t.Fatalf("expected viewer's wallet_shares rows onto deleted wallets gone, got %d", got)
	}
	if got := countAccountDeletionRows(t, pool, `SELECT COUNT(*) FROM users WHERE id = $1`, viewerID); got != 1 {
		t.Fatalf("expected viewer account untouched, got %d rows", got)
	}
}

// TestDeleteAccountGoalOwnerAndSharedWalletWriter covers the two schemas the
// bare users-cascade would abort on (verified against Postgres 17):
//  1. wallets.goal_id -> goals ON DELETE CASCADE would drag ANOTHER member's
//     goal wallet down with the owner's goals and trip that member's
//     contribution transactions (ON DELETE RESTRICT).
//  2. A joined shared-wallet member's transactions on the owner's wallet
//     RESTRICT-block the owner's wallet cascade.
func TestDeleteAccountGoalOwnerAndSharedWalletWriter(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := newAccountDeletionRouter(t, pool)

	ownerID, ownerToken, _ := registerAccountDeletionUser(t, router, "delete-account-goal-owner")
	defer cleanupServerIntegrationUsers(t, pool, ownerID)
	memberID, memberToken, memberEmail := registerAccountDeletionUser(t, router, "delete-account-goal-member")
	defer cleanupServerIntegrationUsers(t, pool, memberID)

	// Owner creates a goal; the member joins and contributes into their own
	// goal wallet via a transfer.
	goalID := createAPIResource(t, router, ownerToken, "/api/v1/goals", `{
		"name": "Liburan Bersama",
		"target_amount_minor": 1000000,
		"deadline": "2030-01-01T00:00:00Z"
	}`)
	assertAPIStatus(t, router, ownerToken, http.MethodPost, "/api/v1/goals/"+goalID+"/members", `{
		"email": "`+memberEmail+`"
	}`, http.StatusOK)
	assertAPIStatus(t, router, memberToken, http.MethodPut, "/api/v1/goals/"+goalID+"/members/"+memberID+"/respond", `{
		"status": "joined"
	}`, http.StatusOK)

	var memberGoalWalletID string
	if err := pool.QueryRow(context.Background(),
		`SELECT id::text FROM wallets WHERE user_id = $1 AND goal_id = $2`,
		memberID, goalID,
	).Scan(&memberGoalWalletID); err != nil {
		t.Fatalf("member goal wallet not found: %v", err)
	}

	memberCashWalletID := createAPIResource(t, router, memberToken, "/api/v1/wallets", `{
		"name": "Member cash",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 100000
	}`)
	createAPIResource(t, router, memberToken, "/api/v1/transactions", `{
		"type": "transfer",
		"wallet_id": "`+memberCashWalletID+`",
		"to_wallet_id": "`+memberGoalWalletID+`",
		"amount_minor": 25000,
		"transaction_at": "2026-07-01T08:00:00Z"
	}`)

	// Owner also shares a wallet with the member (read+write) and the member
	// writes a transaction on it.
	sharedWalletID := createAPIResource(t, router, ownerToken, "/api/v1/wallets", `{
		"name": "Owner family wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 500000
	}`)
	assertAPIStatus(t, router, ownerToken, http.MethodPost, "/api/v1/wallets/"+sharedWalletID+"/invites", `{
		"email": "`+memberEmail+`"
	}`, http.StatusCreated)
	assertAPIStatus(t, router, memberToken, http.MethodPatch, "/api/v1/wallets/"+sharedWalletID+"/members/"+memberID, `{
		"status": "joined"
	}`, http.StatusOK)
	memberCategoryID := createAPIResource(t, router, memberToken, "/api/v1/categories", `{
		"name": "Member groceries",
		"type": "expense"
	}`)
	createAPIResource(t, router, memberToken, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+sharedWalletID+`",
		"category_id": "`+memberCategoryID+`",
		"amount_minor": 7000,
		"transaction_at": "2026-07-02T08:00:00Z"
	}`)

	// The whole point: this must be a 204, not a 500 from a cascade abort.
	assertAPIStatus(t, router, ownerToken, http.MethodDelete, "/api/v1/auth/account", `{
		"password": "`+accountDeletionPassword+`"
	}`, http.StatusNoContent)

	if got := countAccountDeletionRows(t, pool, `SELECT COUNT(*) FROM users WHERE id = $1`, ownerID); got != 0 {
		t.Fatalf("expected owner deleted, got %d rows", got)
	}
	if got := countAccountDeletionRows(t, pool, `SELECT COUNT(*) FROM goals WHERE user_id = $1`, ownerID); got != 0 {
		t.Fatalf("expected owner's goals deleted, got %d rows", got)
	}

	// The member keeps their money: the goal wallet is detached from the dead
	// goal and converted to a manageable cash wallet, contribution intact.
	var walletType string
	var goalRef *string
	if err := pool.QueryRow(context.Background(),
		`SELECT type, goal_id::text FROM wallets WHERE id = $1`,
		memberGoalWalletID,
	).Scan(&walletType, &goalRef); err != nil {
		t.Fatalf("expected member's goal wallet to survive: %v", err)
	}
	if walletType != "cash" || goalRef != nil {
		t.Fatalf("expected detached cash wallet, got type=%q goal_id=%v", walletType, goalRef)
	}
	if got := countAccountDeletionRows(t, pool,
		`SELECT COUNT(*) FROM transactions WHERE user_id = $1 AND to_wallet_id = $2`,
		memberID, memberGoalWalletID,
	); got != 1 {
		t.Fatalf("expected member's goal contribution transaction to survive, got %d", got)
	}

	// The member's records on the deleted owner's shared wallet are gone with it.
	if got := countAccountDeletionRows(t, pool,
		`SELECT COUNT(*) FROM transactions WHERE user_id = $1 AND wallet_id = $2`,
		memberID, sharedWalletID,
	); got != 0 {
		t.Fatalf("expected member's transactions on deleted shared wallet gone, got %d", got)
	}
	if got := countAccountDeletionRows(t, pool, `SELECT COUNT(*) FROM users WHERE id = $1`, memberID); got != 1 {
		t.Fatalf("expected member account untouched, got %d rows", got)
	}
}
