package server

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"affluena-api/internal/config"
	"affluena-api/internal/dashboard"
	"affluena-api/internal/transaction"
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

func TestSharedWalletOwnerSeesMemberTransactionsAndAnalyticsOnce(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "wallet-share-owner-visibility-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	ownerID, ownerToken := registerIntegrationAPIUser(t, router, "share-owner-visibility")
	memberBID, memberBToken := registerIntegrationAPIUser(t, router, "share-member-b-visibility")
	memberCID, memberCToken := registerIntegrationAPIUser(t, router, "share-member-c-visibility")
	defer cleanupServerIntegrationUsers(t, pool, ownerID)
	defer cleanupServerIntegrationUsers(t, pool, memberBID)
	defer cleanupServerIntegrationUsers(t, pool, memberCID)

	walletID := createAPIResource(t, router, ownerToken, "/api/v1/wallets", `{
		"name": "Household Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 100000
	}`)
	memberBEmail := getIntegrationUserEmail(t, router, memberBToken)
	memberCEmail := getIntegrationUserEmail(t, router, memberCToken)
	assertAPIStatus(t, router, ownerToken, http.MethodPost, "/api/v1/wallets/"+walletID+"/invites", `{
		"email": "`+memberBEmail+`"
	}`, http.StatusCreated)
	assertAPIStatus(t, router, memberBToken, http.MethodPatch, "/api/v1/wallets/"+walletID+"/members/"+memberBID, `{
		"status": "joined"
	}`, http.StatusOK)
	assertAPIStatus(t, router, ownerToken, http.MethodPost, "/api/v1/wallets/"+walletID+"/invites", `{
		"email": "`+memberCEmail+`"
	}`, http.StatusCreated)
	assertAPIStatus(t, router, memberCToken, http.MethodPatch, "/api/v1/wallets/"+walletID+"/members/"+memberCID, `{
		"status": "joined"
	}`, http.StatusOK)

	categoryID := createAPIResource(t, router, memberBToken, "/api/v1/categories", `{
		"name": "Shared Groceries",
		"type": "expense"
	}`)
	txAt := time.Now().UTC().Truncate(time.Second)
	txID := createAPIResource(t, router, memberBToken, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 10000,
		"transaction_at": "`+txAt.Format(time.RFC3339)+`",
		"note": "Shared groceries"
	}`)

	transactionsBody := performAPIRequest(t, router, ownerToken, http.MethodGet, "/api/v1/transactions?wallet_id="+walletID, "", http.StatusOK)
	var transactionsResp struct {
		Transactions []transaction.Transaction `json:"transactions"`
	}
	if err := json.Unmarshal(transactionsBody, &transactionsResp); err != nil {
		t.Fatalf("parse owner transaction list: %v", err)
	}
	if countTransactionID(transactionsResp.Transactions, txID) != 1 {
		t.Fatalf("expected owner transaction list to contain member transaction %s once, got %+v", txID, transactionsResp.Transactions)
	}

	month := txAt.Format("2006-01")
	summaryBody := performAPIRequest(t, router, ownerToken, http.MethodGet, "/api/v1/dashboard/summary?month="+month, "", http.StatusOK)
	var summary struct {
		MonthlyExpenseMinor int64 `json:"monthly_expense_minor"`
		NetWorthMinor       int64 `json:"net_worth_minor"`
	}
	if err := json.Unmarshal(summaryBody, &summary); err != nil {
		t.Fatalf("parse owner dashboard summary: %v", err)
	}
	if summary.MonthlyExpenseMinor != 10000 || summary.NetWorthMinor != 90000 {
		t.Fatalf("expected owner summary expense/net worth 10000/90000, got %+v", summary)
	}

	trendBody := performAPIRequest(t, router, ownerToken, http.MethodGet, "/api/v1/dashboard/cashflow-trend?months=1", "", http.StatusOK)
	var trendResp struct {
		Trend []dashboard.CashflowTrend `json:"trend"`
	}
	if err := json.Unmarshal(trendBody, &trendResp); err != nil {
		t.Fatalf("parse owner cashflow trend: %v", err)
	}
	if len(trendResp.Trend) != 1 || trendResp.Trend[0].ExpenseMinor != 10000 {
		t.Fatalf("expected owner cashflow trend expense 10000, got %+v", trendResp.Trend)
	}

	forecastBody := performAPIRequest(t, router, ownerToken, http.MethodGet, "/api/v1/dashboard/forecast?month="+month, "", http.StatusOK)
	var forecast dashboard.Forecast
	if err := json.Unmarshal(forecastBody, &forecast); err != nil {
		t.Fatalf("parse owner forecast: %v", err)
	}
	if forecast.CurrentExpenseMinor != 10000 {
		t.Fatalf("expected owner forecast current expense 10000, got %+v", forecast)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/export/csv", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected owner export status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	records, err := csv.NewReader(strings.NewReader(recorder.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("parse owner export CSV: %v", err)
	}
	if countCSVRowsWithID(records, txID) != 1 {
		t.Fatalf("expected owner export to contain member transaction %s once, got %v", txID, records)
	}
}

func TestSharedWalletMemberCanUseQuickEntryTemplate(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "wallet-share-quick-entry-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	ownerID, ownerToken := registerIntegrationAPIUser(t, router, "share-quick-owner")
	memberID, memberToken := registerIntegrationAPIUser(t, router, "share-quick-member")
	defer cleanupServerIntegrationUsers(t, pool, ownerID)
	defer cleanupServerIntegrationUsers(t, pool, memberID)

	sharedWalletID := createAPIResource(t, router, ownerToken, "/api/v1/wallets", `{
		"name": "Shared Quick Entry Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 100000
	}`)
	memberEmail := getIntegrationUserEmail(t, router, memberToken)
	assertAPIStatus(t, router, ownerToken, http.MethodPost, "/api/v1/wallets/"+sharedWalletID+"/invites", `{
		"email": "`+memberEmail+`"
	}`, http.StatusCreated)
	assertAPIStatus(t, router, memberToken, http.MethodPatch, "/api/v1/wallets/"+sharedWalletID+"/members/"+memberID, `{
		"status": "joined"
	}`, http.StatusOK)

	categoryID := createAPIResource(t, router, memberToken, "/api/v1/categories", `{
		"name": "Shared Quick Groceries",
		"type": "expense"
	}`)
	templateID := createAPIResource(t, router, memberToken, "/api/v1/quick-entry-templates", `{
		"name": "Shared grocery run",
		"type": "expense",
		"wallet_id": "`+sharedWalletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 25000,
		"note": "Quick shared grocery"
	}`)
	executeBody := performAPIRequest(t, router, memberToken, http.MethodPost, "/api/v1/quick-entry-templates/"+templateID+"/execute", `{
		"transaction_at": "2026-06-14T08:30:00Z"
	}`, http.StatusCreated)
	var executed struct {
		Transaction transaction.Transaction `json:"transaction"`
	}
	if err := json.Unmarshal(executeBody, &executed); err != nil {
		t.Fatalf("parse quick entry execution response: %v", err)
	}
	if executed.Transaction.WalletID != sharedWalletID || executed.Transaction.AmountMinor != 25000 {
		t.Fatalf("expected quick entry to create shared-wallet expense, got %+v", executed.Transaction)
	}

	assertWalletBalance(t, router, ownerToken, sharedWalletID, 75000)
}

func TestFormerSharedWalletMemberCannotMutateWalletBalanceThroughOldTransactions(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "wallet-share-former-member-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	ownerID, ownerToken := registerIntegrationAPIUser(t, router, "former-member-owner")
	memberID, memberToken := registerIntegrationAPIUser(t, router, "former-member")
	defer cleanupServerIntegrationUsers(t, pool, ownerID)
	defer cleanupServerIntegrationUsers(t, pool, memberID)

	sharedWalletID := createAPIResource(t, router, ownerToken, "/api/v1/wallets", `{
		"name": "Former Member Shared Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 100000
	}`)
	memberEmail := getIntegrationUserEmail(t, router, memberToken)
	assertAPIStatus(t, router, ownerToken, http.MethodPost, "/api/v1/wallets/"+sharedWalletID+"/invites", `{
		"email": "`+memberEmail+`"
	}`, http.StatusCreated)
	assertAPIStatus(t, router, memberToken, http.MethodPatch, "/api/v1/wallets/"+sharedWalletID+"/members/"+memberID, `{
		"status": "joined"
	}`, http.StatusOK)

	expenseCategoryID := createAPIResource(t, router, memberToken, "/api/v1/categories", `{
		"name": "Former Member Expense",
		"type": "expense"
	}`)
	updateAttemptTxID := createAPIResource(t, router, memberToken, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+sharedWalletID+`",
		"category_id": "`+expenseCategoryID+`",
		"amount_minor": 10000,
		"transaction_at": "2026-06-14T08:00:00Z"
	}`)
	deleteAttemptTxID := createAPIResource(t, router, memberToken, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+sharedWalletID+`",
		"category_id": "`+expenseCategoryID+`",
		"amount_minor": 15000,
		"transaction_at": "2026-06-14T09:00:00Z"
	}`)
	assertWalletBalance(t, router, ownerToken, sharedWalletID, 75000)

	assertAPIStatus(t, router, memberToken, http.MethodPatch, "/api/v1/wallets/"+sharedWalletID+"/members/"+memberID, `{
		"status": "rejected"
	}`, http.StatusOK)
	memberWalletID := createAPIResource(t, router, memberToken, "/api/v1/wallets", `{
		"name": "Former Member Personal Wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 50000
	}`)

	assertAPIStatus(t, router, memberToken, http.MethodPut, "/api/v1/transactions/"+updateAttemptTxID, `{
		"type": "expense",
		"wallet_id": "`+memberWalletID+`",
		"category_id": "`+expenseCategoryID+`",
		"amount_minor": 5000,
		"transaction_at": "2026-06-14T08:00:00Z"
	}`, http.StatusNotFound)
	assertAPIStatus(t, router, memberToken, http.MethodDelete, "/api/v1/transactions/"+deleteAttemptTxID, "", http.StatusNotFound)

	assertWalletBalance(t, router, ownerToken, sharedWalletID, 75000)
	assertWalletBalance(t, router, memberToken, memberWalletID, 50000)
	assertAPIStatus(t, router, ownerToken, http.MethodGet, "/api/v1/transactions/"+updateAttemptTxID, "", http.StatusOK)
	assertAPIStatus(t, router, ownerToken, http.MethodGet, "/api/v1/transactions/"+deleteAttemptTxID, "", http.StatusOK)
}

func TestSharedWalletOwnerCannotMutateMemberTransaction(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "wallet-share-owner-mutate-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	ownerID, ownerToken := registerIntegrationAPIUser(t, router, "share-owner-mutate")
	memberID, memberToken := registerIntegrationAPIUser(t, router, "share-member-mutate")
	defer cleanupServerIntegrationUsers(t, pool, ownerID)
	defer cleanupServerIntegrationUsers(t, pool, memberID)

	walletID := createAPIResource(t, router, ownerToken, "/api/v1/wallets", `{
		"name": "Household Mutate Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 100000
	}`)
	memberEmail := getIntegrationUserEmail(t, router, memberToken)
	assertAPIStatus(t, router, ownerToken, http.MethodPost, "/api/v1/wallets/"+walletID+"/invites", `{
		"email": "`+memberEmail+`"
	}`, http.StatusCreated)
	assertAPIStatus(t, router, memberToken, http.MethodPatch, "/api/v1/wallets/"+walletID+"/members/"+memberID, `{
		"status": "joined"
	}`, http.StatusOK)

	categoryID := createAPIResource(t, router, memberToken, "/api/v1/categories", `{
		"name": "Member Mutate Groceries",
		"type": "expense"
	}`)

	// Member creates a transaction
	txID := createAPIResource(t, router, memberToken, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 20000,
		"transaction_at": "2026-06-15T08:00:00Z"
	}`)

	// Owner attempts to UPDATE member's transaction - should fail with Forbidden
	assertAPIStatus(t, router, ownerToken, http.MethodPut, "/api/v1/transactions/"+txID, `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 50000,
		"transaction_at": "2026-06-15T08:00:00Z"
	}`, http.StatusForbidden)

	// Owner attempts to DELETE member's transaction - should fail with Forbidden
	assertAPIStatus(t, router, ownerToken, http.MethodDelete, "/api/v1/transactions/"+txID, "", http.StatusForbidden)

	// Unrelated user attempts to mutate - should fail with Not Found (because they don't have access to wallet/tx)
	unrelatedID, unrelatedToken := registerIntegrationAPIUser(t, router, "share-unrelated-mutate")
	defer cleanupServerIntegrationUsers(t, pool, unrelatedID)
	assertAPIStatus(t, router, unrelatedToken, http.MethodPut, "/api/v1/transactions/"+txID, `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 50000,
		"transaction_at": "2026-06-15T08:00:00Z"
	}`, http.StatusForbidden)

	// Member CAN UPDATE their own transaction
	assertAPIStatus(t, router, memberToken, http.MethodPut, "/api/v1/transactions/"+txID, `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 25000,
		"transaction_at": "2026-06-15T08:00:00Z"
	}`, http.StatusOK)

	// Member CAN DELETE their own transaction
	assertAPIStatus(t, router, memberToken, http.MethodDelete, "/api/v1/transactions/"+txID, "", http.StatusNoContent)
}

// TestPendingInviteeCannotSeeWalletBalanceOrRoster locks the wallet-sharing
// isolation fix: a pending (not-yet-accepted) invitee may see that the wallet
// exists (so they can accept the invite) but must NOT see its balance or the
// member roster (which would leak other members' emails). Both become visible
// once they join.
func TestPendingInviteeCannotSeeWalletBalanceOrRoster(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "wallet-share-pending-leak-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	ownerID, ownerToken := registerIntegrationAPIUser(t, router, "pending-leak-owner")
	inviteeID, inviteeToken := registerIntegrationAPIUser(t, router, "pending-leak-invitee")
	defer cleanupServerIntegrationUsers(t, pool, ownerID)
	defer cleanupServerIntegrationUsers(t, pool, inviteeID)

	walletID := createAPIResource(t, router, ownerToken, "/api/v1/wallets", `{
		"name": "Secret Balance Wallet",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 500000
	}`)
	inviteeEmail := getIntegrationUserEmail(t, router, inviteeToken)
	assertAPIStatus(t, router, ownerToken, http.MethodPost, "/api/v1/wallets/"+walletID+"/invites", `{
		"email": "`+inviteeEmail+`"
	}`, http.StatusCreated)

	// While PENDING: GET /wallets/:id is visible but balance + roster are hidden.
	pendingBody := performAPIRequest(t, router, inviteeToken, http.MethodGet, "/api/v1/wallets/"+walletID, "", http.StatusOK)
	var pendingWallet wallet.Wallet
	if err := json.Unmarshal(pendingBody, &pendingWallet); err != nil {
		t.Fatalf("parse pending wallet: %v", err)
	}
	if pendingWallet.ShareStatus != "pending" {
		t.Fatalf("expected pending share_status, got %q", pendingWallet.ShareStatus)
	}
	if pendingWallet.BalanceMinor != 0 {
		t.Fatalf("pending invitee leaked balance via Get: got %d, want 0", pendingWallet.BalanceMinor)
	}
	if len(pendingWallet.Members) != 0 {
		t.Fatalf("pending invitee leaked member roster via Get: got %d members", len(pendingWallet.Members))
	}

	// Same hiding via the list endpoint.
	listBody := performAPIRequest(t, router, inviteeToken, http.MethodGet, "/api/v1/wallets", "", http.StatusOK)
	var listResp struct {
		Wallets []wallet.Wallet `json:"wallets"`
	}
	if err := json.Unmarshal(listBody, &listResp); err != nil {
		t.Fatalf("parse wallet list: %v", err)
	}
	if len(listResp.Wallets) != 1 {
		t.Fatalf("expected 1 wallet for pending invitee, got %d", len(listResp.Wallets))
	}
	if listResp.Wallets[0].BalanceMinor != 0 {
		t.Fatalf("pending invitee leaked balance via List: got %d, want 0", listResp.Wallets[0].BalanceMinor)
	}

	// After joining, balance and roster become visible.
	assertAPIStatus(t, router, inviteeToken, http.MethodPatch, "/api/v1/wallets/"+walletID+"/members/"+inviteeID, `{
		"status": "joined"
	}`, http.StatusOK)
	joinedBody := performAPIRequest(t, router, inviteeToken, http.MethodGet, "/api/v1/wallets/"+walletID, "", http.StatusOK)
	var joinedWallet wallet.Wallet
	if err := json.Unmarshal(joinedBody, &joinedWallet); err != nil {
		t.Fatalf("parse joined wallet: %v", err)
	}
	if joinedWallet.BalanceMinor != 500000 {
		t.Fatalf("joined member should see balance 500000, got %d", joinedWallet.BalanceMinor)
	}
	if len(joinedWallet.Members) < 2 {
		t.Fatalf("joined member should see the full roster, got %d members", len(joinedWallet.Members))
	}
}

func getIntegrationUserEmail(t *testing.T, router http.Handler, token string) string {
	t.Helper()

	body := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/auth/me", "", http.StatusOK)
	var parsed struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse auth/me response: %v", err)
	}
	if parsed.User.Email == "" {
		t.Fatalf("auth/me response missing email: %s", string(body))
	}
	return parsed.User.Email
}

func countTransactionID(transactions []transaction.Transaction, id string) int {
	count := 0
	for _, tx := range transactions {
		if tx.ID == id {
			count++
		}
	}
	return count
}

func countCSVRowsWithID(records [][]string, id string) int {
	count := 0
	for _, record := range records {
		if len(record) > 0 && record[0] == id {
			count++
		}
	}
	return count
}
