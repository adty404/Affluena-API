package httpx

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter configures rate limiting per IP.
type RateLimiter struct {
	limiters sync.Map // map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter.
// rate: requests per second (e.g., 5 for 5 req/s)
// burst: maximum burst size (e.g., 10 allows 10 requests instantly)
func NewRateLimiter(r rate.Limit, burst int) *RateLimiter {
	return &RateLimiter{
		rate:  r,
		burst: burst,
	}
}

// getLimiter returns the rate limiter for the given key.
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	limiter, exists := rl.limiters.Load(key)
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters.Store(key, limiter)
	}
	return limiter.(*rate.Limiter)
}

// Middleware returns a gin middleware that rate limits by client IP.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		limiter := rl.getLimiter(key)

		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CleanupInterval defines how often to cleanup old limiters.
const CleanupInterval = 5 * time.Minute

// StartCleanup periodically removes unused limiters.
func (rl *RateLimiter) StartCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			rl.limiters.Range(func(key, value any) bool {
				limiter := value.(*rate.Limiter)
				// Remove if no tokens are being used
				if limiter.Allow() {
					rl.limiters.Delete(key)
				}
				return true
			})
		}
	}()
}

// Common rate limiters for different use cases
var (
	// AuthLimiter: 5 req/s, burst 10 - for login/register
	AuthLimiter = NewRateLimiter(5, 10)

	// APILimiter: 100 req/s, burst 200 - for general API
	APILimiter = NewRateLimiter(100, 200)
)
