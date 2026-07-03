package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"affluena-api/internal/config"
	"affluena-api/internal/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationIntegration(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "integration-test-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	// Register user A
	_, userAToken := registerIntegrationAPIUser(t, router, "notif-user-a")

	// GET /notifications/rules for user A
	req, _ := http.NewRequest("GET", "/api/v1/notifications/rules", nil)
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var listResp struct {
		Rules []notification.NotificationRule `json:"rules"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &listResp)
	require.NoError(t, err)
	require.Len(t, listResp.Rules, 5)

	// Find a rule to update
	var ruleToUpdate notification.NotificationRule
	for _, r := range listResp.Rules {
		if r.RuleKey == "budget-alert" {
			ruleToUpdate = r
			break
		}
	}
	require.NotEmpty(t, ruleToUpdate.ID)
	require.True(t, ruleToUpdate.Enabled)

	// PUT /notifications/rules/:id to toggle enabled
	enabled := false
	updateBody := map[string]interface{}{
		"enabled": enabled,
	}
	bodyBytes, _ := json.Marshal(updateBody)
	req, _ = http.NewRequest("PUT", "/api/v1/notifications/rules/"+ruleToUpdate.ID, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+userAToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var updatedRule notification.NotificationRule
	err = json.Unmarshal(w.Body.Bytes(), &updatedRule)
	require.NoError(t, err)
	assert.False(t, updatedRule.Enabled)

	// GET again to assert toggled
	req, _ = http.NewRequest("GET", "/api/v1/notifications/rules", nil)
	req.Header.Set("Authorization", "Bearer "+userAToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &listResp)
	require.NoError(t, err)
	for _, r := range listResp.Rules {
		if r.ID == ruleToUpdate.ID {
			assert.False(t, r.Enabled)
		}
	}

	// Register user B
	_, userBToken := registerIntegrationAPIUser(t, router, "notif-user-b")

	// GET /notifications/rules for user B
	req, _ = http.NewRequest("GET", "/api/v1/notifications/rules", nil)
	req.Header.Set("Authorization", "Bearer "+userBToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var listRespB struct {
		Rules []notification.NotificationRule `json:"rules"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &listRespB)
	require.NoError(t, err)
	require.Len(t, listRespB.Rules, 5)

	// Assert isolation: user B's budget-alert is still enabled
	for _, r := range listRespB.Rules {
		if r.RuleKey == "budget-alert" {
			assert.True(t, r.Enabled)
		}
	}

	// User B tries to update User A's rule
	req, _ = http.NewRequest("PUT", "/api/v1/notifications/rules/"+ruleToUpdate.ID, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+userBToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

// TestNotificationRulesLocalizedToIndonesian asserts the settings list returns
// Indonesian title/description (keyed on rule_key) even though the DB seeds
// English copy.
func TestNotificationRulesLocalizedToIndonesian(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "test",
		JWTSecret:            "integration-test-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "notif-id-copy")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	req, _ := http.NewRequest("GET", "/api/v1/notifications/rules", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Rules []notification.NotificationRule `json:"rules"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Rules, 5)

	want := map[string]string{
		"budget-alert":   "Peringatan anggaran",
		"due-reminder":   "Pengingat jatuh tempo",
		"recurring-run":  "Hasil transaksi berulang",
		"security-alert": "Peringatan keamanan",
		"weekly-summary": "Ringkasan keuangan mingguan",
	}
	for _, r := range resp.Rules {
		if wantTitle, ok := want[r.RuleKey]; ok {
			assert.Equal(t, wantTitle, r.Title, "rule %s title should be Indonesian", r.RuleKey)
		}
	}
}

// TestNotificationDeliveryGatingAndDedup exercises the real gating + de-dupe SQL:
// a disabled rule yields no send, an enabled rule delivers once, and a repeat
// with the same dedupe_key is suppressed.
func TestNotificationDeliveryGatingAndDedup(t *testing.T) {
	pool := openServerIntegrationPool(t)
	ctx := context.Background()
	router := NewRouter(config.Config{
		Env:                  "test",
		JWTSecret:            "integration-test-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	// Registering + listing seeds the default rules (due-reminder enabled=true).
	userID, token := registerIntegrationAPIUser(t, router, "notif-deliver")
	defer cleanupServerIntegrationUsers(t, pool, userID)
	listReq, _ := http.NewRequest("GET", "/api/v1/notifications/rules", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(httptest.NewRecorder(), listReq)

	repo := notification.NewDeliveryRepository(pool)
	notifier := notification.NewNotifier(repo, nil) // no mailer: in-app only path

	notif := notification.Notification{
		RuleKey:   "due-reminder",
		DedupeKey: "subscription:test:H-3:2026-07-10",
		Subject:   "s", Title: "Langganan jatuh tempo", Message: "m",
	}

	// due-reminder is enabled by default → first send records a delivery.
	sent, err := notifier.Send(ctx, userID, notif)
	require.NoError(t, err)
	require.True(t, sent, "first send should record a delivery")

	// Repeat with the same dedupe_key → suppressed.
	sent, err = notifier.Send(ctx, userID, notif)
	require.NoError(t, err)
	require.False(t, sent, "repeat send should be de-duped")

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM notification_deliveries WHERE user_id = $1 AND rule_key = 'due-reminder'`, userID,
	).Scan(&count))
	require.Equal(t, 1, count, "exactly one delivery row expected")

	// Disable the rule → subsequent send with a NEW dedupe key must not deliver.
	_, err = pool.Exec(ctx, `UPDATE notification_rules SET enabled = false WHERE user_id = $1 AND rule_key = 'due-reminder'`, userID)
	require.NoError(t, err)

	notif2 := notif
	notif2.DedupeKey = "subscription:test:H-1:2026-07-12"
	sent, err = notifier.Send(ctx, userID, notif2)
	require.NoError(t, err)
	require.False(t, sent, "disabled rule must not deliver")
}
