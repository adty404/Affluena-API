package server

import (
	"bytes"
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
