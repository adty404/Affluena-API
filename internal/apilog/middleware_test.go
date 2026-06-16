package apilog

import (
	"bytes"
	"context"
	"io"
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

func TestAPILogMiddleware_MasksAuthResponseTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockRepo{}

	router := gin.New()
	router.Use(APILogMiddleware(repo))

	router.POST("/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"tokens": gin.H{
				"access_token":  "access-token-secret",
				"refresh_token": "refresh-token-secret",
			},
		})
	})

	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"test@example.com","password":"secretpassword"}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	time.Sleep(50 * time.Millisecond)

	if repo.lastLog.ResponsePayload == nil {
		t.Fatal("expected response payload to be captured")
	}
	if *repo.lastLog.ResponsePayload != `{"masked": true}` {
		t.Fatalf("expected masked auth response payload, got %s", *repo.lastLog.ResponsePayload)
	}
}

func TestAPILogMiddleware_LargeRequestBodyPassthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockRepo{}

	router := gin.New()
	router.Use(APILogMiddleware(repo))

	largePayloadSize := 40 * 1024 // 40KB
	largeBody := bytes.Repeat([]byte("A"), largePayloadSize)

	var handlerReadSize int
	router.POST("/test-large", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		handlerReadSize = len(body)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("POST", "/test-large", bytes.NewReader(largeBody))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	time.Sleep(50 * time.Millisecond)

	if handlerReadSize != largePayloadSize {
		t.Errorf("handler read %d bytes, expected %d", handlerReadSize, largePayloadSize)
	}

	if repo.lastLog.RequestPayload == nil {
		t.Fatal("expected request payload to be logged as truncated")
	}
	expectedTruncated := `{"truncated": true, "reason": "payload exceeds log limit"}`
	if *repo.lastLog.RequestPayload != expectedTruncated {
		t.Errorf("expected %s, got %s", expectedTruncated, *repo.lastLog.RequestPayload)
	}
}

func TestAPILogMiddleware_ExportResponseSkip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockRepo{}

	router := gin.New()
	router.Use(APILogMiddleware(repo))

	largeResponseSize := 40 * 1024 // 40KB
	largeResponse := bytes.Repeat([]byte("B"), largeResponseSize)

	router.GET("/export/csv", func(c *gin.Context) {
		c.Writer.Write(largeResponse)
	})

	req, _ := http.NewRequest("GET", "/export/csv", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	time.Sleep(50 * time.Millisecond)

	if w.Body.Len() != largeResponseSize {
		t.Errorf("client received %d bytes, expected %d", w.Body.Len(), largeResponseSize)
	}

	if repo.lastLog.ResponsePayload != nil {
		t.Errorf("expected response payload to be nil (skipped) for export path, got %s", *repo.lastLog.ResponsePayload)
	}
}
