package transaction

import (
	"context"
	"errors"
	"time"

	"affluena-api/internal/page"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Transaction permission rules:
// - View: Anyone with wallet access (owner or joined member) can view transactions
// - Create: Any user with wallet access can create transactions on shared wallets
// - Update/Delete: ONLY the transaction creator can update or delete their own transactions
// - Balance: All balance changes are atomic within database transactions
//
// Category/Tag behavior on shared wallets:
// - Categories and tags are personal metadata owned by the transaction creator
// - When viewing shared wallet transactions, you may see categories/tags from other users
// - This is intentional - it shows how the transaction creator categorized their expense
// - Dashboard reports handle "Uncategorized" (missing category) gracefully
// - No category sharing validation - each user uses their own category hierarchy

type Repository struct {
	pool *pgxpool.Pool
}

const accessibleTransactionPredicate = `(
	transactions.user_id = $1 OR
	EXISTS (SELECT 1 FROM wallets aw WHERE aw.id = transactions.wallet_id AND aw.user_id = $1) OR
	EXISTS (SELECT 1 FROM wallets atw WHERE atw.id = transactions.to_wallet_id AND atw.user_id = $1) OR
	transactions.wallet_id IN (SELECT wallet_id FROM wallet_shares WHERE user_id = $1 AND status = 'joined') OR
	transactions.to_wallet_id IN (SELECT wallet_id FROM wallet_shares WHERE user_id = $1 AND status = 'joined')
)`

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
	transaction, err := insertTransaction(ctx, tx, userID, input)
	if err != nil {
		return Transaction{}, err
	}
	if err := syncTags(ctx, tx, userID, transaction.ID, input.TagIDs); err != nil {
		return Transaction{}, err
	}
	return transaction, nil
}

func (r *Repository) List(ctx context.Context, userID string, filter TransactionFilter, pagination page.Params) (page.Result[Transaction], error) {
	orderBy := transactionOrderBy(pagination.Sort)
	rows, err := r.pool.Query(ctx, `
		SELECT transactions.id::text, transactions.user_id::text, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor,
			ARRAY(SELECT tag_id::text FROM transaction_tags WHERE user_id = transactions.user_id AND transaction_id = transactions.id ORDER BY tag_id),
			transaction_at, note, created_at, updated_at
		FROM transactions
		WHERE `+accessibleTransactionPredicate+`
			AND ($2 = '' OR type = $2)
			AND ($3 = '' OR wallet_id = NULLIF($3, '')::uuid OR to_wallet_id = NULLIF($3, '')::uuid)
			AND ($4 = '' OR category_id IN (
				WITH RECURSIVE subtree AS (
					SELECT id FROM categories WHERE user_id = $1 AND id = NULLIF($4, '')::uuid
					UNION ALL
					SELECT c.id FROM categories c JOIN subtree s ON c.parent_id = s.id WHERE c.user_id = $1
				)
				SELECT id FROM subtree
			))
			AND ($5::timestamptz IS NULL OR transaction_at >= $5)
			AND ($6::timestamptz IS NULL OR transaction_at < $6)
			AND ($9 = '' OR transactions.id IN (SELECT transaction_id FROM transaction_tags WHERE user_id = $1 AND tag_id = NULLIF($9, '')::uuid))
		ORDER BY `+orderBy+`
		LIMIT $7 OFFSET $8
	`, userID, filter.Type, filter.WalletID, filter.CategoryID, nullableTime(filter.From), nullableTime(filter.To), pagination.Limit, pagination.Offset, filter.TagID)
	if err != nil {
		return page.Result[Transaction]{}, err
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			return page.Result[Transaction]{}, err
		}
		transactions = append(transactions, transaction)
	}
	if err := rows.Err(); err != nil {
		return page.Result[Transaction]{}, err
	}

	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM transactions
		WHERE `+accessibleTransactionPredicate+`
			AND ($2 = '' OR type = $2)
			AND ($3 = '' OR wallet_id = NULLIF($3, '')::uuid OR to_wallet_id = NULLIF($3, '')::uuid)
			AND ($4 = '' OR category_id IN (
				WITH RECURSIVE subtree AS (
					SELECT id FROM categories WHERE user_id = $1 AND id = NULLIF($4, '')::uuid
					UNION ALL
					SELECT c.id FROM categories c JOIN subtree s ON c.parent_id = s.id WHERE c.user_id = $1
				)
				SELECT id FROM subtree
			))
			AND ($5::timestamptz IS NULL OR transaction_at >= $5)
			AND ($6::timestamptz IS NULL OR transaction_at < $6)
			AND ($7 = '' OR id IN (SELECT transaction_id FROM transaction_tags WHERE user_id = $1 AND tag_id = NULLIF($7, '')::uuid))
	`, userID, filter.Type, filter.WalletID, filter.CategoryID, nullableTime(filter.From), nullableTime(filter.To), filter.TagID).Scan(&total); err != nil {
		return page.Result[Transaction]{}, err
	}
	return page.NewResult(transactions, pagination, total), nil
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

	oldTransaction, err := r.getForUpdate(ctx, tx, id)
	if err != nil {
		return Transaction{}, translateNotFound(err)
	}
	// Only the transaction creator can update
	if oldTransaction.UserID != userID {
		return Transaction{}, ErrUnauthorized
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
	if err := syncTags(ctx, tx, userID, transaction.ID, input.TagIDs); err != nil {
		return Transaction{}, err
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

	oldTransaction, err := r.getForUpdate(ctx, tx, id)
	if err != nil {
		return translateNotFound(err)
	}
	// Only the transaction creator can delete
	if oldTransaction.UserID != userID {
		return ErrUnauthorized
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
		SELECT transactions.id::text, transactions.user_id::text, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor,
			ARRAY(SELECT tag_id::text FROM transaction_tags WHERE user_id = transactions.user_id AND transaction_id = transactions.id ORDER BY tag_id),
			transaction_at, note, created_at, updated_at
		FROM transactions
		WHERE ` + accessibleTransactionPredicate + ` AND id = $2
	`
	if forUpdate {
		sql += ` FOR UPDATE`
	}
	return scanTransaction(q.QueryRow(ctx, sql, userID, id))
}

// getForUpdate fetches a transaction by ID with FOR UPDATE lock, without wallet access checks.
// Used for update/delete operations where the caller verifies user_id separately.
func (r *Repository) getForUpdate(ctx context.Context, tx pgx.Tx, id string) (Transaction, error) {
	sql := `
		SELECT transactions.id::text, transactions.user_id::text, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor,
			ARRAY(SELECT tag_id::text FROM transaction_tags WHERE user_id = transactions.user_id AND transaction_id = transactions.id ORDER BY tag_id),
			transaction_at, note, created_at, updated_at
		FROM transactions
		WHERE id = $1
		FOR UPDATE
	`
	return scanTransaction(tx.QueryRow(ctx, sql, id))
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
		if err := ensureCategory(ctx, tx, userID, input.CategoryID, string(input.Type)); err != nil {
			return err
		}
	}
	return ensureTags(ctx, tx, userID, input.TagIDs)
}

func insertTransaction(ctx context.Context, tx pgx.Tx, userID string, input TransactionInput) (Transaction, error) {
	transaction, err := scanTransaction(tx.QueryRow(ctx, `
		INSERT INTO transactions (user_id, type, wallet_id, to_wallet_id, category_id, amount_minor, transaction_at, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text, user_id::text, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, '{}'::text[], transaction_at, note, created_at, updated_at
	`, userID, input.Type, input.WalletID, nullableUUID(input.ToWalletID), nullableUUID(input.CategoryID), input.AmountMinor, input.TransactionUTC, input.Note))
	if err != nil {
		return Transaction{}, err
	}
	transaction.TagIDs = uniqueTagIDs(input.TagIDs)
	return transaction, nil
}

func updateTransaction(ctx context.Context, tx pgx.Tx, userID string, id string, input TransactionInput) (Transaction, error) {
	transaction, err := scanTransaction(tx.QueryRow(ctx, `
		UPDATE transactions
		SET type = $3, wallet_id = $4, to_wallet_id = $5, category_id = $6, amount_minor = $7,
			transaction_at = $8, note = $9, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, '{}'::text[], transaction_at, note, created_at, updated_at
	`, userID, id, input.Type, input.WalletID, nullableUUID(input.ToWalletID), nullableUUID(input.CategoryID), input.AmountMinor, input.TransactionUTC, input.Note))
	if err != nil {
		return Transaction{}, err
	}
	transaction.TagIDs = uniqueTagIDs(input.TagIDs)
	return transaction, nil
}

func applyDeltas(ctx context.Context, tx pgx.Tx, userID string, deltas []BalanceDelta) error {
	for _, delta := range deltas {
		tag, err := tx.Exec(ctx, `
			UPDATE wallets
			SET balance_minor = balance_minor + $1, updated_at = now()
			WHERE id = $2
				AND (
					user_id = $3 OR
					EXISTS (
						SELECT 1
						FROM wallet_shares
						WHERE wallet_id = wallets.id AND user_id = $3 AND status = 'joined'
					)
				)
		`, delta.AmountMinor, delta.WalletID, userID)
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
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM wallets w
			LEFT JOIN wallet_shares ws ON w.id = ws.wallet_id AND ws.user_id = $1 AND ws.status = 'joined'
			WHERE (w.user_id = $1 OR ws.wallet_id IS NOT NULL) AND w.id = $2
		)
	`, userID, walletID).Scan(&exists); err != nil {
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

func ensureTags(ctx context.Context, tx pgx.Tx, userID string, tagIDs []string) error {
	uniqueIDs := uniqueTagIDs(tagIDs)
	if len(uniqueIDs) == 0 {
		return nil
	}

	var count int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM tags
		WHERE user_id = $1 AND id = ANY($2::uuid[])
	`, userID, uniqueIDs).Scan(&count); err != nil {
		return err
	}
	if count != len(uniqueIDs) {
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
		&transaction.TagIDs,
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

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

func transactionOrderBy(sort string) string {
	switch sort {
	case "transaction_at_asc":
		return "transaction_at ASC, created_at ASC"
	case "created_at_desc":
		return "created_at DESC"
	case "created_at_asc":
		return "created_at ASC"
	case "amount_desc":
		return "amount_minor DESC, transaction_at DESC"
	case "amount_asc":
		return "amount_minor ASC, transaction_at DESC"
	default:
		return "transaction_at DESC, created_at DESC"
	}
}

func translateNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func syncTags(ctx context.Context, tx pgx.Tx, userID string, transactionID string, tagIDs []string) error {
	_, err := tx.Exec(ctx, `DELETE FROM transaction_tags WHERE user_id = $1 AND transaction_id = $2`, userID, transactionID)
	if err != nil {
		return err
	}
	for _, tagID := range uniqueTagIDs(tagIDs) {
		_, err := tx.Exec(ctx, `INSERT INTO transaction_tags (user_id, transaction_id, tag_id) VALUES ($1, $2, $3)`, userID, transactionID, tagID)
		if err != nil {
			return err
		}
	}
	return nil
}

func uniqueTagIDs(tagIDs []string) []string {
	if len(tagIDs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tagIDs))
	uniqueIDs := make([]string, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		if _, ok := seen[tagID]; ok {
			continue
		}
		seen[tagID] = struct{}{}
		uniqueIDs = append(uniqueIDs, tagID)
	}
	return uniqueIDs
}
