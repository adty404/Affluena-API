package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"affluena-api/internal/config"
)

func TestAPILogAndActivityIsolation(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "integration-test-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userAID, userAToken := registerIntegrationAPIUser(t, router, "api-log-a")
	userBID, userBToken := registerIntegrationAPIUser(t, router, "api-log-b")
	defer cleanupServerIntegrationUsers(t, pool, userAID, userBID)

	// Seed an activity for User A by creating a wallet
	walletAID := createAPIResource(t, router, userAToken, "/api/v1/wallets", `{
		"name": "Log Wallet A",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 100000
	}`)

	// Wait a bit for async activity log to be saved
	time.Sleep(100 * time.Millisecond)

	// 1. Get Activities for User A
	req, _ := http.NewRequest("GET", "/api/v1/activities", nil)
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for activities list, got %d: %s", w.Code, w.Body.String())
	}

	var activitiesResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &activitiesResp); err != nil {
		t.Fatalf("failed to parse activities: %v", err)
	}

	if len(activitiesResp.Data) == 0 {
		t.Fatalf("expected at least 1 activity for user A, got 0")
	}
	activityID := activitiesResp.Data[0].ID

	// 2. Get Activity Detail for User A
	req, _ = http.NewRequest("GET", "/api/v1/activities/"+activityID, nil)
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for activity detail, got %d", w.Code)
	}

	// 3. User B tries to get User A's activity (Isolation)
	req, _ = http.NewRequest("GET", "/api/v1/activities/"+activityID, nil)
	req.Header.Set("Authorization", "Bearer "+userBToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when user B accesses user A's activity, got %d", w.Code)
	}

	// 4. Get API Logs for User A
	req, _ = http.NewRequest("GET", "/api/v1/system-logs", nil)
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for system-logs list, got %d: %s", w.Code, w.Body.String())
	}

	var logsResp struct {
		Logs []struct {
			ID string `json:"id"`
		} `json:"logs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &logsResp); err != nil {
		t.Fatalf("failed to parse system-logs: %v", err)
	}

	if len(logsResp.Logs) == 0 {
		t.Fatalf("expected at least 1 system log for user A, got 0")
	}
	logID := logsResp.Logs[0].ID

	// 5. Get API Log Detail for User A
	req, _ = http.NewRequest("GET", "/api/v1/system-logs/"+logID, nil)
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for system-log detail, got %d", w.Code)
	}

	// 6. User B tries to get User A's API Log (Isolation)
	req, _ = http.NewRequest("GET", "/api/v1/system-logs/"+logID, nil)
	req.Header.Set("Authorization", "Bearer "+userBToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when user B accesses user A's system-log, got %d", w.Code)
	}

	// 7. 404 for unknown ID
	req, _ = http.NewRequest("GET", "/api/v1/system-logs/00000000-0000-0000-0000-000000000000", nil)
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown system-log, got %d", w.Code)
	}

	req, _ = http.NewRequest("GET", "/api/v1/activities/00000000-0000-0000-0000-000000000000", nil)
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown activity, got %d", w.Code)
	}

	_ = walletAID
}
