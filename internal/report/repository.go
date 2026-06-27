package report

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

type CategoryAggregation struct {
	CategoryName string
	AmountMinor  int64
	WalletName   string
}

func (r *Repository) GetIncomeExpenseAggregation(ctx context.Context, userID string, txType string, from, to time.Time) ([]CategoryAggregation, error) {
	query := `
		SELECT 
			COALESCE(c.name, 'Uncategorized') as category_name,
			SUM(t.amount_minor) as amount_minor,
			COALESCE(MAX(w.name), 'All wallets') as wallet_name
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		LEFT JOIN wallets w ON t.wallet_id = w.id
		WHERE (
			t.user_id = $1 OR
			EXISTS (SELECT 1 FROM wallets aw WHERE aw.id = t.wallet_id AND aw.user_id = $1) OR
			EXISTS (SELECT 1 FROM wallets atw WHERE atw.id = t.to_wallet_id AND atw.user_id = $1) OR
			t.wallet_id IN (SELECT wallet_id FROM wallet_shares WHERE user_id = $1 AND status = 'joined') OR
			t.to_wallet_id IN (SELECT wallet_id FROM wallet_shares WHERE user_id = $1 AND status = 'joined')
		)
		AND t.type = $2
		AND t.transaction_at >= $3
		AND t.transaction_at < $4
		GROUP BY c.name
		ORDER BY amount_minor DESC
	`

	rows, err := r.pool.Query(ctx, query, userID, txType, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CategoryAggregation
	for rows.Next() {
		var agg CategoryAggregation
		if err := rows.Scan(&agg.CategoryName, &agg.AmountMinor, &agg.WalletName); err != nil {
			return nil, err
		}
		result = append(result, agg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []CategoryAggregation{}
	}

	return result, nil
}

type WeeklyCashflow struct {
	Week         int
	IncomeMinor  int64
	ExpenseMinor int64
}

func (r *Repository) GetWeeklyCashflow(ctx context.Context, userID string, from, to time.Time) ([]WeeklyCashflow, error) {
	query := `
		SELECT 
			EXTRACT(WEEK FROM t.transaction_at) as week_num,
			SUM(CASE WHEN t.type = 'income' THEN t.amount_minor ELSE 0 END) as income_minor,
			SUM(CASE WHEN t.type = 'expense' THEN t.amount_minor ELSE 0 END) as expense_minor
		FROM transactions t
		WHERE (
			t.user_id = $1 OR
			EXISTS (SELECT 1 FROM wallets aw WHERE aw.id = t.wallet_id AND aw.user_id = $1) OR
			EXISTS (SELECT 1 FROM wallets atw WHERE atw.id = t.to_wallet_id AND atw.user_id = $1) OR
			t.wallet_id IN (SELECT wallet_id FROM wallet_shares WHERE user_id = $1 AND status = 'joined') OR
			t.to_wallet_id IN (SELECT wallet_id FROM wallet_shares WHERE user_id = $1 AND status = 'joined')
		)
		AND t.type IN ('income', 'expense')
		AND t.transaction_at >= $2
		AND t.transaction_at < $3
		GROUP BY week_num
		ORDER BY week_num ASC
	`

	rows, err := r.pool.Query(ctx, query, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []WeeklyCashflow
	for rows.Next() {
		var agg WeeklyCashflow
		if err := rows.Scan(&agg.Week, &agg.IncomeMinor, &agg.ExpenseMinor); err != nil {
			return nil, err
		}
		result = append(result, agg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []WeeklyCashflow{}
	}

	return result, nil
}

type DebtAggregation struct {
	ID                   string
	CounterpartyName     string
	Type                 string
	PrincipalAmountMinor int64
	PaidAmountMinor      int64
	WalletName           string
	Status               string
}

func (r *Repository) GetDebts(ctx context.Context, userID string) ([]DebtAggregation, error) {
	query := `
		SELECT 
			d.id::text,
			d.counterparty_name,
			d.type,
			d.principal_amount_minor,
			d.paid_amount_minor,
			COALESCE(w.name, 'Unknown wallet') as wallet_name,
			d.status
		FROM debts d
		LEFT JOIN wallets w ON d.wallet_id = w.id
		WHERE d.user_id = $1
		ORDER BY d.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DebtAggregation
	for rows.Next() {
		var agg DebtAggregation
		if err := rows.Scan(&agg.ID, &agg.CounterpartyName, &agg.Type, &agg.PrincipalAmountMinor, &agg.PaidAmountMinor, &agg.WalletName, &agg.Status); err != nil {
			return nil, err
		}
		result = append(result, agg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []DebtAggregation{}
	}

	return result, nil
}

type GoalAggregation struct {
	ID                string
	Name              string
	TargetAmountMinor int64
	BalanceMinor      int64
	Status            string
	WalletName        string
}

func (r *Repository) GetGoals(ctx context.Context, userID string) ([]GoalAggregation, error) {
	query := `
		SELECT 
			g.id::text,
			g.name,
			g.target_amount_minor,
			COALESCE(SUM(w.balance_minor), 0) as balance_minor,
			g.status,
			COALESCE(MAX(w.name), 'Goal wallet') as wallet_name
		FROM goals g
		LEFT JOIN goal_members gm ON g.id = gm.goal_id
		LEFT JOIN wallets w ON w.goal_id = g.id
		WHERE g.user_id = $1 OR (gm.user_id = $1 AND gm.status = 'joined')
		GROUP BY g.id, g.name, g.target_amount_minor, g.status
		ORDER BY g.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []GoalAggregation
	for rows.Next() {
		var agg GoalAggregation
		if err := rows.Scan(&agg.ID, &agg.Name, &agg.TargetAmountMinor, &agg.BalanceMinor, &agg.Status, &agg.WalletName); err != nil {
			return nil, err
		}
		result = append(result, agg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []GoalAggregation{}
	}

	return result, nil
}
