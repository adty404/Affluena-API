package apilog

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type mockRepo struct {
	lastLog APILog
}

func (m *mockRepo) SaveLog(ctx context.Context, logEntry APILog) error {
	m.lastLog = logEntry
	return nil
}

func TestAPILogMiddleware_CapturesPayloads(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockRepo{}

	router := gin.New()
	router.Use(APILogMiddleware(repo))

	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	reqBody := `{"key":"value"}`
	req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Wait for async goroutine to finish
	time.Sleep(50 * time.Millisecond)

	if repo.lastLog.RequestPayload == nil || *repo.lastLog.RequestPayload != `{"key":"value"}` {
		t.Errorf("expected request payload `{\"key\":\"value\"}`, got %v", repo.lastLog.RequestPayload)
	}

	if repo.lastLog.ResponsePayload == nil || *repo.lastLog.ResponsePayload != `{"status":"ok"}` {
		t.Errorf("expected response payload `{\"status\":\"ok\"}`, got %v", repo.lastLog.ResponsePayload)
	}
}

func TestAPILogMiddleware_MasksPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockRepo{}

	router := gin.New()
	router.Use(APILogMiddleware(repo))

	router.POST("/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	reqBody := `{"email":"test@example.com","password":"secretpassword"}`
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	time.Sleep(50 * time.Millisecond)

	if repo.lastLog.RequestPayload == nil || *repo.lastLog.RequestPayload != `{"masked": true}` {
		t.Errorf("expected masked request payload, got %v", repo.lastLog.RequestPayload)
	}
}
