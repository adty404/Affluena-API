package recurring

import (
	"testing"
	"time"
)

func TestAdvanceNextRunAtWeeklyAndMonthly(t *testing.T) {
	weekly := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	nextWeekly, err := AdvanceNextRunAt(weekly, FrequencyWeekly, 2)
	if err != nil {
		t.Fatalf("AdvanceNextRunAt weekly returned error: %v", err)
	}
	if !nextWeekly.Equal(time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected weekly next run: %s", nextWeekly)
	}

	monthly := time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC)
	nextMonthly, err := AdvanceNextRunAt(monthly, FrequencyMonthly, 1)
	if err != nil {
		t.Fatalf("AdvanceNextRunAt monthly returned error: %v", err)
	}
	if !nextMonthly.Equal(time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected Go AddDate monthly rollover to 2026-03-03, got %s", nextMonthly)
	}
}

func TestAdvancePastAdvancesUntilAfterNow(t *testing.T) {
	start := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	now := time.Date(2026, 6, 20, 9, 0, 0, 0, time.UTC)

	next, err := AdvancePast(start, FrequencyWeekly, 1, now)
	if err != nil {
		t.Fatalf("AdvancePast returned error: %v", err)
	}
	if !next.After(now) {
		t.Fatalf("expected next run after %s, got %s", now, next)
	}
	if !next.Equal(time.Date(2026, 6, 22, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected catch-up next run: %s", next)
	}
}

func TestValidationRejectsBadFrequencyStatusAndInterval(t *testing.T) {
	if IsValidFrequency(Frequency("daily")) {
		t.Fatal("expected daily to be invalid for MVP")
	}
	if IsValidStatus(Status("unknown")) {
		t.Fatal("expected unknown status to be invalid")
	}
	if _, err := AdvanceNextRunAt(time.Now().UTC(), FrequencyWeekly, 0); err == nil {
		t.Fatal("expected zero interval to fail")
	}
}

func TestNextStateAfterRunCancelsWhenNextRunPassesEndAt(t *testing.T) {
	current := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	endAt := time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC)

	nextRunAt, status, err := NextStateAfterRun(current, FrequencyWeekly, 1, &endAt, StatusActive)
	if err != nil {
		t.Fatalf("NextStateAfterRun returned error: %v", err)
	}
	if !nextRunAt.Equal(time.Date(2026, 6, 8, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected next run: %s", nextRunAt)
	}
	if status != StatusCancelled {
		t.Fatalf("expected cancelled status, got %s", status)
	}
}
