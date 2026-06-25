package splitbill

import (
	"context"
	"errors"

	"affluena-api/internal/page"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides read access to split bills. A split bill is an expense
// transaction that originated one or more receivable debts; the link is
// debts.origination_transaction_id -> transactions.id. There is no dedicated
// split_bills table — the group is reconstructed by aggregating those debts.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// remainingExpr sums the outstanding principal for debts still owed.
const remainingExpr = `COALESCE(SUM(CASE WHEN d.status IN ('open','partial') THEN d.principal_amount_minor - d.paid_amount_minor ELSE 0 END), 0)`

// havingForStatus returns the HAVING clause for a validated status filter.
func havingForStatus(status string) string {
	switch status {
	case SplitBillStatusOngoing:
		return "HAVING bool_or(d.status IN ('open','partial'))"
	case SplitBillStatusSettled:
		return "HAVING NOT bool_or(d.status IN ('open','partial'))"
	default:
		return ""
	}
}

// List returns split bills aggregated per origination transaction, newest first.
// status is "" (all), "ongoing", or "settled".
func (r *Repository) List(ctx context.Context, userID string, status string, pagination page.Params) (page.Result[SplitBillSummary], error) {
	having := havingForStatus(status)

	rows, err := r.pool.Query(ctx, `
		SELECT t.id::text, t.note, t.amount_minor, t.transaction_at,
			COUNT(d.id) AS participant_count,
			COUNT(*) FILTER (WHERE d.status IN ('paid_off','cancelled')) AS settled_count,
			COALESCE(SUM(d.principal_amount_minor), 0) AS total_owed,
			`+remainingExpr+` AS total_remaining,
			bool_or(d.status IN ('open','partial')) AS ongoing
		FROM transactions t
		JOIN debts d ON d.origination_transaction_id = t.id
		WHERE d.user_id = $1
		GROUP BY t.id, t.note, t.amount_minor, t.transaction_at
		`+having+`
		ORDER BY t.transaction_at DESC
		LIMIT $2 OFFSET $3
	`, userID, pagination.Limit, pagination.Offset)
	if err != nil {
		return page.Result[SplitBillSummary]{}, err
	}
	defer rows.Close()

	var items []SplitBillSummary
	for rows.Next() {
		var s SplitBillSummary
		var ongoing bool
		if err := rows.Scan(&s.TransactionID, &s.Note, &s.TotalAmountMinor, &s.TransactionAt,
			&s.ParticipantCount, &s.SettledCount, &s.TotalOwedMinor, &s.TotalRemainingMinor, &ongoing); err != nil {
			return page.Result[SplitBillSummary]{}, err
		}
		s.Status = statusLabel(ongoing)
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return page.Result[SplitBillSummary]{}, err
	}

	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			SELECT t.id
			FROM transactions t
			JOIN debts d ON d.origination_transaction_id = t.id
			WHERE d.user_id = $1
			GROUP BY t.id
			`+having+`
		) grouped
	`, userID).Scan(&total); err != nil {
		return page.Result[SplitBillSummary]{}, err
	}

	return page.NewResult(items, pagination, total), nil
}

// Get returns one split bill (origination transaction + participant debts).
func (r *Repository) Get(ctx context.Context, userID string, transactionID string) (SplitBillDetail, error) {
	var detail SplitBillDetail
	var ongoing bool
	err := r.pool.QueryRow(ctx, `
		SELECT t.id::text, t.note, t.amount_minor, t.transaction_at,
			COUNT(d.id) AS participant_count,
			COUNT(*) FILTER (WHERE d.status IN ('paid_off','cancelled')) AS settled_count,
			COALESCE(SUM(d.principal_amount_minor), 0) AS total_owed,
			`+remainingExpr+` AS total_remaining,
			bool_or(d.status IN ('open','partial')) AS ongoing
		FROM transactions t
		JOIN debts d ON d.origination_transaction_id = t.id
		WHERE d.user_id = $1 AND t.id = $2::uuid
		GROUP BY t.id, t.note, t.amount_minor, t.transaction_at
	`, userID, transactionID).Scan(&detail.TransactionID, &detail.Note, &detail.TotalAmountMinor, &detail.TransactionAt,
		&detail.ParticipantCount, &detail.SettledCount, &detail.TotalOwedMinor, &detail.TotalRemainingMinor, &ongoing)
	if errors.Is(err, pgx.ErrNoRows) {
		return SplitBillDetail{}, ErrNotFound
	}
	if err != nil {
		return SplitBillDetail{}, err
	}
	detail.Status = statusLabel(ongoing)

	rows, err := r.pool.Query(ctx, `
		SELECT d.id::text, d.counterparty_name, d.principal_amount_minor, d.paid_amount_minor,
			(d.principal_amount_minor - d.paid_amount_minor) AS remaining, d.status, d.due_date
		FROM debts d
		WHERE d.user_id = $1 AND d.origination_transaction_id = $2::uuid
		ORDER BY d.created_at ASC
	`, userID, transactionID)
	if err != nil {
		return SplitBillDetail{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var p SplitBillParticipant
		if err := rows.Scan(&p.DebtID, &p.CounterpartyName, &p.PrincipalAmountMinor, &p.PaidAmountMinor,
			&p.RemainingAmountMinor, &p.Status, &p.DueDate); err != nil {
			return SplitBillDetail{}, err
		}
		if p.Status == "cancelled" {
			p.RemainingAmountMinor = 0
		}
		detail.Participants = append(detail.Participants, p)
	}
	if err := rows.Err(); err != nil {
		return SplitBillDetail{}, err
	}
	return detail, nil
}

func statusLabel(ongoing bool) string {
	if ongoing {
		return SplitBillStatusOngoing
	}
	return SplitBillStatusSettled
}
