package notification

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeNotificationUseCase struct {
	listErr   error
	updateErr error
}

func (f fakeNotificationUseCase) List(ctx context.Context, userID string) ([]NotificationRule, error) {
	return nil, f.listErr
}

func (f fakeNotificationUseCase) Update(ctx context.Context, userID, id string, update NotificationRuleUpdate) (NotificationRule, error) {
	return NotificationRule{}, f.updateErr
}

func TestHandler_List_sanitizes_internal_error_response(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(fakeNotificationUseCase{listErr: errors.New("database password leaked")})
	router := gin.New()
	router.GET("/notifications/rules", func(c *gin.Context) {
		c.Set("user_id", "user-1")
		handler.List(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/notifications/rules", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Equal(t, "internal server error", decodeErrorMessage(t, w))
	require.NotContains(t, w.Body.String(), "database password leaked")
}

func TestHandler_Update_returns_generic_bad_request_when_binding_fails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(fakeNotificationUseCase{})
	router := gin.New()
	router.PUT("/notifications/rules/:id", func(c *gin.Context) {
		c.Set("user_id", "user-1")
		handler.Update(c)
	})

	req := httptest.NewRequest(http.MethodPut, "/notifications/rules/11111111-1111-1111-1111-111111111111", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, "invalid request body", decodeErrorMessage(t, w))
}

func decodeErrorMessage(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()

	var body struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body.Error
}
