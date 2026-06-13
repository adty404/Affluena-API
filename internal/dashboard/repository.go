package dashboard

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Summary(ctx context.Context, userID string, month time.Time) (Summary, error) {
	nextMonth := month.AddDate(0, 1, 0)
	summary := Summary{Month: month.Format("2006-01")}

	if err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(balance_minor), 0)
		FROM wallets
		WHERE user_id = $1
	`, userID).Scan(&summary.NetWorthMinor); err != nil {
		return Summary{}, err
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount_minor) FILTER (WHERE type = 'income'), 0),
			COALESCE(SUM(amount_minor) FILTER (WHERE type = 'expense'), 0)
		FROM transactions
		WHERE user_id = $1
			AND transaction_at >= $2
			AND transaction_at < $3
	`, userID, month, nextMonth).Scan(&summary.MonthlyIncomeMinor, &summary.MonthlyExpenseMinor); err != nil {
		return Summary{}, err
	}
	summary.MonthlyCashflowMinor = summary.MonthlyIncomeMinor - summary.MonthlyExpenseMinor

	budget, err := r.budgetSummary(ctx, userID, month)
	if err != nil {
		return Summary{}, err
	}
	summary.Budget = budget

	subscriptions, err := r.upcomingSubscriptions(ctx, userID, month, nextMonth)
	if err != nil {
		return Summary{}, err
	}
	summary.UpcomingSubscriptions = subscriptions

	installments, err := r.upcomingInstallments(ctx, userID, month, nextMonth)
	if err != nil {
		return Summary{}, err
	}
	summary.UpcomingInstallments = installments

	debts, err := r.upcomingDebts(ctx, userID, month, nextMonth)
	if err != nil {
		return Summary{}, err
	}
	summary.UpcomingDebts = debts

	return summary, nil
}

func (r *Repository) budgetSummary(ctx context.Context, userID string, month time.Time) (BudgetSummary, error) {
	var limitMinor int64
	var spentMinor int64
	err := r.pool.QueryRow(ctx, `
		WITH budget_spend AS (
			SELECT b.limit_minor,
				COALESCE(SUM(t.amount_minor) FILTER (
					WHERE t.type = 'expense'
						AND t.transaction_at >= b.month
						AND t.transaction_at < b.month + interval '1 month'
				), 0) AS spent_minor
			FROM category_budgets b
			LEFT JOIN transactions t ON t.user_id = b.user_id AND t.category_id = b.category_id
			WHERE b.user_id = $1 AND b.month = $2
			GROUP BY b.id
		)
		SELECT COALESCE(SUM(limit_minor), 0), COALESCE(SUM(spent_minor), 0)
		FROM budget_spend
	`, userID, month).Scan(&limitMinor, &spentMinor)
	if err != nil {
		return BudgetSummary{}, err
	}
	return newBudgetSummary(limitMinor, spentMinor), nil
}

func (r *Repository) upcomingSubscriptions(ctx context.Context, userID string, month time.Time, nextMonth time.Time) ([]UpcomingSubscription, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name, account_detail, amount_minor, next_due_date
		FROM subscriptions
		WHERE user_id = $1
			AND status = 'active'
			AND next_due_date >= $2
			AND next_due_date < $3
		ORDER BY next_due_date ASC, created_at DESC
		LIMIT 10
	`, userID, month, nextMonth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []UpcomingSubscription
	for rows.Next() {
		var subscription UpcomingSubscription
		if err := rows.Scan(&subscription.ID, &subscription.Name, &subscription.AccountDetail, &subscription.AmountMinor, &subscription.NextDueDate); err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, subscription)
	}
	return subscriptions, rows.Err()
}

func (r *Repository) upcomingInstallments(ctx context.Context, userID string, month time.Time, nextMonth time.Time) ([]UpcomingInstallment, error) {
	rows, err := r.pool.Query(ctx, `
		WITH installment_due AS (
			SELECT id::text, name, monthly_amount_minor, remaining_months,
				make_date(
					EXTRACT(year FROM $2::date)::int,
					EXTRACT(month FROM $2::date)::int,
					LEAST(due_day, EXTRACT(day FROM ($2::date + interval '1 month' - interval '1 day'))::int)
				) AS due_date
			FROM installments
			WHERE user_id = $1 AND status = 'active'
		)
		SELECT id, name, monthly_amount_minor, remaining_months, due_date
		FROM installment_due
		WHERE due_date >= $2 AND due_date < $3
		ORDER BY due_date ASC, name ASC
		LIMIT 10
	`, userID, month, nextMonth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var installments []UpcomingInstallment
	for rows.Next() {
		var installment UpcomingInstallment
		if err := rows.Scan(&installment.ID, &installment.Name, &installment.MonthlyAmountMinor, &installment.RemainingMonths, &installment.DueDate); err != nil {
			return nil, err
		}
		installments = append(installments, installment)
	}
	return installments, rows.Err()
}

func (r *Repository) upcomingDebts(ctx context.Context, userID string, month time.Time, nextMonth time.Time) ([]UpcomingDebt, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, type, counterparty_name, principal_amount_minor - paid_amount_minor AS remaining_amount_minor, due_date
		FROM debts
		WHERE user_id = $1
			AND status IN ('open', 'partial')
			AND due_date IS NOT NULL
			AND due_date >= $2
			AND due_date < $3
		ORDER BY due_date ASC, created_at DESC
		LIMIT 10
	`, userID, month, nextMonth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var debts []UpcomingDebt
	for rows.Next() {
		var debt UpcomingDebt
		if err := rows.Scan(&debt.ID, &debt.Type, &debt.CounterpartyName, &debt.RemainingAmountMinor, &debt.DueDate); err != nil {
			return nil, err
		}
		debts = append(debts, debt)
	}
	return debts, rows.Err()
}

func (r *Repository) CashflowTrend(ctx context.Context, userID string, months int) ([]CashflowTrend, error) {
	// Generate months from N months ago to current month
	// Group transactions by month
	query := `
		WITH RECURSIVE months AS (
			SELECT date_trunc('month', CURRENT_DATE - interval '1 month' * ($2 - 1)) AS m
			UNION ALL
			SELECT m + interval '1 month'
			FROM months
			WHERE m < date_trunc('month', CURRENT_DATE)
		)
		SELECT 
			to_char(m.m, 'YYYY-MM') AS month,
			COALESCE(SUM(t.amount_minor) FILTER (WHERE t.type = 'income'), 0) AS income_minor,
			COALESCE(SUM(t.amount_minor) FILTER (WHERE t.type = 'expense'), 0) AS expense_minor
		FROM months m
		LEFT JOIN transactions t 
			ON t.user_id = $1 
			AND date_trunc('month', t.transaction_at AT TIME ZONE 'UTC') = m.m
		GROUP BY m.m
		ORDER BY m.m ASC
	`
	rows, err := r.pool.Query(ctx, query, userID, months)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []CashflowTrend
	for rows.Next() {
		var t CashflowTrend
		if err := rows.Scan(&t.Month, &t.IncomeMinor, &t.ExpenseMinor); err != nil {
			return nil, err
		}
		t.CashflowMinor = t.IncomeMinor - t.ExpenseMinor
		trends = append(trends, t)
	}
	return trends, rows.Err()
}

func (r *Repository) ExpenseDistribution(ctx context.Context, userID string, month time.Time) ([]ExpenseDistribution, error) {
	nextMonth := month.AddDate(0, 1, 0)
	query := `
		WITH category_spend AS (
			SELECT 
				COALESCE(c.id::text, '') AS category_id,
				COALESCE(c.name, 'Uncategorized') AS category_name,
				SUM(t.amount_minor) AS amount_minor
			FROM transactions t
			LEFT JOIN categories c ON t.category_id = c.id
			WHERE t.user_id = $1 
				AND t.type = 'expense'
				AND t.transaction_at >= $2 
				AND t.transaction_at < $3
			GROUP BY c.id, c.name
		), total_spend AS (
			SELECT SUM(amount_minor) AS total FROM category_spend
		)
		SELECT 
			cs.category_id, 
			cs.category_name, 
			cs.amount_minor, 
			CASE WHEN ts.total = 0 THEN 0 ELSE (cs.amount_minor::numeric / ts.total::numeric) * 100 END AS percentage
		FROM category_spend cs, total_spend ts
		ORDER BY cs.amount_minor DESC
	`
	rows, err := r.pool.Query(ctx, query, userID, month, nextMonth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dist []ExpenseDistribution
	for rows.Next() {
		var d ExpenseDistribution
		if err := rows.Scan(&d.CategoryID, &d.CategoryName, &d.AmountMinor, &d.Percentage); err != nil {
			return nil, err
		}
		dist = append(dist, d)
	}
	return dist, rows.Err()
}

func (r *Repository) Forecast(ctx context.Context, userID string, month time.Time) (Forecast, error) {
	nextMonth := month.AddDate(0, 1, 0)

	// Get total spent in this month
	var currentSpent int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount_minor), 0)
		FROM transactions
		WHERE user_id = $1 AND type = 'expense'
			AND transaction_at >= $2 AND transaction_at < $3
	`, userID, month, nextMonth).Scan(&currentSpent)
	if err != nil {
		return Forecast{}, err
	}

	// Get total budget for this month
	budget, err := r.budgetSummary(ctx, userID, month)
	if err != nil {
		return Forecast{}, err
	}

	// Calculate forecast
	now := time.Now().UTC()
	var daysElapsed int
	var totalDaysInMonth int

	// If calculating for a past month, days elapsed = total days in month
	if now.Year() > month.Year() || (now.Year() == month.Year() && now.Month() > month.Month()) {
		daysElapsed = nextMonth.AddDate(0, 0, -1).Day()
		totalDaysInMonth = daysElapsed
	} else if now.Year() == month.Year() && now.Month() == month.Month() {
		daysElapsed = now.Day()
		totalDaysInMonth = nextMonth.AddDate(0, 0, -1).Day()
	} else {
		// Future month
		daysElapsed = 1
		totalDaysInMonth = nextMonth.AddDate(0, 0, -1).Day()
	}

	if daysElapsed == 0 {
		daysElapsed = 1
	}

	dailyAverage := currentSpent / int64(daysElapsed)
	forecastedSpend := dailyAverage * int64(totalDaysInMonth)

	status := "safe"
	if forecastedSpend > budget.LimitMinor {
		status = "overbudget"
	}

	return Forecast{
		CurrentExpenseMinor:    currentSpent,
		DailyAverageMinor:      dailyAverage,
		ForecastedExpenseMinor: forecastedSpend,
		BudgetLimitMinor:       budget.LimitMinor,
		Status:                 status,
	}, nil
}
