package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := allowedCORSOrigins(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
