package notification

import (
	"context"
	"testing"
	"time"
)

func TestPreviousISOWeek(t *testing.T) {
	// Wednesday 2026-07-08. The previous full Mon–Sun week is
	// 2026-06-29 (Mon) .. 2026-07-06 (Mon, exclusive), which is ISO week 27.
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	from, to, key := previousISOWeek(now)

	wantFrom := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
	if !from.Equal(wantFrom) {
		t.Fatalf("from = %v, want %v", from, wantFrom)
	}
	if !to.Equal(wantTo) {
		t.Fatalf("to = %v, want %v", to, wantTo)
	}
	if key != "2026-W27" {
		t.Fatalf("key = %q, want 2026-W27", key)
	}
	// The window is exactly 7 days and ends before 'now'.
	if to.Sub(from) != 7*24*time.Hour {
		t.Fatalf("window is not 7 days: %v", to.Sub(from))
	}
	if !to.Before(now) {
		t.Fatalf("previous week must be complete (to < now)")
	}
}

func TestISOWeekKeyZeroPads(t *testing.T) {
	if got := isoWeekKey(2026, 7); got != "2026-W07" {
		t.Fatalf("isoWeekKey(2026,7) = %q, want 2026-W07", got)
	}
	if got := isoWeekKey(2026, 27); got != "2026-W27" {
		t.Fatalf("isoWeekKey(2026,27) = %q, want 2026-W27", got)
	}
}

func TestSchedulerDueRemindersSelectBothWindowsAndGate(t *testing.T) {
	ctx := context.Background()
	user := "user-1"

	repo := newFakeRepo()
	repo.usersByKey[RuleKeyDueReminder] = []string{user}
	repo.setRule(user, RuleKeyDueReminder, true, "in-app")
	// One item at H-3, one at H-1 → both windows should be queried and sent.
	repo.dueItems[3] = []DueItem{{EntityType: "subscription", EntityID: "s3", Name: "Netflix", AmountMinor: 100000, DueDate: time.Now().UTC().AddDate(0, 0, 3), DaysUntilDue: 3}}
	repo.dueItems[1] = []DueItem{{EntityType: "debt", EntityID: "d1", Name: "BCA", AmountMinor: 200000, DueDate: time.Now().UTC().AddDate(0, 0, 1), DaysUntilDue: 1}}

	s := NewScheduler(repo, NewNotifier(repo, nil), time.Hour)
	sent := s.runDueReminders(ctx)
	if sent != 2 {
		t.Fatalf("expected 2 reminders sent (H-3 + H-1), got %d", sent)
	}

	// A second run must be fully de-duped (nothing new sent).
	sent2 := s.runDueReminders(ctx)
	if sent2 != 0 {
		t.Fatalf("expected 0 on re-run (de-duped), got %d", sent2)
	}
}

func TestSchedulerDueRemindersSkipDisabledUsers(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	// No users returned as enabled → nothing sent even if due items exist.
	repo.dueItems[3] = []DueItem{{EntityType: "subscription", EntityID: "s3", Name: "X", AmountMinor: 1, DueDate: time.Now(), DaysUntilDue: 3}}

	s := NewScheduler(repo, NewNotifier(repo, nil), time.Hour)
	if sent := s.runDueReminders(ctx); sent != 0 {
		t.Fatalf("expected 0 sends when no user has the rule enabled, got %d", sent)
	}
}

func TestSchedulerWeeklySummaryOncePerWeek(t *testing.T) {
	ctx := context.Background()
	user := "user-1"

	repo := newFakeRepo()
	repo.usersByKey[RuleKeyWeeklySummary] = []string{user}
	repo.setRule(user, RuleKeyWeeklySummary, true, "email")
	repo.email = "u@example.com"
	repo.cashflow = CashflowSummary{IncomeMinor: 5000000, ExpenseMinor: 3000000, NetMinor: 2000000}

	// Pin 'now' so both runs land in the same ISO week.
	fixedNow := time.Date(2026, 7, 8, 9, 0, 0, 0, time.UTC)
	s := NewScheduler(repo, NewNotifier(repo, &recordingMailer{}), time.Hour)
	s.now = func() time.Time { return fixedNow }

	if sent := s.runWeeklySummary(ctx); sent != 1 {
		t.Fatalf("expected 1 weekly summary, got %d", sent)
	}
	// Same week → de-duped.
	if sent := s.runWeeklySummary(ctx); sent != 0 {
		t.Fatalf("expected weekly summary de-duped within the same week, got %d", sent)
	}
}
