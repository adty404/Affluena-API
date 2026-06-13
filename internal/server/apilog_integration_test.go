package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"affluena-api/internal/config"
)

func TestAPILogsAreSaved(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "test",
		JWTSecret:            "secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "apilog-user")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	// Make a request
	req, _ := http.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Sleep a bit to allow async log saving to complete
	time.Sleep(50 * time.Millisecond)

	// Check if log was saved
	var count int
	var logUserID *string
	var method, path string

	err := pool.QueryRow(context.Background(), `
		SELECT count(*), max(user_id::text), max(method), max(path)
		FROM api_logs
		WHERE user_id = $1
	`, userID).Scan(&count, &logUserID, &method, &path)

	if err != nil {
		t.Fatalf("query api_logs: %v", err)
	}

	if count == 0 {
		t.Fatalf("expected api log to be saved, got 0")
	}

	if logUserID == nil || *logUserID != userID {
		t.Fatalf("expected user_id %s, got %v", userID, logUserID)
	}

	if method != "GET" {
		t.Fatalf("expected method GET, got %s", method)
	}

	if path != "/api/v1/auth/me" {
		t.Fatalf("expected path /api/v1/auth/me, got %s", path)
	}
}

func TestHealthCheckIsNotLogged(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env: "test",
	}, pool)

	var initialCount int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM api_logs WHERE path = '/healthz'`).Scan(&initialCount)

	// Make a request to healthz
	req, _ := http.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Sleep a bit to ensure async has time
	time.Sleep(50 * time.Millisecond)

	var finalCount int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM api_logs WHERE path = '/healthz'`).Scan(&finalCount)

	if finalCount != initialCount {
		t.Fatalf("expected healthz logs to remain %d, got %d", initialCount, finalCount)
	}
}
