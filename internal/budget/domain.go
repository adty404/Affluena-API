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
	Color      string    `json:"color"`
	Icon       string    `json:"icon"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type BudgetSummary struct {
	Budget
	SpentMinor     int64   `json:"spent_minor"`
	RemainingMinor int64   `json:"remaining_minor"`
	UsagePercent   float64 `json:"usage_percent"`
}

type BudgetAlert struct {
	ID           string  `json:"id"`
	BudgetID     string  `json:"budget_id"`
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Title        string  `json:"title"`
	Message      string  `json:"message"`
	Threshold    int     `json:"threshold"`
	Severity     string  `json:"severity"`
	UsagePercent float64 `json:"usage_percent"`
	SpentMinor   int64   `json:"spent_minor"`
	LimitMinor   int64   `json:"limit_minor"`
	NotifiedAt   *string `json:"notified_at"`
	Month        string  `json:"month"`
}

type BudgetReportItem struct {
	BudgetSummary
	VarianceMinor       int64  `json:"variance_minor"`
	DailyAllowanceMinor int64  `json:"daily_allowance_minor"`
	Recommendation      string `json:"recommendation"`
}

type BudgetReportSummary struct {
	TotalLimitMinor     int64 `json:"total_limit_minor"`
	TotalSpentMinor     int64 `json:"total_spent_minor"`
	TotalRemainingMinor int64 `json:"total_remaining_minor"`
	SafeCount           int   `json:"safe_count"`
	WarningCount        int   `json:"warning_count"`
	ExceededCount       int   `json:"exceeded_count"`
	DailyAllowanceMinor int64 `json:"daily_allowance_minor"`
	ForecastMinor       int64 `json:"forecast_minor"`
}

type CreateBudgetInput struct {
	CategoryID string
	Month      string
	MonthDate  time.Time
	LimitMinor int64
	Color      string
	Icon       string
}

type UpdateBudgetInput struct {
	CategoryID string
	Month      string
	MonthDate  time.Time
	LimitMinor int64
	Color      string
	Icon       string
}

func NotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
