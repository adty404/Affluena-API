package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"affluena-api/internal/apilog"
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
	var reqPayload, respPayload *string

	err := pool.QueryRow(context.Background(), `
		SELECT count(*), max(user_id::text), max(method), max(path), max(request_payload), max(response_payload)
		FROM api_logs
		WHERE user_id = $1
	`, userID).Scan(&count, &logUserID, &method, &path, &reqPayload, &respPayload)

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

	if reqPayload != nil && *reqPayload != "" {
		t.Fatalf("expected empty request payload for GET, got %s", *reqPayload)
	}

	if respPayload == nil || len(*respPayload) < 10 {
		t.Fatalf("expected non-empty response payload, got %v", respPayload)
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

func TestAuthResponsesAreMaskedInAPILogs(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "test",
		JWTSecret:            "secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, _ := registerIntegrationAPIUser(t, router, "apilog-mask")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	time.Sleep(50 * time.Millisecond)

	rows, err := pool.Query(context.Background(), `
		SELECT COALESCE(request_payload, ''), COALESCE(response_payload, '')
		FROM api_logs
		WHERE path = '/api/v1/auth/register'
		ORDER BY created_at DESC
		LIMIT 1
	`)
	if err != nil {
		t.Fatalf("query auth api log: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("expected auth register api log")
	}
	var requestPayload string
	var responsePayload string
	if err := rows.Scan(&requestPayload, &responsePayload); err != nil {
		t.Fatalf("scan auth api log: %v", err)
	}
	for _, payload := range []string{requestPayload, responsePayload} {
		if payload != `{"masked": true}` {
			t.Fatalf("expected auth payload to be masked, got %s", payload)
		}
		if strings.Contains(payload, "access_token") || strings.Contains(payload, "refresh_token") || strings.Contains(payload, "password") {
			t.Fatalf("expected auth payload to avoid secrets, got %s", payload)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate auth api log: %v", err)
	}
}

// TestAPILogRetentionDeleteOlderThan verifies the retention prune removes rows
// older than the cutoff while leaving recent rows intact. Backed by the
// idx_api_logs_created_at index (migration 000013).
func TestAPILogRetentionDeleteOlderThan(t *testing.T) {
	pool := openServerIntegrationPool(t)
	ctx := context.Background()

	// Two rows with an explicit created_at: one old (45 days) and one fresh.
	oldPath := "/retention-old-" + time.Now().UTC().Format("20060102150405.000000000")
	newPath := "/retention-new-" + time.Now().UTC().Format("20060102150405.000000000")
	insert := func(path string, createdAt time.Time) {
		_, err := pool.Exec(ctx, `
			INSERT INTO api_logs (method, path, status_code, latency_ms, client_ip, created_at)
			VALUES ('GET', $1, 200, 1, '127.0.0.1', $2)`, path, createdAt)
		if err != nil {
			t.Fatalf("insert api_log: %v", err)
		}
	}
	now := time.Now().UTC()
	insert(oldPath, now.AddDate(0, 0, -45))
	insert(newPath, now)

	repo := apilog.NewRepository(pool)
	cutoff := now.AddDate(0, 0, -30)
	deleted, err := repo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteOlderThan: %v", err)
	}
	if deleted < 1 {
		t.Fatalf("expected at least the old row deleted, got %d", deleted)
	}

	var oldCount, newCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM api_logs WHERE path = $1`, oldPath).Scan(&oldCount); err != nil {
		t.Fatalf("count old: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM api_logs WHERE path = $1`, newPath).Scan(&newCount); err != nil {
		t.Fatalf("count new: %v", err)
	}
	if oldCount != 0 {
		t.Fatalf("expected old row pruned, still present (%d)", oldCount)
	}
	if newCount != 1 {
		t.Fatalf("expected fresh row retained, got %d", newCount)
	}

	// Cleanup the fresh row we inserted.
	_, _ = pool.Exec(ctx, `DELETE FROM api_logs WHERE path = $1`, newPath)
}
