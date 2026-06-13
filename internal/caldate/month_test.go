package caldate

import (
	"testing"
	"time"
)

func TestAddMonthsClampedPreservesTimeAndLocation(t *testing.T) {
	loc := time.FixedZone("WIB", 7*60*60)
	current := time.Date(2026, 1, 31, 23, 59, 58, 123, loc)

	next := AddMonthsClamped(current, 1)

	expected := time.Date(2026, 2, 28, 23, 59, 58, 123, loc)
	if !next.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected, next)
	}
	if next.Location() != loc {
		t.Fatalf("expected location to be preserved")
	}
}

func TestAddMonthsClampedHandlesLeapYearAndBackwardMonth(t *testing.T) {
	leap := AddMonthsClamped(time.Date(2028, 1, 31, 0, 0, 0, 0, time.UTC), 1)
	if !leap.Equal(time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected leap-year February 29, got %s", leap)
	}

	backward := AddMonthsClamped(time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC), -1)
	if !backward.Equal(time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected backward clamp to February 28, got %s", backward)
	}
}
