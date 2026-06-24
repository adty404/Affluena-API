package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
)

// TestGoalStatusTransition proves PUT /goals/:id can transition the goal
// lifecycle status (active -> achieved -> cancelled), that omitting status
// preserves the current value (COALESCE), and that an invalid status is
// rejected with 400.
func TestGoalStatusTransition(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "goal-status-integration-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "goalstatus")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	goalID := createAPIResource(t, router, token, "/api/v1/goals", `{
		"name": "Emergency Fund",
		"target_amount_minor": 5000000,
		"deadline": "2026-12-31T00:00:00Z"
	}`)

	readStatus := func() string {
		body := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/goals/"+goalID, "", http.StatusOK)
		var g struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(body, &g); err != nil {
			t.Fatalf("unmarshal goal: %v", err)
		}
		return g.Status
	}

	if got := readStatus(); got != "active" {
		t.Fatalf("new goal status = %q, want active", got)
	}

	// Transition to achieved.
	performAPIRequest(t, router, token, http.MethodPut, "/api/v1/goals/"+goalID, `{
		"name": "Emergency Fund",
		"target_amount_minor": 5000000,
		"deadline": "2026-12-31T00:00:00Z",
		"status": "achieved"
	}`, http.StatusOK)
	if got := readStatus(); got != "achieved" {
		t.Fatalf("after achieved transition status = %q, want achieved", got)
	}

	// Update without a status field must preserve the current status.
	performAPIRequest(t, router, token, http.MethodPut, "/api/v1/goals/"+goalID, `{
		"name": "Emergency Fund Renamed",
		"target_amount_minor": 7500000,
		"deadline": "2026-12-31T00:00:00Z"
	}`, http.StatusOK)
	if got := readStatus(); got != "achieved" {
		t.Fatalf("status after status-less update = %q, want achieved (preserved)", got)
	}

	// Transition to cancelled.
	performAPIRequest(t, router, token, http.MethodPut, "/api/v1/goals/"+goalID, `{
		"name": "Emergency Fund Renamed",
		"target_amount_minor": 7500000,
		"deadline": "2026-12-31T00:00:00Z",
		"status": "cancelled"
	}`, http.StatusOK)
	if got := readStatus(); got != "cancelled" {
		t.Fatalf("after cancelled transition status = %q, want cancelled", got)
	}

	// Invalid status is rejected.
	assertAPIStatus(t, router, token, http.MethodPut, "/api/v1/goals/"+goalID, `{
		"name": "Emergency Fund Renamed",
		"target_amount_minor": 7500000,
		"deadline": "2026-12-31T00:00:00Z",
		"status": "bogus"
	}`, http.StatusBadRequest)
}
