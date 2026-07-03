package notification

import (
	"context"
	"log/slog"
	"time"
)

// Scheduler periodically emits rule-gated notifications, mirroring
// recurring.NewScheduler's lifecycle (immediate run on start, then every
// interval until the context is cancelled). Each tick:
//   - Due reminders: for every user with the due-reminder rule enabled, find
//     subscriptions/installments/debts due at H-3 and H-1 and emit a gated,
//     de-duped notification.
//   - Weekly summary: for every user with the weekly-summary rule enabled, emit
//     a cashflow summary. De-dupe is keyed on the ISO week, so it fires at most
//     once per week regardless of tick frequency.
//
// All sends go through Notifier, which fails closed on a disabled/missing rule
// and de-dupes via notification_deliveries, so frequent ticks never spam.
type Scheduler struct {
	repo     DeliveryRepository
	notifier *Notifier
	interval time.Duration
	// weekStart is the weekday the summary window ends on / new week begins.
	// Monday-based ISO weeks are used for the de-dupe key.
	now func() time.Time
}

func NewScheduler(repo DeliveryRepository, notifier *Notifier, interval time.Duration) *Scheduler {
	if interval <= 0 {
		interval = time.Hour
	}
	return &Scheduler{repo: repo, notifier: notifier, interval: interval, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Scheduler) Start(ctx context.Context) {
	go func() {
		s.runOnce(ctx)

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("notification scheduler stopped")
				return
			case <-ticker.C:
				s.runOnce(ctx)
			}
		}
	}()
}

func (s *Scheduler) runOnce(ctx context.Context) {
	if sent := s.runDueReminders(ctx); sent > 0 {
		slog.Info("notification scheduler sent due reminders", "count", sent)
	}
	if sent := s.runWeeklySummary(ctx); sent > 0 {
		slog.Info("notification scheduler sent weekly summaries", "count", sent)
	}
}

// runDueReminders emits H-3 and H-1 reminders for every user with the
// due-reminder rule enabled. Returns the number of notifications actually sent.
func (s *Scheduler) runDueReminders(ctx context.Context) int {
	users, err := s.repo.ListUserIDsWithRuleEnabled(ctx, RuleKeyDueReminder)
	if err != nil {
		slog.Error("notification scheduler: list due-reminder users failed", "error", err)
		return 0
	}

	now := s.now()
	sent := 0
	for _, userID := range users {
		for _, daysAhead := range []int{3, 1} {
			items, err := s.repo.DueItemsForUser(ctx, userID, now, daysAhead)
			if err != nil {
				slog.Error("notification scheduler: due items query failed", "error", err, "user_id", userID, "days_ahead", daysAhead)
				continue
			}
			for _, item := range items {
				notif := dueReminderNotification(item)
				ok, err := s.notifier.Send(ctx, userID, notif)
				if err != nil {
					slog.Error("notification scheduler: due reminder send failed", "error", err, "user_id", userID, "entity", item.EntityID)
					continue
				}
				if ok {
					sent++
				}
			}
		}
	}
	return sent
}

// runWeeklySummary emits a cashflow summary for the PREVIOUS full ISO week to
// every user with the weekly-summary rule enabled. De-dupe on the ISO week key
// keeps it to once per week no matter how often the scheduler ticks.
func (s *Scheduler) runWeeklySummary(ctx context.Context) int {
	users, err := s.repo.ListUserIDsWithRuleEnabled(ctx, RuleKeyWeeklySummary)
	if err != nil {
		slog.Error("notification scheduler: list weekly-summary users failed", "error", err)
		return 0
	}

	from, to, weekKey := previousISOWeek(s.now())
	sent := 0
	for _, userID := range users {
		summary, err := s.repo.WeeklyCashflowForUser(ctx, userID, from, to)
		if err != nil {
			slog.Error("notification scheduler: weekly cashflow query failed", "error", err, "user_id", userID)
			continue
		}
		notif := weeklySummaryNotification(summary, weekKey)
		ok, err := s.notifier.Send(ctx, userID, notif)
		if err != nil {
			slog.Error("notification scheduler: weekly summary send failed", "error", err, "user_id", userID)
			continue
		}
		if ok {
			sent++
		}
	}
	return sent
}

// previousISOWeek returns [from, to) covering the most recent COMPLETED Monday-
// to-Sunday week relative to now, plus that week's ISO key ("2006-W27"). Running
// on the summary of the finished week avoids a partial-week report.
func previousISOWeek(now time.Time) (from, to time.Time, key string) {
	// Monday of the current week (UTC, day granularity).
	weekday := int(now.Weekday())
	// Go's Weekday has Sunday=0; convert so Monday=0.
	offset := (weekday + 6) % 7
	currentMonday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -offset)
	from = currentMonday.AddDate(0, 0, -7) // previous Monday
	to = currentMonday                     // exclusive: this Monday
	year, week := from.ISOWeek()
	key = isoWeekKey(year, week)
	return from, to, key
}

func isoWeekKey(year, week int) string {
	// Zero-pad the week to two digits for a stable key, e.g. 2026-W07.
	w := week
	prefix := "W"
	if w < 10 {
		prefix = "W0"
	}
	return itoa(year) + "-" + prefix + itoa(w)
}

// itoa is a tiny local int-to-string to avoid importing strconv for two calls.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
