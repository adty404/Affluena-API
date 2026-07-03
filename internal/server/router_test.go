package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"affluena-api/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowedCORSOrigins(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string returns default",
			input:    "",
			expected: []string{"http://localhost:5173"},
		},
		{
			name:     "whitespace only returns default",
			input:    "   ",
			expected: []string{"http://localhost:5173"},
		},
		{
			name:     "single origin",
			input:    "http://example.com",
			expected: []string{"http://example.com"},
		},
		{
			name:     "multiple origins",
			input:    "http://localhost:5173,https://example.com,http://test.com",
			expected: []string{"http://localhost:5173", "https://example.com", "http://test.com"},
		},
		{
			name:     "trims whitespace",
			input:    " http://localhost:5173 , https://example.com ",
			expected: []string{"http://localhost:5173", "https://example.com"},
		},
		{
			name:     "skips empty entries",
			input:    "http://localhost:5173,,https://example.com",
			expected: []string{"http://localhost:5173", "https://example.com"},
		},
		{
			name:     "all empty entries returns default",
			input:    ",,",
			expected: []string{"http://localhost:5173"},
		},
		{
			name:     "wildcard is dropped (credentials enabled)",
			input:    "*",
			expected: []string{"http://localhost:5173"},
		},
		{
			name:     "scheme-less origin is dropped",
			input:    "example.com",
			expected: []string{"http://localhost:5173"},
		},
		{
			name:     "bare host/IP is dropped, schemed origin kept",
			input:    "1.2.3.4,https://example.com",
			expected: []string{"https://example.com"},
		},
		{
			name:     "ws/wss schemes are allowed",
			input:    "ws://localhost:5173,wss://example.com",
			expected: []string{"ws://localhost:5173", "wss://example.com"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := allowedCORSOrigins(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestApplyTrustedProxies_ClientIPHonoursForwardedForOnlyFromTrustedHop verifies
// that the trusted-proxy list drives ClientIP(): a forged X-Forwarded-For from an
// UNtrusted remote address is ignored (so the real remote addr is used for the
// per-IP AuthLimiter), while X-Forwarded-For from the trusted nginx hop is
// honoured (so the real end-user IP is recovered behind the proxy).
func TestApplyTrustedProxies_ClientIPHonoursForwardedForOnlyFromTrustedHop(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newEngine := func(proxies []string) *gin.Engine {
		e := gin.New()
		applyTrustedProxies(e, proxies)
		e.GET("/whoami", func(c *gin.Context) {
			c.String(http.StatusOK, c.ClientIP())
		})
		return e
	}

	// Default private/loopback ranges from config.
	trusted := config.Load().TrustedProxies

	tests := []struct {
		name       string
		proxies    []string
		remoteAddr string
		forwarded  string
		wantIP     string
	}{
		{
			name:       "untrusted client cannot spoof via X-Forwarded-For",
			proxies:    trusted,
			remoteAddr: "203.0.113.10:5555", // public, NOT in trusted ranges
			forwarded:  "1.2.3.4",
			wantIP:     "203.0.113.10", // forged header ignored, real remote used
		},
		{
			name:       "trusted nginx hop forwards the real end-user IP",
			proxies:    trusted,
			remoteAddr: "10.0.0.5:5555", // private, in trusted ranges (nginx)
			forwarded:  "1.2.3.4",
			wantIP:     "1.2.3.4", // header honoured, real user recovered
		},
		{
			name:       "malformed list fails closed: no proxy trusted",
			proxies:    []string{"not-a-cidr"},
			remoteAddr: "10.0.0.5:5555",
			forwarded:  "1.2.3.4",
			wantIP:     "10.0.0.5", // header ignored because trust reset to nil
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := newEngine(tc.proxies)
			req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
			req.RemoteAddr = tc.remoteAddr
			req.Header.Set("X-Forwarded-For", tc.forwarded)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			assert.Equal(t, tc.wantIP, rec.Body.String())
		})
	}
}

// TestAllowedCORSOrigins_NeverPanicsCorsNew guards against the startup boot-crash
// where gin-contrib/cors panics on an origin without a scheme: the output of
// allowedCORSOrigins must always be safe to pass to cors.New.
func TestAllowedCORSOrigins_NeverPanicsCorsNew(t *testing.T) {
	inputs := []string{"", "*", "example.com", "1.2.3.4", "http://VPS_PUBLIC_IP", "bad,,*,host-only", "https://app.example.com"}
	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			require.NotPanics(t, func() {
				cors.New(cors.Config{
					AllowOrigins:     allowedCORSOrigins(in),
					AllowMethods:     []string{"GET"},
					AllowCredentials: true,
					MaxAge:           12 * time.Hour,
				})
			})
		})
	}
}
