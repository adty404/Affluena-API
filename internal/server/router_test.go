package server

import (
	"testing"
	"time"

	"github.com/gin-contrib/cors"
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
