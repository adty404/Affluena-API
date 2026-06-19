package report

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateChangePercent(t *testing.T) {
	tests := []struct {
		name     string
		current  int64
		previous int64
		expected float64
	}{
		{"both zero", 0, 0, 0},
		{"previous zero", 100, 0, 0},
		{"current zero", 0, 100, -100},
		{"increase", 150, 100, 50},
		{"decrease", 50, 100, -50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateChangePercent(tt.current, tt.previous)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseMonth(t *testing.T) {
	t.Run("empty month uses current month", func(t *testing.T) {
		currStart, currEnd, prevStart, prevEnd, err := parseMonth("")
		assert.NoError(t, err)

		now := time.Now()
		expectedStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expectedStart, currStart)
		assert.Equal(t, expectedStart.AddDate(0, 1, 0), currEnd)
		assert.Equal(t, expectedStart.AddDate(0, -1, 0), prevStart)
		assert.Equal(t, expectedStart, prevEnd)
	})

	t.Run("valid month", func(t *testing.T) {
		currStart, currEnd, prevStart, prevEnd, err := parseMonth("2023-05")
		assert.NoError(t, err)

		expectedStart := time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expectedStart, currStart)
		assert.Equal(t, expectedStart.AddDate(0, 1, 0), currEnd)
		assert.Equal(t, expectedStart.AddDate(0, -1, 0), prevStart)
		assert.Equal(t, expectedStart, prevEnd)
	})

	t.Run("invalid month", func(t *testing.T) {
		_, _, _, _, err := parseMonth("invalid")
		assert.Error(t, err)
	})
}
