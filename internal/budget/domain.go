package budget

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("category budget not found")

type Budget struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	CategoryID string    `json:"category_id"`
	Month      time.Time `json:"month"`
	LimitMinor int64     `json:"limit_minor"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type BudgetSummary struct {
	Budget
	SpentMinor     int64   `json:"spent_minor"`
	RemainingMinor int64   `json:"remaining_minor"`
	UsagePercent   float64 `json:"usage_percent"`
}

type CreateBudgetInput struct {
	CategoryID string
	Month      string
	MonthDate  time.Time
	LimitMinor int64
}

type UpdateBudgetInput struct {
	CategoryID string
	Month      string
	MonthDate  time.Time
	LimitMinor int64
}

func NotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
