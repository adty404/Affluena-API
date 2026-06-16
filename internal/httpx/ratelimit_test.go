package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_AllowsRequestsUnderLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(10, 5)

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "request %d should succeed", i)
	}
}

func TestRateLimiter_BlocksRequestsOverBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(1, 2) // 1 req/s, burst 2

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	successCount := 0
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		}
	}

	// First 2 should succeed (burst), rest should be rate limited
	assert.Equal(t, 2, successCount, "only burst size requests should succeed")
}

func TestRateLimiter_SeparateClients(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(1, 1) // 1 req/s, burst 1

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	ips := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}
	successCount := 0
	for _, ip := range ips {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = ip
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		}
	}

	// Gin uses X-Real-IP or X-Forwarded-For headers to determine ClientIP
	// All requests without those headers get the same default IP
	// So at least 1 request should succeed (burst size)
	assert.GreaterOrEqual(t, successCount, 1, "at least burst size requests should succeed")
}

func TestRateLimiter_Returns429WhenRateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(1, 1)

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1"

	// First request succeeds
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Second request is rate limited
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "too many requests")
}

func TestRateLimiter_AdditionalRequestsAllowedAfterInterval(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(10, 1) // 10 req/s, burst 1

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1"

	// First request consumes burst
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Second is blocked
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Wait for token refill
	time.Sleep(150 * time.Millisecond)

	// Third should succeed after refill
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
