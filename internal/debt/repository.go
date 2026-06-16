package debt

import (
	"context"
	"errors"
	"time"

	"affluena-api/internal/page"
	"affluena-api/internal/transaction"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool            *pgxpool.Pool
	transactionRepo *transaction.Repository
}

func NewRepository(pool *pgxpool.Pool, transactionRepo *transaction.Repository) *Repository {
	return &Repository{pool: pool, transactionRepo: transactionRepo}
}

func (r *Repository) Create(ctx context.Context, userID string, input DebtInput) (Debt, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Debt{}, err
	}
	defer tx.Rollback(ctx)

	debt, err := r.CreateInTx(ctx, tx, userID, input)
	if err != nil {
		return Debt{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Debt{}, err
	}
	return debt, nil
}

func (r *Repository) CreateInTx(ctx context.Context, tx pgx.Tx, userID string, input DebtInput) (Debt, error) {
	if !IsValidDebtType(input.Type) {
		return Debt{}, errInvalidDebtType
	}
	if input.PrincipalAmountMinor <= 0 {
		return Debt{}, errors.New("principal_amount_minor must be positive")
	}
	if input.OpenedAt.IsZero() {
		input.OpenedAt = time.Now().UTC()
	}

	paymentType, err := TransactionTypeFor(input.Type, DebtActionPayment)
	if err != nil {
		return Debt{}, err
	}
	if err := ensureCategory(ctx, tx, userID, input.PaymentCategoryID, string(paymentType)); err != nil {
		return Debt{}, translateNotFound(err)
	}

	txType, err := TransactionTypeFor(input.Type, DebtActionOrigination)
	if err != nil {
		return Debt{}, err
	}
	origination, err := r.transactionRepo.CreateInTx(ctx, tx, userID, transaction.TransactionInput{
		Type:           txType,
		WalletID:       input.WalletID,
		CategoryID:     input.DisbursementCategoryID,
		AmountMinor:    input.PrincipalAmountMinor,
		TransactionUTC: input.OpenedAt.UTC(),
		Note:           originationNote(input.Type, input.CounterpartyName, input.Note),
	})
	if err != nil {
		return Debt{}, err
	}

	debt, err := scanDebt(tx.QueryRow(ctx, `
		INSERT INTO debts (
			user_id, type, counterparty_name, wallet_id, disbursement_category_id,
			payment_category_id, origination_transaction_id, principal_amount_minor,
			paid_amount_minor, opened_at, due_date, status, note
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, $9, $10, $11, $12)
		RETURNING id::text, user_id::text, type, counterparty_name, wallet_id::text,
			disbursement_category_id::text, payment_category_id::text,
			origination_transaction_id::text, principal_amount_minor, paid_amount_minor,
			(principal_amount_minor - paid_amount_minor), opened_at, due_date, status, note,
			created_at, updated_at
	`, userID, input.Type, input.CounterpartyName, input.WalletID, input.DisbursementCategoryID,
		input.PaymentCategoryID, origination.ID, input.PrincipalAmountMinor, input.OpenedAt.UTC(),
		nullableDate(input.DueDate), DebtStatusOpen, input.Note))
	if err != nil {
		return Debt{}, translateNotFound(err)
	}
	return debt, nil
}

func (r *Repository) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Debt], error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, user_id::text, type, counterparty_name, wallet_id::text,
			disbursement_category_id::text, payment_category_id::text,
			origination_transaction_id::text, principal_amount_minor, paid_amount_minor,
			(principal_amount_minor - paid_amount_minor), opened_at, due_date, status, note,
			created_at, updated_at
		FROM debts
		WHERE user_id = $1
		ORDER BY `+debtOrderBy(pagination.Sort)+`
		LIMIT $2 OFFSET $3
	`, userID, pagination.Limit, pagination.Offset)
	if err != nil {
		return page.Result[Debt]{}, err
	}
	defer rows.Close()

	var debts []Debt
	for rows.Next() {
		debt, err := scanDebt(rows)
		if err != nil {
			return page.Result[Debt]{}, err
		}
		debts = append(debts, debt)
	}
	if err := rows.Err(); err != nil {
		return page.Result[Debt]{}, err
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM debts WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return page.Result[Debt]{}, err
	}
	return page.NewResult(debts, pagination, total), nil
}

func debtOrderBy(sort string) string {
	switch sort {
	case "opened_at_asc":
		return "opened_at ASC, created_at ASC"
	case "due_date_asc":
		return "due_date ASC NULLS LAST, created_at DESC"
	case "due_date_desc":
		return "due_date DESC NULLS LAST, created_at DESC"
	case "amount_desc":
		return "principal_amount_minor DESC, opened_at DESC"
	case "amount_asc":
		return "principal_amount_minor ASC, opened_at DESC"
	default:
		return "opened_at DESC, created_at DESC"
	}
}

func (r *Repository) Get(ctx context.Context, userID string, id string) (Debt, error) {
	debt, err := r.get(ctx, r.pool, userID, id, false)
	return debt, translateNotFound(err)
}

func (r *Repository) Update(ctx context.Context, userID string, id string, update DebtUpdate) (Debt, error) {
	current, err := r.get(ctx, r.pool, userID, id, false)
	if err != nil {
		return Debt{}, translateNotFound(err)
	}
	if update.Status == "" {
		update.Status = current.Status
	}
	if !IsValidDebtStatus(update.Status) {
		return Debt{}, errInvalidDebtStatus
	}
	if update.Status != DebtStatusCancelled {
		if _, err := ResolveStatus(current.PrincipalAmountMinor, current.PaidAmountMinor, update.Status); err != nil {
			return Debt{}, err
		}
	}
	debt, err := scanDebt(r.pool.QueryRow(ctx, `
		UPDATE debts
		SET counterparty_name = $3, due_date = $4, status = $5, note = $6, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, type, counterparty_name, wallet_id::text,
			disbursement_category_id::text, payment_category_id::text,
			origination_transaction_id::text, principal_amount_minor, paid_amount_minor,
			(principal_amount_minor - paid_amount_minor), opened_at, due_date, status, note,
			created_at, updated_at
	`, userID, id, update.CounterpartyName, nullableDate(update.DueDate), update.Status, update.Note))
	return debt, translateNotFound(err)
}

// Delete performs a soft-cancel by setting status to 'cancelled'.
// This preserves the audit trail including the original transaction.
// The debt remains visible in history but is marked as cancelled.
func (r *Repository) Delete(ctx context.Context, userID string, id string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE debts
		SET status = $3, updated_at = now()
		WHERE user_id = $1 AND id = $2
	`, userID, id, DebtStatusCancelled)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Pay(ctx context.Context, userID string, id string, amountMinor int64, paidAt time.Time, note string) (DebtPayment, error) {
	if paidAt.IsZero() {
		paidAt = time.Now().UTC()
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return DebtPayment{}, err
	}
	defer tx.Rollback(ctx)

	current, err := r.get(ctx, tx, userID, id, true)
	if err != nil {
		return DebtPayment{}, translateNotFound(err)
	}
	next, err := ApplyPayment(PaymentState{
		PrincipalAmountMinor: current.PrincipalAmountMinor,
		PaidAmountMinor:      current.PaidAmountMinor,
		Status:               current.Status,
	}, amountMinor)
	if err != nil {
		return DebtPayment{}, err
	}

	txType, err := TransactionTypeFor(current.Type, DebtActionPayment)
	if err != nil {
		return DebtPayment{}, err
	}
	createdTx, err := r.transactionRepo.CreateInTx(ctx, tx, userID, transaction.TransactionInput{
		Type:           txType,
		WalletID:       current.WalletID,
		CategoryID:     current.PaymentCategoryID,
		AmountMinor:    amountMinor,
		TransactionUTC: paidAt.UTC(),
		Note:           paymentNote(current.Type, current.CounterpartyName, note),
	})
	if err != nil {
		return DebtPayment{}, err
	}

	payment, err := scanDebtPayment(tx.QueryRow(ctx, `
		INSERT INTO debt_payments (user_id, debt_id, transaction_id, amount_minor, paid_at, note)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, user_id::text, debt_id::text, transaction_id::text,
			amount_minor, paid_at, note, created_at
	`, userID, id, createdTx.ID, amountMinor, paidAt.UTC(), note))
	if err != nil {
		return DebtPayment{}, err
	}

	updated, err := scanDebt(tx.QueryRow(ctx, `
		UPDATE debts
		SET paid_amount_minor = $3, status = $4, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, type, counterparty_name, wallet_id::text,
			disbursement_category_id::text, payment_category_id::text,
			origination_transaction_id::text, principal_amount_minor, paid_amount_minor,
			(principal_amount_minor - paid_amount_minor), opened_at, due_date, status, note,
			created_at, updated_at
	`, userID, id, next.PaidAmountMinor, next.Status))
	if err != nil {
		return DebtPayment{}, translateNotFound(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return DebtPayment{}, err
	}

	payment.Debt = updated
	payment.Transaction = createdTx
	return payment, nil
}

func (r *Repository) get(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID string, id string, forUpdate bool) (Debt, error) {
	sql := `
		SELECT id::text, user_id::text, type, counterparty_name, wallet_id::text,
			disbursement_category_id::text, payment_category_id::text,
			origination_transaction_id::text, principal_amount_minor, paid_amount_minor,
			(principal_amount_minor - paid_amount_minor), opened_at, due_date, status, note,
			created_at, updated_at
		FROM debts
		WHERE user_id = $1 AND id = $2
	`
	if forUpdate {
		sql += ` FOR UPDATE`
	}
	return scanDebt(q.QueryRow(ctx, sql, userID, id))
}

func scanDebt(row rowScanner) (Debt, error) {
	var debt Debt
	var dueDate pgtype.Date
	err := row.Scan(
		&debt.ID,
		&debt.UserID,
		&debt.Type,
		&debt.CounterpartyName,
		&debt.WalletID,
		&debt.DisbursementCategoryID,
		&debt.PaymentCategoryID,
		&debt.OriginationTransactionID,
		&debt.PrincipalAmountMinor,
		&debt.PaidAmountMinor,
		&debt.RemainingAmountMinor,
		&debt.OpenedAt,
		&dueDate,
		&debt.Status,
		&debt.Note,
		&debt.CreatedAt,
		&debt.UpdatedAt,
	)
	if err != nil {
		return Debt{}, err
	}
	if dueDate.Valid {
		debt.DueDate = &dueDate.Time
	}
	return debt, nil
}

func scanDebtPayment(row rowScanner) (DebtPayment, error) {
	var payment DebtPayment
	err := row.Scan(
		&payment.ID,
		&payment.UserID,
		&payment.DebtID,
		&payment.TransactionID,
		&payment.AmountMinor,
		&payment.PaidAt,
		&payment.Note,
		&payment.CreatedAt,
	)
	return payment, err
}

func ensureCategory(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID string, categoryID string, transactionType string) error {
	categoryType := transactionType
	if transactionType != "income" {
		categoryType = "expense"
	}

	var exists bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM categories
			WHERE user_id = $1 AND id = $2 AND type = $3
		)
	`, userID, categoryID, categoryType).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return pgx.ErrNoRows
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func nullableDate(value *time.Time) any {
	if value == nil {
		return nil
	}
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func originationNote(debtType DebtType, counterpartyName string, note string) string {
	if note != "" {
		return note
	}
	if debtType == DebtTypeReceivable {
		return "Loan given to " + counterpartyName
	}
	return "Loan received from " + counterpartyName
}

func paymentNote(debtType DebtType, counterpartyName string, note string) string {
	if note != "" {
		return note
	}
	if debtType == DebtTypeReceivable {
		return "Loan repayment from " + counterpartyName
	}
	return "Loan repayment to " + counterpartyName
}

func translateNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
