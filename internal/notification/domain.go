package notification

import (
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("notification rule not found")
)

type NotificationRule struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	RuleKey     string    `json:"rule_key"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	Channel     string    `json:"channel"`
	Tone        string    `json:"tone"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type NotificationRuleUpdate struct {
	Enabled *bool   `json:"enabled"`
	Channel *string `json:"channel"`
}

func IsValidChannel(channel string) bool {
	switch channel {
	case "email", "in-app", "both":
		return true
	default:
		return false
	}
}

// Rule keys for the five seeded notification rules. Used to gate sends and to
// look up a user's preference row.
const (
	RuleKeyBudgetAlert   = "budget-alert"
	RuleKeyDueReminder   = "due-reminder"
	RuleKeyRecurringRun  = "recurring-run"
	RuleKeySecurityAlert = "security-alert"
	RuleKeyWeeklySummary = "weekly-summary"
)

// Decision is the result of consulting a user's notification_rules row for a
// given rule_key. It says whether to send at all and, if so, over which
// channels. A disabled rule (or a missing row) yields Send=false.
type Decision struct {
	Send    bool
	Email   bool
	InApp   bool
	Channel string // the raw rule channel, recorded on the delivery row
}

// decisionFor turns a rule's enabled/channel into a Decision. A missing rule
// (enabled defaults to false via the zero value) or a disabled rule sends
// nothing — callers must fail closed.
func decisionFor(enabled bool, channel string) Decision {
	if !enabled {
		return Decision{}
	}
	switch channel {
	case "email":
		return Decision{Send: true, Email: true, Channel: channel}
	case "in-app":
		return Decision{Send: true, InApp: true, Channel: channel}
	case "both":
		return Decision{Send: true, Email: true, InApp: true, Channel: channel}
	default:
		// Unknown channel: treat as no-send rather than guessing.
		return Decision{}
	}
}

// DueItem is a subscription/installment/debt due within a reminder window
// (H-3 or H-1), used to build due-reminder notifications.
type DueItem struct {
	EntityType   string // "subscription" | "installment" | "debt"
	EntityID     string
	Name         string
	AmountMinor  int64
	DueDate      time.Time
	DaysUntilDue int // 3 or 1
}

// CashflowSummary is the minimal cashflow rollup used by the weekly summary.
type CashflowSummary struct {
	IncomeMinor  int64
	ExpenseMinor int64
	NetMinor     int64
}
