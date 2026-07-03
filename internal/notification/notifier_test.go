package notification

import (
	"context"
	"strings"
	"testing"
	"time"
)

// fakeDeliveryRepo is an in-memory DeliveryRepository for gating/de-dupe tests.
type fakeDeliveryRepo struct {
	rules map[string]struct {
		enabled bool
		channel string
	} // key: user|rule
	found      map[string]bool // key: user|rule
	delivered  map[string]bool // key: user|rule|dedupe
	email      string
	dueItems   map[int][]DueItem // key: daysAhead
	usersByKey map[string][]string
	cashflow   CashflowSummary

	recordCalls int
}

func newFakeRepo() *fakeDeliveryRepo {
	return &fakeDeliveryRepo{
		rules: map[string]struct {
			enabled bool
			channel string
		}{},
		found:      map[string]bool{},
		delivered:  map[string]bool{},
		dueItems:   map[int][]DueItem{},
		usersByKey: map[string][]string{},
	}
}

func rk(user, rule string) string { return user + "|" + rule }

func (f *fakeDeliveryRepo) setRule(user, rule string, enabled bool, channel string) {
	f.rules[rk(user, rule)] = struct {
		enabled bool
		channel string
	}{enabled, channel}
	f.found[rk(user, rule)] = true
}

func (f *fakeDeliveryRepo) GetRule(ctx context.Context, userID, ruleKey string) (bool, string, bool, error) {
	if !f.found[rk(userID, ruleKey)] {
		return false, "", false, nil
	}
	r := f.rules[rk(userID, ruleKey)]
	return r.enabled, r.channel, true, nil
}

func (f *fakeDeliveryRepo) RecordDelivery(ctx context.Context, d Delivery) (bool, error) {
	f.recordCalls++
	key := d.UserID + "|" + d.RuleKey + "|" + d.DedupeKey
	if f.delivered[key] {
		return false, nil
	}
	f.delivered[key] = true
	return true, nil
}

func (f *fakeDeliveryRepo) UserEmail(ctx context.Context, userID string) (string, error) {
	return f.email, nil
}

func (f *fakeDeliveryRepo) ListUserIDsWithRuleEnabled(ctx context.Context, ruleKey string) ([]string, error) {
	return f.usersByKey[ruleKey], nil
}

func (f *fakeDeliveryRepo) DueItemsForUser(ctx context.Context, userID string, asOf time.Time, daysAhead int) ([]DueItem, error) {
	return f.dueItems[daysAhead], nil
}

func (f *fakeDeliveryRepo) WeeklyCashflowForUser(ctx context.Context, userID string, from, to time.Time) (CashflowSummary, error) {
	return f.cashflow, nil
}

type recordingMailer struct {
	sends                         int
	lastTo, lastSubject, lastBody string
}

func (m *recordingMailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	m.sends++
	m.lastTo, m.lastSubject, m.lastBody = to, subject, htmlBody
	return nil
}

func TestDecisionForGating(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		channel string
		want    Decision
	}{
		{"disabled sends nothing", false, "both", Decision{}},
		{"email only", true, "email", Decision{Send: true, Email: true, Channel: "email"}},
		{"in-app only", true, "in-app", Decision{Send: true, InApp: true, Channel: "in-app"}},
		{"both", true, "both", Decision{Send: true, Email: true, InApp: true, Channel: "both"}},
		{"unknown channel sends nothing", true, "sms", Decision{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := decisionFor(tc.enabled, tc.channel)
			if got != tc.want {
				t.Fatalf("decisionFor(%v,%q) = %+v, want %+v", tc.enabled, tc.channel, got, tc.want)
			}
		})
	}
}

func TestNotifierSendGatingAndDedup(t *testing.T) {
	ctx := context.Background()
	user := "user-1"
	notif := Notification{RuleKey: RuleKeyDueReminder, DedupeKey: "sub:1:H-3:2026-07-10", Subject: "s", Title: "t", Message: "m"}

	t.Run("disabled rule sends nothing", func(t *testing.T) {
		repo := newFakeRepo()
		repo.setRule(user, RuleKeyDueReminder, false, "both")
		mm := &recordingMailer{}
		n := NewNotifier(repo, mm)
		sent, err := n.Send(ctx, user, notif)
		if err != nil || sent {
			t.Fatalf("expected no send for disabled rule, sent=%v err=%v", sent, err)
		}
		if repo.recordCalls != 0 || mm.sends != 0 {
			t.Fatalf("disabled rule must not record or email (record=%d email=%d)", repo.recordCalls, mm.sends)
		}
	})

	t.Run("missing rule fails closed", func(t *testing.T) {
		repo := newFakeRepo() // no rule set
		n := NewNotifier(repo, &recordingMailer{})
		sent, err := n.Send(ctx, user, notif)
		if err != nil || sent {
			t.Fatalf("missing rule must fail closed, sent=%v err=%v", sent, err)
		}
	})

	t.Run("in-app only records but does not email", func(t *testing.T) {
		repo := newFakeRepo()
		repo.setRule(user, RuleKeyDueReminder, true, "in-app")
		repo.email = "u@example.com"
		mm := &recordingMailer{}
		n := NewNotifier(repo, mm)
		sent, err := n.Send(ctx, user, notif)
		if err != nil || !sent {
			t.Fatalf("expected send=true, got sent=%v err=%v", sent, err)
		}
		if mm.sends != 0 {
			t.Fatalf("in-app channel must not email, got %d", mm.sends)
		}
	})

	t.Run("email channel emails and records", func(t *testing.T) {
		repo := newFakeRepo()
		repo.setRule(user, RuleKeyDueReminder, true, "both")
		repo.email = "u@example.com"
		mm := &recordingMailer{}
		n := NewNotifier(repo, mm)
		sent, err := n.Send(ctx, user, notif)
		if err != nil || !sent {
			t.Fatalf("expected send, got sent=%v err=%v", sent, err)
		}
		if mm.sends != 1 || mm.lastTo != "u@example.com" {
			t.Fatalf("expected one email to user, got sends=%d to=%q", mm.sends, mm.lastTo)
		}
	})

	t.Run("second identical send is de-duped", func(t *testing.T) {
		repo := newFakeRepo()
		repo.setRule(user, RuleKeyDueReminder, true, "both")
		repo.email = "u@example.com"
		mm := &recordingMailer{}
		n := NewNotifier(repo, mm)
		if _, err := n.Send(ctx, user, notif); err != nil {
			t.Fatal(err)
		}
		sent, err := n.Send(ctx, user, notif)
		if err != nil || sent {
			t.Fatalf("second send must be de-duped, sent=%v err=%v", sent, err)
		}
		if mm.sends != 1 {
			t.Fatalf("de-dupe must prevent a second email, got %d", mm.sends)
		}
	})
}

func TestDueReminderNotificationCopy(t *testing.T) {
	item := DueItem{
		EntityType:   "subscription",
		EntityID:     "abc",
		Name:         "Netflix",
		AmountMinor:  186000,
		DueDate:      time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		DaysUntilDue: 3,
	}
	n := dueReminderNotification(item)
	if !strings.Contains(n.Message, "Rp 186.000") {
		t.Fatalf("expected grouped rupiah in message, got %q", n.Message)
	}
	if !strings.Contains(n.Message, "Langganan") {
		t.Fatalf("expected Indonesian entity label, got %q", n.Message)
	}
	if n.DedupeKey != "subscription:abc:H-3:2026-07-10" {
		t.Fatalf("unexpected dedupe key %q", n.DedupeKey)
	}
	if n.ActionPath != "/subscriptions/abc" {
		t.Fatalf("unexpected action path %q", n.ActionPath)
	}
}

func TestWeeklySummaryNotificationCopy(t *testing.T) {
	n := weeklySummaryNotification(CashflowSummary{IncomeMinor: 5000000, ExpenseMinor: 3200000, NetMinor: 1800000}, "2026-W28")
	if !strings.Contains(n.Message, "Rp 5.000.000") || !strings.Contains(n.Message, "Rp 3.200.000") {
		t.Fatalf("expected grouped rupiah in summary, got %q", n.Message)
	}
	if !strings.Contains(n.Message, "surplus") {
		t.Fatalf("expected surplus label for positive net, got %q", n.Message)
	}
	if n.DedupeKey != "weekly-summary:2026-W28" {
		t.Fatalf("unexpected dedupe key %q", n.DedupeKey)
	}

	deficit := weeklySummaryNotification(CashflowSummary{IncomeMinor: 1000000, ExpenseMinor: 1500000, NetMinor: -500000}, "2026-W28")
	if !strings.Contains(deficit.Message, "defisit Rp 500.000") {
		t.Fatalf("expected defisit label with absolute value, got %q", deficit.Message)
	}
}
