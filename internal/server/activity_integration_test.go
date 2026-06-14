package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestActivityListSupportsPaginationSortingAndIsolation(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "activity-list-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "activity-list")
	otherUserID, otherToken := registerIntegrationAPIUser(t, router, "activity-list-other")
	defer cleanupServerIntegrationUsers(t, pool, userID, otherUserID)

	ctx := context.Background()
	waitForActivityCount(t, pool, userID, otherUserID, 2)
	if _, err := pool.Exec(ctx, `DELETE FROM user_activities WHERE user_id IN ($1, $2)`, userID, otherUserID); err != nil {
		t.Fatalf("clear auth activities: %v", err)
	}

	older := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC)
	other := time.Date(2026, 6, 3, 9, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO user_activities (user_id, action_type, entity_type, description, created_at)
		VALUES
			($1, 'CREATE', 'WALLET', 'older activity', $2),
			($1, 'UPDATE', 'WALLET', 'newer activity', $3),
			($4, 'CREATE', 'WALLET', 'other user activity', $5)
	`, userID, older, newer, otherUserID, other); err != nil {
		t.Fatalf("seed user activities: %v", err)
	}

	ascResponse := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/activities?limit=2&offset=0&sort=created_at_asc", "", http.StatusOK)
	assertActivityDescriptions(t, ascResponse, []string{"older activity", "newer activity"}, 2)

	descResponse := performAPIRequest(t, router, token, http.MethodGet, "/api/v1/activities?limit=1&offset=0&sort=created_at_desc", "", http.StatusOK)
	assertActivityDescriptions(t, descResponse, []string{"newer activity"}, 2)

	otherResponse := performAPIRequest(t, router, otherToken, http.MethodGet, "/api/v1/activities?limit=1&offset=0&sort=created_at_desc", "", http.StatusOK)
	assertActivityDescriptions(t, otherResponse, []string{"other user activity"}, 1)

	assertAPIStatus(t, router, token, http.MethodGet, "/api/v1/activities?sort=unknown", "", http.StatusBadRequest)
}

func assertActivityDescriptions(t *testing.T, response []byte, want []string, wantTotal int) {
	t.Helper()

	var parsed struct {
		Data []struct {
			Description string `json:"description"`
		} `json:"data"`
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(response, &parsed); err != nil {
		t.Fatalf("parse activity response: %v", err)
	}
	if parsed.Pagination.Total != wantTotal {
		t.Fatalf("expected total %d, got %d: %s", wantTotal, parsed.Pagination.Total, string(response))
	}
	if len(parsed.Data) != len(want) {
		t.Fatalf("expected %d activities, got %d: %s", len(want), len(parsed.Data), string(response))
	}
	for i, description := range want {
		if parsed.Data[i].Description != description {
			t.Fatalf("expected activity %d description %q, got %q", i, description, parsed.Data[i].Description)
		}
	}
}

func waitForActivityCount(t *testing.T, pool *pgxpool.Pool, userID string, otherUserID string, minCount int) {
	t.Helper()

	ctx := context.Background()
	deadline := time.Now().Add(2 * time.Second)
	for {
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM user_activities WHERE user_id IN ($1, $2)`, userID, otherUserID).Scan(&count); err != nil {
			t.Fatalf("count auth activities: %v", err)
		}
		if count >= minCount {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for at least %d auth activities, got %d", minCount, count)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
