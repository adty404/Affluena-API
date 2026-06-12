package budget

import (
	"testing"
	"time"
)

func TestParseBudgetMonthUsesFirstDayUTC(t *testing.T) {
	month, err := ParseBudgetMonth("2026-06")
	if err != nil {
		t.Fatalf("ParseBudgetMonth returned error: %v", err)
	}

	expected := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if !month.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected, month)
	}
}

func TestUsageSummaryCalculatesRemainingAndPercent(t *testing.T) {
	summary := NewUsageSummary(200_000, 75_000)

	if summary.RemainingMinor != 125_000 {
		t.Fatalf("expected remaining 125000, got %d", summary.RemainingMinor)
	}
	if summary.UsagePercent != 37.5 {
		t.Fatalf("expected usage percent 37.5, got %f", summary.UsagePercent)
	}
}

func TestUsageSummaryAllowsOverspend(t *testing.T) {
	summary := NewUsageSummary(100_000, 125_000)

	if summary.RemainingMinor != -25_000 {
		t.Fatalf("expected remaining -25000, got %d", summary.RemainingMinor)
	}
	if summary.UsagePercent != 125 {
		t.Fatalf("expected usage percent 125, got %f", summary.UsagePercent)
	}
}
