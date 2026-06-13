package recurring

import (
	"context"
	"errors"
	"time"

	"affluena-api/internal/page"
	"affluena-api/internal/transaction"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAlreadyExecuted = errors.New("recurring transaction occurrence already executed")
	ErrRuleInactive    = errors.New("recurring transaction rule is not active")
)

type Repository struct {
	pool            *pgxpool.Pool
	transactionRepo *transaction.Repository
}

func NewRepository(pool *pgxpool.Pool, transactionRepo *transaction.Repository) *Repository {
	return &Repository{pool: pool, transactionRepo: transactionRepo}
}

func (r *Repository) Create(ctx context.Context, userID string, input RuleInput) (Rule, error) {
	if err := validateInput(input); err != nil {
		return Rule{}, err
	}
	if input.Status == "" {
		input.Status = StatusActive
	}
	if input.IntervalCount == 0 {
		input.IntervalCount = 1
	}
	if err := r.ensureRefs(ctx, r.pool, userID, input); err != nil {
		return Rule{}, err
	}

	return scanRule(r.pool.QueryRow(ctx, `
		INSERT INTO recurring_transaction_rules (
			user_id, name, type, wallet_id, to_wallet_id, category_id, amount_minor,
			frequency, interval_count, next_run_at, end_at, status, note
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id::text, user_id::text, name, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, frequency, interval_count, next_run_at,
			end_at, last_run_at, status, note, created_at, updated_at
	`, userID, input.Name, input.Type, input.WalletID, nullableUUID(input.ToWalletID), nullableUUID(input.CategoryID),
		input.AmountMinor, input.Frequency, input.IntervalCount, input.NextRunAt.UTC(), input.EndAt, input.Status, input.Note))
}

func (r *Repository) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Rule], error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, user_id::text, name, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, frequency, interval_count, next_run_at,
			end_at, last_run_at, status, note, created_at, updated_at
		FROM recurring_transaction_rules
		WHERE user_id = $1
		ORDER BY `+recurringOrderBy(pagination.Sort)+`
		LIMIT $2 OFFSET $3
	`, userID, pagination.Limit, pagination.Offset)
	if err != nil {
		return page.Result[Rule]{}, err
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return page.Result[Rule]{}, err
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return page.Result[Rule]{}, err
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM recurring_transaction_rules WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return page.Result[Rule]{}, err
	}
	return page.NewResult(rules, pagination, total), nil
}

func recurringOrderBy(sort string) string {
	switch sort {
	case "next_run_at_desc":
		return "next_run_at DESC, created_at DESC"
	case "created_at_desc":
		return "created_at DESC"
	case "created_at_asc":
		return "created_at ASC"
	case "name_asc":
		return "name ASC"
	case "name_desc":
		return "name DESC"
	default:
		return "next_run_at ASC, created_at DESC"
	}
}

func (r *Repository) Get(ctx context.Context, userID string, id string) (Rule, error) {
	return r.get(ctx, r.pool, userID, id, false)
}

func (r *Repository) Update(ctx context.Context, userID string, id string, input RuleInput) (Rule, error) {
	if err := validateInput(input); err != nil {
		return Rule{}, err
	}
	if input.Status == "" {
		input.Status = StatusActive
	}
	if input.IntervalCount == 0 {
		input.IntervalCount = 1
	}
	if err := r.ensureRefs(ctx, r.pool, userID, input); err != nil {
		return Rule{}, err
	}

	return scanRule(r.pool.QueryRow(ctx, `
		UPDATE recurring_transaction_rules
		SET name = $3, type = $4, wallet_id = $5, to_wallet_id = $6, category_id = $7,
			amount_minor = $8, frequency = $9, interval_count = $10, next_run_at = $11,
			end_at = $12, status = $13, note = $14, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, name, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, frequency, interval_count, next_run_at,
			end_at, last_run_at, status, note, created_at, updated_at
	`, userID, id, input.Name, input.Type, input.WalletID, nullableUUID(input.ToWalletID), nullableUUID(input.CategoryID),
		input.AmountMinor, input.Frequency, input.IntervalCount, input.NextRunAt.UTC(), input.EndAt, input.Status, input.Note))
}

func (r *Repository) Delete(ctx context.Context, userID string, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM recurring_transaction_rules WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) RunManual(ctx context.Context, userID string, id string, now time.Time) (Run, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Run{}, err
	}
	defer tx.Rollback(ctx)

	rule, err := r.get(ctx, tx, userID, id, true)
	if err != nil {
		return Run{}, err
	}
	run, err := r.executeLocked(ctx, tx, rule, RunTypeManual, now.UTC())
	if err != nil {
		return Run{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Run{}, err
	}
	return run, nil
}

func (r *Repository) RunDue(ctx context.Context, now time.Time, limit int) ([]Run, error) {
	if limit <= 0 {
		limit = 20
	}

	var runs []Run
	for len(runs) < limit {
		run, ok, err := r.runOneDue(ctx, now.UTC())
		if err != nil {
			return runs, err
		}
		if !ok {
			break
		}
		if run.ID != "" {
			runs = append(runs, run)
		}
	}
	return runs, nil
}

func (r *Repository) runOneDue(ctx context.Context, now time.Time) (Run, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Run{}, false, err
	}
	defer tx.Rollback(ctx)

	rule, err := scanRule(tx.QueryRow(ctx, `
		SELECT id::text, user_id::text, name, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, frequency, interval_count, next_run_at,
			end_at, last_run_at, status, note, created_at, updated_at
		FROM recurring_transaction_rules
		WHERE status = 'active'
			AND next_run_at <= $1
			AND (end_at IS NULL OR next_run_at <= end_at)
		ORDER BY next_run_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`, now))
	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, false, nil
	}
	if err != nil {
		return Run{}, false, err
	}

	run, err := r.executeLocked(ctx, tx, rule, RunTypeScheduled, now)
	if errors.Is(err, ErrAlreadyExecuted) {
		if err := tx.Commit(ctx); err != nil {
			return Run{}, false, err
		}
		return run, true, nil
	}
	if err != nil {
		return Run{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Run{}, false, err
	}
	return run, true, nil
}

func (r *Repository) get(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID string, id string, forUpdate bool) (Rule, error) {
	sql := `
		SELECT id::text, user_id::text, name, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, frequency, interval_count, next_run_at,
			end_at, last_run_at, status, note, created_at, updated_at
		FROM recurring_transaction_rules
		WHERE user_id = $1 AND id = $2
	`
	if forUpdate {
		sql += ` FOR UPDATE`
	}
	return scanRule(q.QueryRow(ctx, sql, userID, id))
}

func (r *Repository) executeLocked(ctx context.Context, tx pgx.Tx, rule Rule, runType RunType, now time.Time) (Run, error) {
	if rule.Status != StatusActive {
		return Run{}, ErrRuleInactive
	}
	scheduledFor := rule.NextRunAt.UTC()
	if rule.EndAt != nil && scheduledFor.After((*rule.EndAt).UTC()) {
		return Run{}, ErrRuleInactive
	}

	var runID string
	err := tx.QueryRow(ctx, `
		INSERT INTO recurring_transaction_runs (rule_id, user_id, scheduled_for, run_type)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (rule_id, scheduled_for) DO NOTHING
		RETURNING id::text
	`, rule.ID, rule.UserID, scheduledFor, runType).Scan(&runID)
	if errors.Is(err, pgx.ErrNoRows) {
		updatedRule, advanceErr := r.advanceLockedRule(ctx, tx, rule, now)
		if advanceErr != nil {
			return Run{}, advanceErr
		}
		return Run{
			RuleID:       rule.ID,
			UserID:       rule.UserID,
			ScheduledFor: scheduledFor,
			RunType:      runType,
			CreatedAt:    now,
			Rule:         updatedRule,
		}, ErrAlreadyExecuted
	}
	if err != nil {
		return Run{}, err
	}

	txNote := rule.Note
	if txNote == "" {
		txNote = "Recurring transaction: " + rule.Name
	}
	createdTx, err := r.transactionRepo.CreateInTx(ctx, tx, rule.UserID, transaction.TransactionInput{
		Type:           rule.Type,
		WalletID:       rule.WalletID,
		ToWalletID:     rule.ToWalletID,
		CategoryID:     rule.CategoryID,
		AmountMinor:    rule.AmountMinor,
		TransactionUTC: scheduledFor,
		Note:           txNote,
	})
	if err != nil {
		return Run{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE recurring_transaction_runs SET transaction_id = $1 WHERE id = $2`, createdTx.ID, runID); err != nil {
		return Run{}, err
	}

	updatedRule, err := r.advanceLockedRule(ctx, tx, rule, now)
	if err != nil {
		return Run{}, err
	}

	return Run{
		ID:            runID,
		RuleID:        rule.ID,
		UserID:        rule.UserID,
		ScheduledFor:  scheduledFor,
		TransactionID: createdTx.ID,
		RunType:       runType,
		CreatedAt:     now,
		Transaction:   createdTx,
		Rule:          updatedRule,
	}, nil
}

func (r *Repository) advanceLockedRule(ctx context.Context, tx pgx.Tx, rule Rule, now time.Time) (Rule, error) {
	nextRunAt, nextStatus, err := NextStateAfterRun(rule.NextRunAt, rule.Frequency, rule.IntervalCount, rule.EndAt, rule.Status)
	if err != nil {
		return Rule{}, err
	}

	return scanRule(tx.QueryRow(ctx, `
		UPDATE recurring_transaction_rules
		SET next_run_at = $3, last_run_at = $4, status = $5, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, name, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, frequency, interval_count, next_run_at,
			end_at, last_run_at, status, note, created_at, updated_at
	`, rule.UserID, rule.ID, nextRunAt, now, nextStatus))
}

func (r *Repository) ensureRefs(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID string, input RuleInput) error {
	if err := ensureWallet(ctx, q, userID, input.WalletID); err != nil {
		return err
	}
	if input.ToWalletID != "" {
		if err := ensureWallet(ctx, q, userID, input.ToWalletID); err != nil {
			return err
		}
	}
	if input.CategoryID != "" {
		return ensureCategory(ctx, q, userID, input.CategoryID, string(input.Type))
	}
	return nil
}

func validateInput(input RuleInput) error {
	if input.Name == "" {
		return errors.New("name is required")
	}
	if input.IntervalCount < 0 {
		return errors.New("interval_count must be positive")
	}
	interval := input.IntervalCount
	if interval == 0 {
		interval = 1
	}
	if !IsValidFrequency(input.Frequency) {
		return errors.New("invalid frequency")
	}
	if input.Status != "" && !IsValidStatus(input.Status) {
		return errors.New("invalid recurring status")
	}
	if input.NextRunAt.IsZero() {
		return errors.New("next_run_at is required")
	}
	if input.EndAt != nil && input.EndAt.Before(input.NextRunAt) {
		return errors.New("end_at must be after next_run_at")
	}
	_, err := transaction.BalanceDeltas(transaction.TransactionInput{
		Type:           input.Type,
		WalletID:       input.WalletID,
		ToWalletID:     input.ToWalletID,
		CategoryID:     input.CategoryID,
		AmountMinor:    input.AmountMinor,
		TransactionUTC: input.NextRunAt,
	})
	if err != nil {
		return err
	}
	_, err = AdvanceNextRunAt(input.NextRunAt, input.Frequency, interval)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRule(row rowScanner) (Rule, error) {
	var rule Rule
	err := row.Scan(
		&rule.ID,
		&rule.UserID,
		&rule.Name,
		&rule.Type,
		&rule.WalletID,
		&rule.ToWalletID,
		&rule.CategoryID,
		&rule.AmountMinor,
		&rule.Frequency,
		&rule.IntervalCount,
		&rule.NextRunAt,
		&rule.EndAt,
		&rule.LastRunAt,
		&rule.Status,
		&rule.Note,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	return rule, err
}

func ensureWallet(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID string, walletID string) error {
	var exists bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM wallets WHERE user_id = $1 AND id = $2)`, userID, walletID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return pgx.ErrNoRows
	}
	return nil
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

func nullableUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func NotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
