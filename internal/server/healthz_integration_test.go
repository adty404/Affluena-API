package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"affluena-api/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func healthzTestConfig() config.Config {
	return config.Config{
		Env:                  "production",
		JWTSecret:            "healthz-integration-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}
}

func performHealthz(t *testing.T, router http.Handler) (int, map[string]string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse healthz response %q: %v", recorder.Body.String(), err)
	}
	return recorder.Code, body
}

func TestHealthzReportsOKWhenDatabaseIsReachable(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(healthzTestConfig(), pool)

	status, body := performHealthz(t, router)
	if status != http.StatusOK {
		t.Fatalf("expected healthz status 200, got %d: %+v", status, body)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected healthz body status ok, got %+v", body)
	}
}

// TestHealthzReportsDegradedWhenDatabaseIsUnreachable proves the deploy
// workflow's `curl -fsS /healthz` gate fails when Postgres is down: the router
// is built over a pool pointing at an address nothing listens on, so the
// SELECT 1 ping cannot succeed and healthz must answer 503.
func TestHealthzReportsDegradedWhenDatabaseIsUnreachable(t *testing.T) {
	// Port 1 on loopback is never a Postgres: connections are refused
	// immediately, well inside the handler's 2s ping timeout. pgxpool connects
	// lazily, so constructing the pool itself succeeds.
	deadPool, err := pgxpool.New(context.Background(), "postgres://healthz:healthz@127.0.0.1:1/healthz?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatalf("build dead pool: %v", err)
	}
	t.Cleanup(deadPool.Close)

	router := NewRouter(healthzTestConfig(), deadPool)

	status, body := performHealthz(t, router)
	if status != http.StatusServiceUnavailable {
		t.Fatalf("expected healthz status 503, got %d: %+v", status, body)
	}
	if body["status"] != "degraded" {
		t.Fatalf("expected healthz body status degraded, got %+v", body)
	}
	// Terse marker only — the body must never leak connection details.
	if body["db"] != "unreachable" {
		t.Fatalf("expected terse db field, got %+v", body)
	}
}
