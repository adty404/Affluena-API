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
	saved chan APILog
}

func newMockRepo() *mockRepo {
	return &mockRepo{saved: make(chan APILog, 1)}
}

func (m *mockRepo) SaveLog(ctx context.Context, logEntry APILog) error {
	m.saved <- logEntry
	return nil
}

func (m *mockRepo) waitLog(t *testing.T) APILog {
	t.Helper()

	select {
	case logEntry := <-m.saved:
		return logEntry
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for api log")
		return APILog{}
	}
}

func TestAPILogMiddleware_CapturesPayloads(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newMockRepo()

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

	logEntry := repo.waitLog(t)

	if logEntry.RequestPayload == nil || *logEntry.RequestPayload != `{"key":"value"}` {
		t.Errorf("expected request payload `{\"key\":\"value\"}`, got %v", logEntry.RequestPayload)
	}

	if logEntry.ResponsePayload == nil || *logEntry.ResponsePayload != `{"status":"ok"}` {
		t.Errorf("expected response payload `{\"status\":\"ok\"}`, got %v", logEntry.ResponsePayload)
	}
}

func TestAPILogMiddleware_MasksPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newMockRepo()

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

	logEntry := repo.waitLog(t)

	if logEntry.RequestPayload == nil || *logEntry.RequestPayload != `{"masked": true}` {
		t.Errorf("expected masked request payload, got %v", logEntry.RequestPayload)
	}
}

func TestAPILogMiddleware_MasksAuthResponseTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newMockRepo()

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

	logEntry := repo.waitLog(t)

	if logEntry.ResponsePayload == nil {
		t.Fatal("expected response payload to be captured")
	}
	if *logEntry.ResponsePayload != `{"masked": true}` {
		t.Fatalf("expected masked auth response payload, got %s", *logEntry.ResponsePayload)
	}
}

func TestAPILogMiddleware_LargeRequestBodyPassthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newMockRepo()

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

	logEntry := repo.waitLog(t)

	if handlerReadSize != largePayloadSize {
		t.Errorf("handler read %d bytes, expected %d", handlerReadSize, largePayloadSize)
	}

	if logEntry.RequestPayload == nil {
		t.Fatal("expected request payload to be logged as truncated")
	}
	expectedTruncated := `{"truncated": true, "reason": "payload exceeds log limit"}`
	if *logEntry.RequestPayload != expectedTruncated {
		t.Errorf("expected %s, got %s", expectedTruncated, *logEntry.RequestPayload)
	}
}

func TestAPILogMiddleware_ExportResponseSkip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newMockRepo()

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

	logEntry := repo.waitLog(t)

	if w.Body.Len() != largeResponseSize {
		t.Errorf("client received %d bytes, expected %d", w.Body.Len(), largeResponseSize)
	}

	if logEntry.ResponsePayload != nil {
		t.Errorf("expected response payload to be nil (skipped) for export path, got %s", *logEntry.ResponsePayload)
	}
}

func TestAPILogMiddleware_ExactBoundaryRequestBodyPassthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newMockRepo()

	router := gin.New()
	router.Use(APILogMiddleware(repo))

	exactPayloadSize := 32 * 1024 // 32KB
	exactBody := bytes.Repeat([]byte("A"), exactPayloadSize)

	var handlerReadSize int
	router.POST("/test-exact", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		handlerReadSize = len(body)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("POST", "/test-exact", bytes.NewReader(exactBody))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	logEntry := repo.waitLog(t)

	if handlerReadSize != exactPayloadSize {
		t.Errorf("handler read %d bytes, expected %d", handlerReadSize, exactPayloadSize)
	}

	if logEntry.RequestPayload == nil {
		t.Fatal("expected request payload to be logged")
	}
	// It should NOT be truncated, so it should equal the exact body string
	expectedPayload := string(exactBody)
	if *logEntry.RequestPayload != expectedPayload {
		t.Errorf("expected exact payload logged, but it was truncated or altered")
	}
}

func (m *mockRepo) GetLogByID(ctx context.Context, userID string, id string) (*APILog, error) {
	return nil, nil
}

func (m *mockRepo) ListLogs(ctx context.Context, userID string, limit int) ([]APILog, error) {
	return nil, nil
}
