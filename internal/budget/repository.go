package budget

import (
	"context"
	"errors"
	"fmt"
	"time"

	"affluena-api/internal/page"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, userID string, input CreateBudgetInput) (Budget, error) {
	if err := r.ensureExpenseCategory(ctx, userID, input.CategoryID); err != nil {
		return Budget{}, err
	}

	budget, err := scanBudget(r.pool.QueryRow(ctx, `
		INSERT INTO category_budgets (user_id, category_id, month, limit_minor)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, user_id::text, category_id::text, month, limit_minor, created_at, updated_at
	`, userID, input.CategoryID, input.MonthDate, input.LimitMinor))
	return budget, translateNotFound(err)
}

func (r *Repository) List(ctx context.Context, userID string, month time.Time, pagination page.Params) (page.Result[BudgetSummary], error) {
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE category_tree AS (
			SELECT id, id AS root_id
			FROM categories
			WHERE user_id = $1
			UNION ALL
			SELECT c.id, ct.root_id
			FROM categories c
			INNER JOIN category_tree ct ON c.parent_id = ct.id
		)
		SELECT b.id::text, b.user_id::text, b.category_id::text, b.month, b.limit_minor,
			b.created_at, b.updated_at,
			COALESCE(SUM(t.amount_minor) FILTER (
				WHERE t.type = 'expense'
					AND t.transaction_at >= b.month
					AND t.transaction_at < b.month + interval '1 month'
			), 0) AS spent_minor
		FROM category_budgets b
		LEFT JOIN category_tree ct ON ct.root_id = b.category_id
		LEFT JOIN transactions t ON t.user_id = b.user_id AND t.category_id = ct.id
		WHERE b.user_id = $1 AND b.month = $2
		GROUP BY b.id
		ORDER BY `+budgetOrderBy(pagination.Sort)+`
		LIMIT $3 OFFSET $4
	`, userID, month, pagination.Limit, pagination.Offset)
	if err != nil {
		return page.Result[BudgetSummary]{}, err
	}
	defer rows.Close()

	var summaries []BudgetSummary
	for rows.Next() {
		summary, err := scanBudgetSummary(rows)
		if err != nil {
			return page.Result[BudgetSummary]{}, err
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return page.Result[BudgetSummary]{}, err
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM category_budgets WHERE user_id = $1 AND month = $2`, userID, month).Scan(&total); err != nil {
		return page.Result[BudgetSummary]{}, err
	}
	return page.NewResult(summaries, pagination, total), nil
}

func (r *Repository) ListAlerts(ctx context.Context, userID string, month time.Time) ([]BudgetAlert, error) {
	monthValue := month.Format("2006-01")
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE category_tree AS (
			SELECT id, id AS root_id
			FROM categories
			WHERE user_id = $1
			UNION ALL
			SELECT c.id, ct.root_id
			FROM categories c
			INNER JOIN category_tree ct ON c.parent_id = ct.id
		),
		budget_usage AS (
			SELECT b.id, b.user_id, b.category_id, b.month, b.limit_minor,
				COALESCE(SUM(t.amount_minor) FILTER (
					WHERE t.type = 'expense'
						AND t.transaction_at >= b.month
						AND t.transaction_at < b.month + interval '1 month'
				), 0) AS spent_minor
			FROM category_budgets b
			LEFT JOIN category_tree ct ON ct.root_id = b.category_id
			LEFT JOIN transactions t ON t.user_id = b.user_id AND t.category_id = ct.id
			WHERE b.user_id = $1 AND b.month = $2
			GROUP BY b.id
		)
		SELECT bu.id::text, bu.category_id::text, c.name, bu.limit_minor, bu.spent_minor,
			MAX(sa.sent_at) AS notified_at
		FROM budget_usage bu
		INNER JOIN categories c ON c.id = bu.category_id
		LEFT JOIN sent_alerts sa ON sa.user_id = bu.user_id 
			AND sa.category_id = bu.category_id 
			AND sa.month_value = $3
			AND sa.alert_type = CASE WHEN (bu.spent_minor::float / NULLIF(bu.limit_minor, 0)) >= 1.0 THEN 'EXCEEDED' ELSE 'WARNING' END
		WHERE bu.limit_minor > 0 AND (bu.spent_minor::float / bu.limit_minor) >= 0.8
		GROUP BY bu.id, bu.category_id, c.name, bu.limit_minor, bu.spent_minor
		ORDER BY (bu.spent_minor::float / bu.limit_minor) DESC
	`, userID, month, monthValue)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []BudgetAlert
	for rows.Next() {
		var alert BudgetAlert
		var notifiedAt *time.Time
		err := rows.Scan(&alert.BudgetID, &alert.CategoryID, &alert.CategoryName, &alert.LimitMinor, &alert.SpentMinor, &notifiedAt)
		if err != nil {
			return nil, err
		}

		alert.ID = fmt.Sprintf("alert-%s-%s", alert.BudgetID, monthValue)
		alert.Month = monthValue
		alert.UsagePercent = float64(alert.SpentMinor) / float64(alert.LimitMinor) * 100

		if alert.UsagePercent >= 100 {
			alert.Threshold = 100
			alert.Severity = "danger"
			alert.Title = fmt.Sprintf("%s budget exceeded", alert.CategoryName)
		} else {
			alert.Threshold = 80
			alert.Severity = "warning"
			alert.Title = fmt.Sprintf("%s budget warning", alert.CategoryName)
		}

		alert.Message = fmt.Sprintf("Usage reached %.0f%%. Limit Rp %d, actual Rp %d.", alert.UsagePercent, alert.LimitMinor, alert.SpentMinor)

		if notifiedAt != nil {
			notifiedAtStr := notifiedAt.Format(time.RFC3339)
			alert.NotifiedAt = &notifiedAtStr
		}

		alerts = append(alerts, alert)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if alerts == nil {
		alerts = []BudgetAlert{}
	}
	return alerts, nil
}

func (r *Repository) ListReport(ctx context.Context, userID string, month time.Time) ([]BudgetReportItem, BudgetReportSummary, error) {
	// Reuse the List query logic but without pagination
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE category_tree AS (
			SELECT id, id AS root_id
			FROM categories
			WHERE user_id = $1
			UNION ALL
			SELECT c.id, ct.root_id
			FROM categories c
			INNER JOIN category_tree ct ON c.parent_id = ct.id
		)
		SELECT b.id::text, b.user_id::text, b.category_id::text, b.month, b.limit_minor,
			b.created_at, b.updated_at,
			COALESCE(SUM(t.amount_minor) FILTER (
				WHERE t.type = 'expense'
					AND t.transaction_at >= b.month
					AND t.transaction_at < b.month + interval '1 month'
			), 0) AS spent_minor
		FROM category_budgets b
		LEFT JOIN category_tree ct ON ct.root_id = b.category_id
		LEFT JOIN transactions t ON t.user_id = b.user_id AND t.category_id = ct.id
		WHERE b.user_id = $1 AND b.month = $2
		GROUP BY b.id
		ORDER BY b.limit_minor DESC, b.created_at DESC
	`, userID, month)
	if err != nil {
		return nil, BudgetReportSummary{}, err
	}
	defer rows.Close()

	var items []BudgetReportItem
	var summary BudgetReportSummary

	now := time.Now().UTC()

	year, m, _ := month.Date()
	firstOfMonth := time.Date(year, m, 1, 0, 0, 0, 0, time.UTC)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	totalDaysInMonth := lastOfMonth.Day()

	var daysLeftInMonth int
	var daysElapsedInMonth int

	if now.Before(firstOfMonth) {
		daysLeftInMonth = totalDaysInMonth
		daysElapsedInMonth = 0
	} else if now.After(lastOfMonth.Add(24 * time.Hour)) {
		daysLeftInMonth = 0
		daysElapsedInMonth = totalDaysInMonth
	} else {
		daysElapsedInMonth = now.Day()
		daysLeftInMonth = totalDaysInMonth - daysElapsedInMonth
		if daysLeftInMonth < 0 {
			daysLeftInMonth = 0
		}
	}

	for rows.Next() {
		bs, err := scanBudgetSummary(rows)
		if err != nil {
			return nil, BudgetReportSummary{}, err
		}

		item := BudgetReportItem{
			BudgetSummary: bs,
			VarianceMinor: bs.RemainingMinor,
		}

		if daysLeftInMonth > 0 && bs.RemainingMinor > 0 {
			item.DailyAllowanceMinor = bs.RemainingMinor / int64(daysLeftInMonth)
		} else {
			item.DailyAllowanceMinor = 0
		}

		if bs.UsagePercent >= 100 {
			item.Recommendation = "exceeded"
			summary.ExceededCount++
		} else if bs.UsagePercent >= 80 {
			item.Recommendation = "warning"
			summary.WarningCount++
		} else {
			item.Recommendation = "safe"
			summary.SafeCount++
		}

		summary.TotalLimitMinor += bs.LimitMinor
		summary.TotalSpentMinor += bs.SpentMinor
		summary.TotalRemainingMinor += bs.RemainingMinor
		summary.DailyAllowanceMinor += item.DailyAllowanceMinor

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, BudgetReportSummary{}, err
	}

	if items == nil {
		items = []BudgetReportItem{}
	}

	if daysElapsedInMonth > 0 {
		dailyAvg := float64(summary.TotalSpentMinor) / float64(daysElapsedInMonth)
		summary.ForecastMinor = summary.TotalSpentMinor + int64(dailyAvg*float64(daysLeftInMonth))
	} else {
		summary.ForecastMinor = 0
	}

	return items, summary, nil
}

func budgetOrderBy(sort string) string {
	switch sort {
	case "created_at_asc":
		return "b.created_at ASC"
	case "limit_desc":
		return "b.limit_minor DESC, b.created_at DESC"
	case "limit_asc":
		return "b.limit_minor ASC, b.created_at DESC"
	case "spent_desc":
		return "spent_minor DESC, b.created_at DESC"
	case "spent_asc":
		return "spent_minor ASC, b.created_at DESC"
	default:
		return "b.created_at DESC"
	}
}

func (r *Repository) Get(ctx context.Context, userID string, id string) (Budget, error) {
	budget, err := scanBudget(r.pool.QueryRow(ctx, `
		SELECT id::text, user_id::text, category_id::text, month, limit_minor, created_at, updated_at
		FROM category_budgets
		WHERE user_id = $1 AND id = $2
	`, userID, id))
	return budget, translateNotFound(err)
}

func (r *Repository) Update(ctx context.Context, userID string, id string, input UpdateBudgetInput) (Budget, error) {
	if err := r.ensureExpenseCategory(ctx, userID, input.CategoryID); err != nil {
		return Budget{}, err
	}

	budget, err := scanBudget(r.pool.QueryRow(ctx, `
		UPDATE category_budgets
		SET category_id = $3, month = $4, limit_minor = $5, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, category_id::text, month, limit_minor, created_at, updated_at
	`, userID, id, input.CategoryID, input.MonthDate, input.LimitMinor))
	return budget, translateNotFound(err)
}

func (r *Repository) Delete(ctx context.Context, userID string, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM category_budgets WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ensureExpenseCategory(ctx context.Context, userID string, categoryID string) error {
	var exists bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM categories
			WHERE user_id = $1 AND id = $2 AND type = 'expense'
		)
	`, userID, categoryID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBudget(row rowScanner) (Budget, error) {
	var budget Budget
	err := row.Scan(&budget.ID, &budget.UserID, &budget.CategoryID, &budget.Month, &budget.LimitMinor, &budget.CreatedAt, &budget.UpdatedAt)
	return budget, err
}

func scanBudgetSummary(row rowScanner) (BudgetSummary, error) {
	var summary BudgetSummary
	var spentMinor int64
	err := row.Scan(
		&summary.ID,
		&summary.UserID,
		&summary.CategoryID,
		&summary.Month,
		&summary.LimitMinor,
		&summary.CreatedAt,
		&summary.UpdatedAt,
		&spentMinor,
	)
	if err != nil {
		return BudgetSummary{}, err
	}
	usage := NewUsageSummary(summary.LimitMinor, spentMinor)
	summary.SpentMinor = usage.SpentMinor
	summary.RemainingMinor = usage.RemainingMinor
	summary.UsagePercent = usage.UsagePercent
	return summary, nil
}

func translateNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
