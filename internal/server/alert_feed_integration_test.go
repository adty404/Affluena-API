package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"affluena-api/internal/alert"
	"affluena-api/internal/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlertFeedIntegration(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "alert-integration-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	user1ID, token1 := registerIntegrationAPIUser(t, router, "alertuser1")
	user2ID, token2 := registerIntegrationAPIUser(t, router, "alertuser2")
	defer cleanupServerIntegrationUsers(t, pool, user1ID, user2ID)

	ctx := context.Background()

	// Setup data for user1
	// 1. Budget overrun
	catID := createAPIResource(t, router, token1, "/api/v1/categories", `{
		"name": "Food",
		"type": "expense"
	}`)

	monthValue := time.Now().UTC().Format("2006-01")
	monthDate, _ := time.Parse("2006-01-02", monthValue+"-01")

	budgetID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO category_budgets (id, user_id, category_id, month, limit_minor) VALUES ($1, $2, $3, $4, $5)`, budgetID, user1ID, catID, monthDate, 1000)
	require.NoError(t, err)

	// Add transaction to exceed budget
	walletID := createAPIResource(t, router, token1, "/api/v1/wallets", `{
		"name": "Cash",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 10000
	}`)

	txnID := createAPIResource(t, router, token1, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+catID+`",
		"amount_minor": 850,
		"transaction_at": "`+time.Now().UTC().Format(time.RFC3339)+`",
		"note": "Food"
	}`)

	// 2. Overdue debt
	debtID1 := uuid.New().String()
	dueDate := time.Now().UTC().Add(-24 * time.Hour)
	_, err = pool.Exec(ctx, `INSERT INTO debts (id, user_id, type, counterparty_name, wallet_id, disbursement_category_id, payment_category_id, origination_transaction_id, principal_amount_minor, paid_amount_minor, opened_at, due_date, status) VALUES ($1, $2, 'payable', 'John Doe', $3, $4, $4, $5, 1000, 0, $6, $7, 'open')`, debtID1, user1ID, walletID, catID, txnID, time.Now().UTC().Add(-48*time.Hour), dueDate)
	require.NoError(t, err)

	// 3. Recent recurring activity
	actID := uuid.New().String()
	entityID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO user_activities (id, user_id, action_type, entity_type, entity_id, description, created_at) VALUES ($1, $2, 'RUN_FAILED', 'RECURRING', $3, 'Failed to run', $4)`, actID, user1ID, entityID, time.Now().UTC().Add(-2*time.Hour))
	require.NoError(t, err)

	// Setup data for user2 (should not be seen by user1)
	catID2 := createAPIResource(t, router, token2, "/api/v1/categories", `{
		"name": "Food",
		"type": "expense"
	}`)
	walletID2 := createAPIResource(t, router, token2, "/api/v1/wallets", `{
		"name": "Cash",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 10000
	}`)
	txnID2 := createAPIResource(t, router, token2, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID2+`",
		"category_id": "`+catID2+`",
		"amount_minor": 850,
		"transaction_at": "`+time.Now().UTC().Format(time.RFC3339)+`",
		"note": "Food"
	}`)

	debtID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO debts (id, user_id, type, counterparty_name, wallet_id, disbursement_category_id, payment_category_id, origination_transaction_id, principal_amount_minor, paid_amount_minor, opened_at, due_date, status) VALUES ($1, $2, 'payable', 'Jane Doe', $3, $4, $4, $5, 1000, 0, $6, $7, 'open')`, debtID2, user2ID, walletID2, catID2, txnID2, time.Now().UTC().Add(-48*time.Hour), dueDate)
	require.NoError(t, err)

	t.Run("List Alerts", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/alerts", nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var resp struct {
			Alerts []alert.Alert `json:"alerts"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Len(t, resp.Alerts, 3)

		// Check types
		types := make(map[string]bool)
		for _, a := range resp.Alerts {
			types[a.Type] = true
		}
		assert.True(t, types["budget"])
		assert.True(t, types["debt"])
		assert.True(t, types["recurring"])
	})

	t.Run("List Alerts Isolation", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/alerts", nil)
		req.Header.Set("Authorization", "Bearer "+token2)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var resp struct {
			Alerts []alert.Alert `json:"alerts"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Len(t, resp.Alerts, 1)
		assert.Equal(t, "debt", resp.Alerts[0].Type)
		assert.Equal(t, "debt-"+debtID2, resp.Alerts[0].ID)
	})

	t.Run("Get Alert Found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/alerts/debt-"+debtID1, nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var a alert.Alert
		err := json.Unmarshal(w.Body.Bytes(), &a)
		require.NoError(t, err)

		assert.Equal(t, "debt-"+debtID1, a.ID)
		assert.Equal(t, "debt", a.Type)
	})

	t.Run("Get Alert Not Found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/alerts/debt-"+uuid.New().String(), nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
