package alert

import (
	"context"
	"time"

	"affluena-api/internal/activity"
	"affluena-api/internal/debt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FeedRepository interface {
	ListOverdueDebts(ctx context.Context, userID string) ([]debt.Debt, error)
	ListRecentRecurringActivities(ctx context.Context, userID string, since time.Time) ([]activity.Activity, error)
}

type feedRepository struct {
	pool *pgxpool.Pool
}

func NewFeedRepository(pool *pgxpool.Pool) FeedRepository {
	return &feedRepository{pool: pool}
}

func (r *feedRepository) ListOverdueDebts(ctx context.Context, userID string) ([]debt.Debt, error) {
	query := `
		SELECT id, user_id, type, counterparty_name, wallet_id, disbursement_category_id, payment_category_id,
		       origination_transaction_id, principal_amount_minor, paid_amount_minor, (principal_amount_minor - paid_amount_minor),
		       opened_at, due_date, status, note, created_at, updated_at
		FROM debts
		WHERE user_id = $1
		  AND type = 'payable'
		  AND status IN ('open', 'partial')
		  AND due_date < now()
		ORDER BY due_date ASC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var debts []debt.Debt
	for rows.Next() {
		var d debt.Debt
		err := rows.Scan(
			&d.ID, &d.UserID, &d.Type, &d.CounterpartyName, &d.WalletID, &d.DisbursementCategoryID, &d.PaymentCategoryID,
			&d.OriginationTransactionID, &d.PrincipalAmountMinor, &d.PaidAmountMinor, &d.RemainingAmountMinor,
			&d.OpenedAt, &d.DueDate, &d.Status, &d.Note, &d.CreatedAt, &d.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		debts = append(debts, d)
	}
	return debts, rows.Err()
}

func (r *feedRepository) ListRecentRecurringActivities(ctx context.Context, userID string, since time.Time) ([]activity.Activity, error) {
	query := `
		SELECT id, user_id, action_type, entity_type, entity_id, description, created_at
		FROM user_activities
		WHERE user_id = $1
		  AND entity_type = 'RECURRING'
		  AND action_type IN ('RUN', 'RUN_MANUAL', 'RUN_FAILED')
		  AND created_at > $2
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []activity.Activity
	for rows.Next() {
		var a activity.Activity
		err := rows.Scan(
			&a.ID, &a.UserID, &a.ActionType, &a.EntityType, &a.EntityID, &a.Description, &a.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		activities = append(activities, a)
	}
	return activities, rows.Err()
}
