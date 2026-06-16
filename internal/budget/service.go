package budget

import (
	"errors"
	"time"
)

var ErrInvalidBudgetMonth = errors.New("invalid month format: must use YYYY-MM")

type UsageSummary struct {
	LimitMinor     int64   `json:"limit_minor"`
	SpentMinor     int64   `json:"spent_minor"`
	RemainingMinor int64   `json:"remaining_minor"`
	UsagePercent   float64 `json:"usage_percent"`
}

func ParseBudgetMonth(value string) (time.Time, error) {
	if value == "" {
		now := time.Now().UTC()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), nil
	}

	parsed, err := time.Parse("2006-01", value)
	if err != nil {
		return time.Time{}, ErrInvalidBudgetMonth
	}
	return time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func NewUsageSummary(limitMinor int64, spentMinor int64) UsageSummary {
	percent := float64(0)
	if limitMinor > 0 {
		percent = float64(spentMinor) / float64(limitMinor) * 100
	}
	return UsageSummary{
		LimitMinor:     limitMinor,
		SpentMinor:     spentMinor,
		RemainingMinor: limitMinor - spentMinor,
		UsagePercent:   percent,
	}
}
