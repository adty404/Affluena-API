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

const maxLogPayloadSize = 32 * 1024 // 32KB

var sensitiveFields = []string{
	"password", "token", "access_token", "refresh_token",
	"authorization", "jwt", "secret", "pass",
}

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

		// 1. Intercept Request Body (with limit)
		var reqBodyBytes []byte
		if c.Request.Body != nil {
			reqBodyBytes, _ = io.ReadAll(io.LimitReader(c.Request.Body, maxLogPayloadSize+1))
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

		// 3. Process Payloads & Mask Sensitive Data
		var requestPayload *string
		if len(reqBodyBytes) > 0 {
			reqStr := logPayload(path, reqBodyBytes, true)
			requestPayload = &reqStr
		}

		// Skip response logging for export/download endpoints
		skipResponseLogging := isExportPath(path)
		var responsePayload *string
		if !skipResponseLogging && w.body.Len() > 0 {
			respStr := logPayload(path, w.body.Bytes(), false)
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

func logPayload(path string, payload []byte, isRequest bool) string {
	// Mask entire payload for auth endpoints (both request and response may contain secrets)
	if isSensitiveAuthPath(path) {
		return `{"masked": true}`
	}

	// Truncate if exceeds max size
	if len(payload) > maxLogPayloadSize {
		payload = payload[:maxLogPayloadSize]
	}

	payloadStr := string(payload)
	if json.Valid(payload) {
		var data map[string]interface{}
		if err := json.Unmarshal(payload, &data); err == nil {
			data = redactSensitiveFields(data)
			if compacted, err := json.Marshal(data); err == nil {
				payloadStr = string(compacted)
			}
		} else {
			// If parsing fails, at least compact the original
			var compacted bytes.Buffer
			if err := json.Compact(&compacted, payload); err == nil {
				payloadStr = compacted.String()
			}
		}
	}
	return payloadStr
}

func redactSensitiveFields(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range data {
		lowerKey := strings.ToLower(key)
		isSensitive := false
		for _, sensitive := range sensitiveFields {
			if strings.Contains(lowerKey, sensitive) {
				isSensitive = true
				break
			}
		}
		if isSensitive {
			result[key] = "***REDACTED***"
		} else if nested, ok := value.(map[string]interface{}); ok {
			result[key] = redactSensitiveFields(nested)
		} else {
			result[key] = value
		}
	}
	return result
}

func isSensitiveAuthPath(path string) bool {
	return strings.Contains(path, "/auth/login") ||
		strings.Contains(path, "/auth/register") ||
		strings.Contains(path, "/auth/refresh")
}

func isExportPath(path string) bool {
	return strings.Contains(path, "/export/")
}
