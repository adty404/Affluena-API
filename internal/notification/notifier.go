package notification

import (
	"context"
	"log/slog"
)

// MailerPort sends a single HTML email. Implemented by the SMTP mailer adapter
// wired in the composition root. Optional: when nil, email sends are skipped
// (in-app deliveries still record).
type MailerPort interface {
	Send(ctx context.Context, to string, subject string, htmlBody string) error
}

// Notifier is the single rule-gated send path used by BOTH the scheduler and the
// budget-alert refactor. It:
//  1. consults the user's notification_rules row (enabled + channel), failing
//     closed when the rule is disabled or missing;
//  2. de-dupes via notification_deliveries (UNIQUE user+rule+dedupe_key), so the
//     same due item / weekly window is never notified twice;
//  3. records an in-app row and, when the channel includes email, sends email.
//
// The delivery row is inserted first: if it already exists (de-dupe hit) nothing
// is sent. This makes the whole operation idempotent under frequent ticks.
type Notifier struct {
	repo   DeliveryRepository
	mailer MailerPort
}

func NewNotifier(repo DeliveryRepository, mailer MailerPort) *Notifier {
	return &Notifier{repo: repo, mailer: mailer}
}

// Notification is a rendered, ready-to-send message. Title/Message must already
// be Indonesian (and any money grouped via money.GroupIDR).
type Notification struct {
	RuleKey    string
	DedupeKey  string
	Subject    string // email subject
	Title      string // in-app title
	Message    string // plain-text body (used for in-app + email)
	HTMLBody   string // optional; when empty an HTML body is built from Message
	Severity   string
	ActionPath string
}

// Decide returns the send decision for (userID, ruleKey) by consulting the
// user's rule row. A disabled or missing rule yields Send=false (fail closed).
func (n *Notifier) Decide(ctx context.Context, userID, ruleKey string) (Decision, error) {
	enabled, channel, found, err := n.repo.GetRule(ctx, userID, ruleKey)
	if err != nil {
		return Decision{}, err
	}
	if !found {
		return Decision{}, nil
	}
	return decisionFor(enabled, channel), nil
}

// Send applies the gate for (userID, ruleKey) and delivers the notification if
// enabled and not already sent. Returns true when a new delivery was recorded
// (i.e. something was actually emitted this call). A disabled/missing rule or a
// de-dupe hit returns false with no error.
func (n *Notifier) Send(ctx context.Context, userID string, notif Notification) (bool, error) {
	decision, err := n.Decide(ctx, userID, notif.RuleKey)
	if err != nil {
		return false, err
	}
	if !decision.Send {
		return false, nil
	}

	// De-dupe: record the delivery first. If a row already exists for this
	// user+rule+dedupe_key, do not send anything.
	inserted, err := n.repo.RecordDelivery(ctx, Delivery{
		UserID:     userID,
		RuleKey:    notif.RuleKey,
		DedupeKey:  notif.DedupeKey,
		Channel:    decision.Channel,
		Title:      notif.Title,
		Message:    notif.Message,
		Severity:   defaultSeverity(notif.Severity),
		ActionPath: notif.ActionPath,
	})
	if err != nil {
		return false, err
	}
	if !inserted {
		return false, nil // already delivered for this window
	}

	// Email leg (only when the channel includes email and a mailer is wired).
	if decision.Email && n.mailer != nil {
		email, err := n.repo.UserEmail(ctx, userID)
		if err != nil {
			slog.Error("notification: failed to load user email", "error", err, "user_id", userID, "rule", notif.RuleKey)
		} else if email != "" {
			body := notif.HTMLBody
			if body == "" {
				body = buildHTMLBody(notif.Title, notif.Message)
			}
			if err := n.mailer.Send(ctx, email, notif.Subject, body); err != nil {
				slog.Error("notification: email send failed", "error", err, "user_id", userID, "rule", notif.RuleKey)
			}
		}
	}

	return true, nil
}

func defaultSeverity(s string) string {
	if s == "" {
		return "info"
	}
	return s
}
