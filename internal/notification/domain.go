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
