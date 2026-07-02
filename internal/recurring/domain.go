package recurring

import (
	"time"

	"affluena-api/internal/transaction"
)

type Rule struct {
	ID            string                      `json:"id"`
	UserID        string                      `json:"user_id"`
	Name          string                      `json:"name"`
	Type          transaction.TransactionType `json:"type"`
	WalletID      string                      `json:"wallet_id"`
	ToWalletID    string                      `json:"to_wallet_id,omitempty"`
	CategoryID    string                      `json:"category_id,omitempty"`
	AmountMinor   int64                       `json:"amount_minor"`
	Frequency     Frequency                   `json:"frequency"`
	IntervalCount int                         `json:"interval_count"`
	NextRunAt     time.Time                   `json:"next_run_at"`
	EndAt         *time.Time                  `json:"end_at,omitempty"`
	LastRunAt     *time.Time                  `json:"last_run_at,omitempty"`
	Status        Status                      `json:"status"`
	Note          string                      `json:"note"`
	Color         string                      `json:"color"`
	Icon          string                      `json:"icon"`
	CreatedAt     time.Time                   `json:"created_at"`
	UpdatedAt     time.Time                   `json:"updated_at"`
}

type RuleInput struct {
	Name          string
	Type          transaction.TransactionType
	WalletID      string
	ToWalletID    string
	CategoryID    string
	AmountMinor   int64
	Frequency     Frequency
	IntervalCount int
	NextRunAt     time.Time
	EndAt         *time.Time
	Status        Status
	Note          string
	Color         string
	Icon          string
}

type Run struct {
	ID            string                  `json:"id"`
	RuleID        string                  `json:"rule_id"`
	UserID        string                  `json:"user_id"`
	ScheduledFor  time.Time               `json:"scheduled_for"`
	TransactionID string                  `json:"transaction_id,omitempty"`
	RunType       RunType                 `json:"run_type"`
	CreatedAt     time.Time               `json:"created_at"`
	Transaction   transaction.Transaction `json:"transaction"`
	Rule          Rule                    `json:"rule"`
}
