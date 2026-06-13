package apilog

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// APILogMiddleware provides structured logging to stdout using slog
// and saves the HTTP request logs into the database asynchronously.
func APILogMiddleware(repo Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Do not log health checks to avoid spamming the logs
		if path == "/healthz" {
			return
		}

		end := time.Now()
		latency := end.Sub(start)
		latencyMs := int(latency.Milliseconds())

		if raw != "" {
			path = path + "?" + raw
		}

		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		userAgent := c.Request.UserAgent()

		var userID *string
		if uid, exists := c.Get("user_id"); exists {
			if uidStr, ok := uid.(string); ok && uidStr != "" {
				userID = &uidStr
			}
		}

		// 1. Structured Logging (Stdout)
		slog.Info("HTTP Request",
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", statusCode),
			slog.Int("latency_ms", latencyMs),
			slog.String("ip", clientIP),
			slog.Any("user_id", userID),
		)

		// 2. Database Save (Async)
		go func(entry APILog) {
			// Using background context because the request context is canceled when response is sent
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_ = repo.SaveLog(ctx, entry)
		}(APILog{
			Method:     method,
			Path:       path,
			StatusCode: statusCode,
			LatencyMs:  latencyMs,
			ClientIP:   clientIP,
			UserAgent:  userAgent,
			UserID:     userID,
		})
	}
}
