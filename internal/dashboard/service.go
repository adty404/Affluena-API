package dashboard

import (
	"errors"
	"time"
)

func ParseMonth(value string) (time.Time, error) {
	if value == "" {
		now := time.Now().UTC()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), nil
	}

	parsed, err := time.Parse("2006-01", value)
	if err != nil {
		return time.Time{}, errors.New("month must use YYYY-MM format")
	}
	return time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func newBudgetSummary(limitMinor int64, spentMinor int64) BudgetSummary {
	usagePercent := float64(0)
	if limitMinor > 0 {
		usagePercent = float64(spentMinor) / float64(limitMinor) * 100
	}
	return BudgetSummary{
		LimitMinor:     limitMinor,
		SpentMinor:     spentMinor,
		RemainingMinor: limitMinor - spentMinor,
		UsagePercent:   usagePercent,
	}
}
