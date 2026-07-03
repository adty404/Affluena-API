package apilog

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakePruner struct {
	gotCutoff time.Time
	calls     int
	deleted   int64
	err       error
}

func (f *fakePruner) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	f.calls++
	f.gotCutoff = cutoff
	return f.deleted, f.err
}

func TestRetentionSchedulerRunPrunesWithConfiguredCutoff(t *testing.T) {
	fp := &fakePruner{deleted: 3}
	s := NewRetentionScheduler(fp, time.Hour, 30)

	before := time.Now().UTC().AddDate(0, 0, -30)
	s.run(context.Background())
	after := time.Now().UTC().AddDate(0, 0, -30)

	if fp.calls != 1 {
		t.Fatalf("expected one prune call, got %d", fp.calls)
	}
	// The cutoff must be ~30 days in the past (bounded by the wall-clock window
	// around the call), so rows older than the retention window are the ones
	// deleted.
	if fp.gotCutoff.Before(before.Add(-time.Second)) || fp.gotCutoff.After(after.Add(time.Second)) {
		t.Fatalf("cutoff %v outside expected 30-day window [%v, %v]", fp.gotCutoff, before, after)
	}
}

func TestRetentionSchedulerRunSwallowsError(t *testing.T) {
	fp := &fakePruner{err: errors.New("db down")}
	s := NewRetentionScheduler(fp, time.Hour, 30)
	// Must not panic and must not propagate (background job); just logs.
	s.run(context.Background())
	if fp.calls != 1 {
		t.Fatalf("expected one prune attempt, got %d", fp.calls)
	}
}

func TestNewRetentionSchedulerAppliesDefaults(t *testing.T) {
	s := NewRetentionScheduler(&fakePruner{}, 0, 0)
	if s.interval != 6*time.Hour {
		t.Fatalf("expected default interval 6h, got %s", s.interval)
	}
	if s.retentionDays != 30 {
		t.Fatalf("expected default retention 30 days, got %d", s.retentionDays)
	}
}

func TestRetentionSchedulerStartRunsAndStops(t *testing.T) {
	fp := &fakePruner{}
	// Short interval so the ticker fires within the test.
	s := NewRetentionScheduler(fp, 10*time.Millisecond, 30)
	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	// Wait for at least the immediate run + one tick.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if fp.calls >= 2 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	if fp.calls < 2 {
		t.Fatalf("expected at least 2 prune runs (immediate + tick), got %d", fp.calls)
	}
}
