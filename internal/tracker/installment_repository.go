package tracker

import (
	"context"
	"errors"
	"time"

	"affluena/internal/transaction"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Installment struct {
	ID                 string            `json:"id"`
	UserID             string            `json:"user_id"`
	Name               string            `json:"name"`
	WalletID           string            `json:"wallet_id"`
	CategoryID         string            `json:"category_id"`
	TotalAmountMinor   int64             `json:"total_amount_minor"`
	MonthlyAmountMinor int64             `json:"monthly_amount_minor"`
	TenorMonths        int               `json:"tenor_months"`
	RemainingMonths    int               `json:"remaining_months"`
	StartDate          time.Time         `json:"start_date"`
	DueDay             int               `json:"due_day"`
	Status             InstallmentStatus `json:"status"`
	Note               string            `json:"note"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

type InstallmentPayment struct {
	Installment Installment             `json:"installment"`
	Transaction transaction.Transaction `json:"transaction"`
}

type InstallmentRepository struct {
	pool            *pgxpool.Pool
	transactionRepo *transaction.Repository
}

func NewInstallmentRepository(pool *pgxpool.Pool, transactionRepo *transaction.Repository) *InstallmentRepository {
	return &InstallmentRepository{pool: pool, transactionRepo: transactionRepo}
}

func (r *InstallmentRepository) Create(ctx context.Context, userID string, installment Installment) (Installment, error) {
	if err := ensureExpensePaymentRefs(ctx, r.pool, userID, installment.WalletID, installment.CategoryID); err != nil {
		return Installment{}, err
	}
	if installment.RemainingMonths == 0 {
		installment.RemainingMonths = installment.TenorMonths
	}
	if installment.Status == "" {
		installment.Status = InstallmentStatusActive
	}

	return scanInstallment(r.pool.QueryRow(ctx, `
		INSERT INTO installments (
			user_id, name, wallet_id, category_id, total_amount_minor, monthly_amount_minor,
			tenor_months, remaining_months, start_date, due_day, status, note
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id::text, user_id::text, name, wallet_id::text, category_id::text,
			total_amount_minor, monthly_amount_minor, tenor_months, remaining_months,
			start_date, due_day, status, note, created_at, updated_at
	`, userID, installment.Name, installment.WalletID, installment.CategoryID, installment.TotalAmountMinor,
		installment.MonthlyAmountMinor, installment.TenorMonths, installment.RemainingMonths,
		installment.StartDate, installment.DueDay, installment.Status, installment.Note))
}

func (r *InstallmentRepository) List(ctx context.Context, userID string) ([]Installment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, user_id::text, name, wallet_id::text, category_id::text,
			total_amount_minor, monthly_amount_minor, tenor_months, remaining_months,
			start_date, due_day, status, note, created_at, updated_at
		FROM installments
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var installments []Installment
	for rows.Next() {
		installment, err := scanInstallment(rows)
		if err != nil {
			return nil, err
		}
		installments = append(installments, installment)
	}
	return installments, rows.Err()
}

func (r *InstallmentRepository) Get(ctx context.Context, userID string, id string) (Installment, error) {
	return r.get(ctx, r.pool, userID, id, false)
}

func (r *InstallmentRepository) Update(ctx context.Context, userID string, id string, installment Installment) (Installment, error) {
	if err := ensureExpensePaymentRefs(ctx, r.pool, userID, installment.WalletID, installment.CategoryID); err != nil {
		return Installment{}, err
	}

	return scanInstallment(r.pool.QueryRow(ctx, `
		UPDATE installments
		SET name = $3, wallet_id = $4, category_id = $5, total_amount_minor = $6,
			monthly_amount_minor = $7, tenor_months = $8, remaining_months = $9,
			start_date = $10, due_day = $11, status = $12, note = $13, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, name, wallet_id::text, category_id::text,
			total_amount_minor, monthly_amount_minor, tenor_months, remaining_months,
			start_date, due_day, status, note, created_at, updated_at
	`, userID, id, installment.Name, installment.WalletID, installment.CategoryID,
		installment.TotalAmountMinor, installment.MonthlyAmountMinor, installment.TenorMonths,
		installment.RemainingMonths, installment.StartDate, installment.DueDay,
		installment.Status, installment.Note))
}

func (r *InstallmentRepository) Delete(ctx context.Context, userID string, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM installments WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *InstallmentRepository) Pay(ctx context.Context, userID string, id string, paidAt time.Time, note string) (InstallmentPayment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return InstallmentPayment{}, err
	}
	defer tx.Rollback(ctx)

	installment, err := r.get(ctx, tx, userID, id, true)
	if err != nil {
		return InstallmentPayment{}, err
	}
	nextState, err := ApplyInstallmentPayment(InstallmentPaymentState{
		RemainingMonths: installment.RemainingMonths,
		Status:          installment.Status,
	})
	if err != nil {
		return InstallmentPayment{}, err
	}

	paymentNote := note
	if paymentNote == "" {
		paymentNote = "Installment payment: " + installment.Name
	}
	createdTx, err := r.transactionRepo.CreateInTx(ctx, tx, userID, transaction.TransactionInput{
		Type:           transaction.TransactionTypeExpense,
		WalletID:       installment.WalletID,
		CategoryID:     installment.CategoryID,
		AmountMinor:    installment.MonthlyAmountMinor,
		TransactionUTC: paidAt.UTC(),
		Note:           paymentNote,
	})
	if err != nil {
		return InstallmentPayment{}, err
	}

	updated, err := scanInstallment(tx.QueryRow(ctx, `
		UPDATE installments
		SET remaining_months = $3, status = $4, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, name, wallet_id::text, category_id::text,
			total_amount_minor, monthly_amount_minor, tenor_months, remaining_months,
			start_date, due_day, status, note, created_at, updated_at
	`, userID, id, nextState.RemainingMonths, nextState.Status))
	if err != nil {
		return InstallmentPayment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return InstallmentPayment{}, err
	}
	return InstallmentPayment{Installment: updated, Transaction: createdTx}, nil
}

func (r *InstallmentRepository) get(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID string, id string, forUpdate bool) (Installment, error) {
	sql := `
		SELECT id::text, user_id::text, name, wallet_id::text, category_id::text,
			total_amount_minor, monthly_amount_minor, tenor_months, remaining_months,
			start_date, due_day, status, note, created_at, updated_at
		FROM installments
		WHERE user_id = $1 AND id = $2
	`
	if forUpdate {
		sql += ` FOR UPDATE`
	}
	return scanInstallment(q.QueryRow(ctx, sql, userID, id))
}

func scanInstallment(row rowScanner) (Installment, error) {
	var installment Installment
	err := row.Scan(
		&installment.ID,
		&installment.UserID,
		&installment.Name,
		&installment.WalletID,
		&installment.CategoryID,
		&installment.TotalAmountMinor,
		&installment.MonthlyAmountMinor,
		&installment.TenorMonths,
		&installment.RemainingMonths,
		&installment.StartDate,
		&installment.DueDay,
		&installment.Status,
		&installment.Note,
		&installment.CreatedAt,
		&installment.UpdatedAt,
	)
	return installment, err
}

func NotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
