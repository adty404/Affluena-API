package budget

import (
	"context"
	"errors"
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
		SELECT b.id::text, b.user_id::text, b.category_id::text, b.month, b.limit_minor,
			b.created_at, b.updated_at,
			COALESCE(SUM(t.amount_minor) FILTER (
				WHERE t.type = 'expense'
					AND t.transaction_at >= b.month
					AND t.transaction_at < b.month + interval '1 month'
			), 0) AS spent_minor
		FROM category_budgets b
		LEFT JOIN transactions t ON t.user_id = b.user_id AND t.category_id = b.category_id
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
