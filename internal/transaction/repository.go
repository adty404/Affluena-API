package transaction

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, userID string, input TransactionInput) (Transaction, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Transaction{}, err
	}
	defer tx.Rollback(ctx)

	transaction, err := r.CreateInTx(ctx, tx, userID, input)
	if err != nil {
		return Transaction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Transaction{}, err
	}
	return transaction, nil
}

func (r *Repository) CreateInTx(ctx context.Context, tx pgx.Tx, userID string, input TransactionInput) (Transaction, error) {
	if err := r.ensureRefs(ctx, tx, userID, input); err != nil {
		return Transaction{}, translateNotFound(err)
	}
	deltas, err := BalanceDeltas(input)
	if err != nil {
		return Transaction{}, err
	}
	if err := applyDeltas(ctx, tx, userID, deltas); err != nil {
		return Transaction{}, translateNotFound(err)
	}
	return insertTransaction(ctx, tx, userID, input)
}

func (r *Repository) List(ctx context.Context, userID string) ([]Transaction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, user_id::text, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, transaction_at, note, created_at, updated_at
		FROM transactions
		WHERE user_id = $1
		ORDER BY transaction_at DESC, created_at DESC
		LIMIT 100
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	return transactions, rows.Err()
}

func (r *Repository) Get(ctx context.Context, userID string, id string) (Transaction, error) {
	transaction, err := r.get(ctx, r.pool, userID, id, false)
	return transaction, translateNotFound(err)
}

func (r *Repository) Update(ctx context.Context, userID string, id string, input TransactionInput) (Transaction, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Transaction{}, err
	}
	defer tx.Rollback(ctx)

	oldTransaction, err := r.get(ctx, tx, userID, id, true)
	if err != nil {
		return Transaction{}, translateNotFound(err)
	}
	oldDeltas, err := BalanceDeltas(oldTransaction.Input())
	if err != nil {
		return Transaction{}, err
	}
	if err := applyDeltas(ctx, tx, userID, reverseDeltas(oldDeltas)); err != nil {
		return Transaction{}, translateNotFound(err)
	}

	if err := r.ensureRefs(ctx, tx, userID, input); err != nil {
		return Transaction{}, translateNotFound(err)
	}
	newDeltas, err := BalanceDeltas(input)
	if err != nil {
		return Transaction{}, err
	}
	if err := applyDeltas(ctx, tx, userID, newDeltas); err != nil {
		return Transaction{}, translateNotFound(err)
	}

	transaction, err := updateTransaction(ctx, tx, userID, id, input)
	if err != nil {
		return Transaction{}, translateNotFound(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Transaction{}, err
	}
	return transaction, nil
}

func (r *Repository) Delete(ctx context.Context, userID string, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	oldTransaction, err := r.get(ctx, tx, userID, id, true)
	if err != nil {
		return translateNotFound(err)
	}
	oldDeltas, err := BalanceDeltas(oldTransaction.Input())
	if err != nil {
		return err
	}
	if err := applyDeltas(ctx, tx, userID, reverseDeltas(oldDeltas)); err != nil {
		return translateNotFound(err)
	}

	tag, err := tx.Exec(ctx, `DELETE FROM transactions WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

type queryer interface {
	Exec(context.Context, string, ...any) (pgconnCommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type pgconnCommandTag interface {
	RowsAffected() int64
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *Repository) get(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID string, id string, forUpdate bool) (Transaction, error) {
	sql := `
		SELECT id::text, user_id::text, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, transaction_at, note, created_at, updated_at
		FROM transactions
		WHERE user_id = $1 AND id = $2
	`
	if forUpdate {
		sql += ` FOR UPDATE`
	}
	return scanTransaction(q.QueryRow(ctx, sql, userID, id))
}

func (r *Repository) ensureRefs(ctx context.Context, tx pgx.Tx, userID string, input TransactionInput) error {
	if err := ensureWallet(ctx, tx, userID, input.WalletID); err != nil {
		return err
	}
	if input.ToWalletID != "" {
		if err := ensureWallet(ctx, tx, userID, input.ToWalletID); err != nil {
			return err
		}
	}
	if input.CategoryID != "" {
		return ensureCategory(ctx, tx, userID, input.CategoryID, string(input.Type))
	}
	return nil
}

func insertTransaction(ctx context.Context, tx pgx.Tx, userID string, input TransactionInput) (Transaction, error) {
	return scanTransaction(tx.QueryRow(ctx, `
		INSERT INTO transactions (user_id, type, wallet_id, to_wallet_id, category_id, amount_minor, transaction_at, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text, user_id::text, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, transaction_at, note, created_at, updated_at
	`, userID, input.Type, input.WalletID, nullableUUID(input.ToWalletID), nullableUUID(input.CategoryID), input.AmountMinor, input.TransactionUTC, input.Note))
}

func updateTransaction(ctx context.Context, tx pgx.Tx, userID string, id string, input TransactionInput) (Transaction, error) {
	return scanTransaction(tx.QueryRow(ctx, `
		UPDATE transactions
		SET type = $3, wallet_id = $4, to_wallet_id = $5, category_id = $6, amount_minor = $7,
			transaction_at = $8, note = $9, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, transaction_at, note, created_at, updated_at
	`, userID, id, input.Type, input.WalletID, nullableUUID(input.ToWalletID), nullableUUID(input.CategoryID), input.AmountMinor, input.TransactionUTC, input.Note))
}

func applyDeltas(ctx context.Context, tx pgx.Tx, userID string, deltas []BalanceDelta) error {
	for _, delta := range deltas {
		tag, err := tx.Exec(ctx, `
			UPDATE wallets
			SET balance_minor = balance_minor + $1, updated_at = now()
			WHERE user_id = $2 AND id = $3
		`, delta.AmountMinor, userID, delta.WalletID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
	}
	return nil
}

func reverseDeltas(deltas []BalanceDelta) []BalanceDelta {
	reversed := make([]BalanceDelta, 0, len(deltas))
	for _, delta := range deltas {
		reversed = append(reversed, BalanceDelta{WalletID: delta.WalletID, AmountMinor: -delta.AmountMinor})
	}
	return reversed
}

func ensureWallet(ctx context.Context, tx pgx.Tx, userID string, walletID string) error {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM wallets WHERE user_id = $1 AND id = $2)`, userID, walletID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

func ensureCategory(ctx context.Context, tx pgx.Tx, userID string, categoryID string, transactionType string) error {
	categoryType := transactionType
	if transactionType != "income" {
		categoryType = "expense"
	}

	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM categories
			WHERE user_id = $1 AND id = $2 AND type = $3
		)
	`, userID, categoryID, categoryType).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

func scanTransaction(row rowScanner) (Transaction, error) {
	var transaction Transaction
	err := row.Scan(
		&transaction.ID,
		&transaction.UserID,
		&transaction.Type,
		&transaction.WalletID,
		&transaction.ToWalletID,
		&transaction.CategoryID,
		&transaction.AmountMinor,
		&transaction.TransactionAt,
		&transaction.Note,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
	)
	return transaction, err
}

func nullableUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func translateNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
