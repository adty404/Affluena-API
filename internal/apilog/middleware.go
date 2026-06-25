package apilog

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"time"

	"affluena-api/internal/async"
	"github.com/gin-gonic/gin"
)

const maxLogPayloadSize = 32 * 1024 // 32KB

var sensitiveFields = []string{
	"password", "token", "access_token", "refresh_token",
	"authorization", "jwt", "secret", "pass",
}

type responseBodyWriter struct {
	gin.ResponseWriter
	body     *bytes.Buffer
	written  int
	limitHit bool
	skipBody bool
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	if w.skipBody {
		return w.ResponseWriter.Write(b)
	}
	remaining := maxLogPayloadSize - w.written
	if remaining <= 0 {
		if !w.limitHit && len(b) > 0 {
			w.limitHit = true
		}
		return w.ResponseWriter.Write(b)
	}
	if len(b) > remaining {
		w.body.Write(b[:remaining])
		w.written += remaining
		w.limitHit = true
		n, err := w.ResponseWriter.Write(b)
		return n, err
	}
	w.body.Write(b)
	w.written += len(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseBodyWriter) WriteString(s string) (int, error) {
	if w.skipBody {
		return w.ResponseWriter.WriteString(s)
	}
	remaining := maxLogPayloadSize - w.written
	if remaining <= 0 {
		if !w.limitHit && len(s) > 0 {
			w.limitHit = true
		}
		return w.ResponseWriter.WriteString(s)
	}
	if len(s) > remaining {
		w.body.WriteString(s[:remaining])
		w.written += remaining
		w.limitHit = true
		n, err := w.ResponseWriter.WriteString(s)
		return n, err
	}
	w.body.WriteString(s)
	w.written += len(s)
	return w.ResponseWriter.WriteString(s)
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

		// 1. Intercept Request Body for logging (read capped, pass full to handler)
		var reqBodyBytes []byte
		var reqTruncated bool
		if c.Request.Body != nil {
			buf := make([]byte, maxLogPayloadSize+1)
			n, err := io.ReadFull(c.Request.Body, buf)

			var readBytesForPassthrough []byte
			if n > maxLogPayloadSize {
				reqBodyBytes = buf[:maxLogPayloadSize]
				reqTruncated = true
			} else {
				reqBodyBytes = buf[:n]
				reqTruncated = false
			}
			readBytesForPassthrough = buf[:n]

			// io.ReadFull returns io.ErrUnexpectedEOF if n < len(buf)
			if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
				slog.Warn("error reading request body for log", "error", err)
			}

			// Reconstruct request body so handler can read it fully
			c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(readBytesForPassthrough), c.Request.Body))
		}

		// 2. Intercept Response Body (capped)
		skipResponseLogging := isExportPath(path)
		w := &responseBodyWriter{
			body:           &bytes.Buffer{},
			ResponseWriter: c.Writer,
			skipBody:       skipResponseLogging,
		}
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
			reqStr := logPayload(path, reqBodyBytes, reqTruncated, true)
			requestPayload = &reqStr
		}

		var responsePayload *string
		if !skipResponseLogging && w.body.Len() > 0 {
			respStr := logPayload(path, w.body.Bytes(), w.limitHit, false)
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
		async.SafeGo(context.Background(), "api_log_save", func(ctx context.Context) {
			ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			_ = repo.SaveLog(ctxTimeout, APILog{
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
		})
	}
}

func logPayload(path string, payload []byte, truncated, isRequest bool) string {
	// Mask entire payload for auth endpoints (both request and response may contain secrets)
	if isSensitiveAuthPath(path) {
		return `{"masked": true}`
	}

	// Add truncation marker
	if truncated {
		return `{"truncated": true, "reason": "payload exceeds log limit"}`
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
		} else if arr, ok := value.([]interface{}); ok {
			result[key] = redactSensitiveSlice(arr)
		} else {
			result[key] = value
		}
	}
	return result
}

// redactSensitiveSlice recurses into arrays so secrets nested inside a JSON
// array (e.g. a list of objects) are still redacted.
func redactSensitiveSlice(items []interface{}) []interface{} {
	result := make([]interface{}, len(items))
	for i, item := range items {
		switch v := item.(type) {
		case map[string]interface{}:
			result[i] = redactSensitiveFields(v)
		case []interface{}:
			result[i] = redactSensitiveSlice(v)
		default:
			result[i] = item
		}
	}
	return result
}

func isSensitiveAuthPath(path string) bool {
	return strings.Contains(path, "/auth/login") ||
		strings.Contains(path, "/auth/register") ||
		strings.Contains(path, "/auth/refresh") ||
		strings.Contains(path, "/auth/forgot-password") ||
		strings.Contains(path, "/auth/reset-password") ||
		strings.Contains(path, "/auth/password")
}

func isExportPath(path string) bool {
	return strings.Contains(path, "/export/")
}
