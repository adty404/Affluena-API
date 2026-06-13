package apilog

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// APILogMiddleware provides structured logging to stdout using slog
// and saves the HTTP request logs into the database asynchronously.
func APILogMiddleware(repo Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		// Do not log health checks to avoid spamming the logs
		if path == "/healthz" {
			c.Next()
			return
		}

		// 1. Intercept Request Body
		var reqBodyBytes []byte
		if c.Request.Body != nil {
			reqBodyBytes, _ = io.ReadAll(c.Request.Body)
			// Restore the io.ReadCloser to its original state
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
		}

		// 2. Intercept Response Body
		w := &responseBodyWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = w

		// Process request
		c.Next()

		end := time.Now()
		latency := end.Sub(start)
		latencyMs := int(latency.Milliseconds())

		if rawQuery != "" {
			path = path + "?" + rawQuery
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

		// 3. Process Payloads & Mask Sensitive Auth Data
		var requestPayload *string
		if len(reqBodyBytes) > 0 {
			reqStr := logPayload(path, reqBodyBytes)
			requestPayload = &reqStr
		}

		var responsePayload *string
		if w.body.Len() > 0 {
			respStr := logPayload(path, w.body.Bytes())
			responsePayload = &respStr
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
			Method:          method,
			Path:            path,
			StatusCode:      statusCode,
			LatencyMs:       latencyMs,
			ClientIP:        clientIP,
			UserAgent:       userAgent,
			UserID:          userID,
			RequestPayload:  requestPayload,
			ResponsePayload: responsePayload,
		})
	}
}

func logPayload(path string, payload []byte) string {
	if isSensitiveAuthPath(path) {
		return `{"masked": true}`
	}
	payloadStr := string(payload)
	if json.Valid(payload) {
		var compacted bytes.Buffer
		if err := json.Compact(&compacted, payload); err == nil {
			payloadStr = compacted.String()
		}
	}
	return payloadStr
}

func isSensitiveAuthPath(path string) bool {
	return strings.Contains(path, "/auth/login") ||
		strings.Contains(path, "/auth/register") ||
		strings.Contains(path, "/auth/refresh")
}
