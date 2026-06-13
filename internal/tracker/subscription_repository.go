package tracker

import (
	"context"
	"time"

	"affluena/internal/page"
	"affluena/internal/transaction"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubscriptionRepository struct {
	pool            *pgxpool.Pool
	transactionRepo *transaction.Repository
}

func NewSubscriptionRepository(pool *pgxpool.Pool, transactionRepo *transaction.Repository) *SubscriptionRepository {
	return &SubscriptionRepository{pool: pool, transactionRepo: transactionRepo}
}

func (r *SubscriptionRepository) Create(ctx context.Context, userID string, subscription Subscription) (Subscription, error) {
	if err := ensureExpensePaymentRefs(ctx, r.pool, userID, subscription.WalletID, subscription.CategoryID); err != nil {
		return Subscription{}, err
	}
	if subscription.Status == "" {
		subscription.Status = SubscriptionStatusActive
	}

	return scanSubscription(r.pool.QueryRow(ctx, `
		INSERT INTO subscriptions (user_id, name, account_detail, wallet_id, category_id, amount_minor, billing_cycle, next_due_date, status, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id::text, user_id::text, name, account_detail, wallet_id::text, category_id::text,
			amount_minor, billing_cycle, next_due_date, status, note, created_at, updated_at
	`, userID, subscription.Name, subscription.AccountDetail, subscription.WalletID, subscription.CategoryID,
		subscription.AmountMinor, subscription.BillingCycle, subscription.NextDueDate, subscription.Status, subscription.Note))
}

func (r *SubscriptionRepository) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Subscription], error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, user_id::text, name, account_detail, wallet_id::text, category_id::text,
			amount_minor, billing_cycle, next_due_date, status, note, created_at, updated_at
		FROM subscriptions
		WHERE user_id = $1
		ORDER BY `+subscriptionOrderBy(pagination.Sort)+`
		LIMIT $2 OFFSET $3
	`, userID, pagination.Limit, pagination.Offset)
	if err != nil {
		return page.Result[Subscription]{}, err
	}
	defer rows.Close()

	var subscriptions []Subscription
	for rows.Next() {
		subscription, err := scanSubscription(rows)
		if err != nil {
			return page.Result[Subscription]{}, err
		}
		subscriptions = append(subscriptions, subscription)
	}
	if err := rows.Err(); err != nil {
		return page.Result[Subscription]{}, err
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM subscriptions WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return page.Result[Subscription]{}, err
	}
	return page.NewResult(subscriptions, pagination, total), nil
}

func subscriptionOrderBy(sort string) string {
	switch sort {
	case "next_due_date_desc":
		return "next_due_date DESC, created_at DESC"
	case "created_at_desc":
		return "created_at DESC"
	case "created_at_asc":
		return "created_at ASC"
	case "name_asc":
		return "name ASC"
	case "name_desc":
		return "name DESC"
	default:
		return "next_due_date ASC, created_at DESC"
	}
}

func (r *SubscriptionRepository) Get(ctx context.Context, userID string, id string) (Subscription, error) {
	return r.get(ctx, r.pool, userID, id, false)
}

func (r *SubscriptionRepository) Update(ctx context.Context, userID string, id string, subscription Subscription) (Subscription, error) {
	if err := ensureExpensePaymentRefs(ctx, r.pool, userID, subscription.WalletID, subscription.CategoryID); err != nil {
		return Subscription{}, err
	}

	return scanSubscription(r.pool.QueryRow(ctx, `
		UPDATE subscriptions
		SET name = $3, account_detail = $4, wallet_id = $5, category_id = $6, amount_minor = $7,
			billing_cycle = $8, next_due_date = $9, status = $10, note = $11, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, name, account_detail, wallet_id::text, category_id::text,
			amount_minor, billing_cycle, next_due_date, status, note, created_at, updated_at
	`, userID, id, subscription.Name, subscription.AccountDetail, subscription.WalletID, subscription.CategoryID,
		subscription.AmountMinor, subscription.BillingCycle, subscription.NextDueDate,
		subscription.Status, subscription.Note))
}

func (r *SubscriptionRepository) Delete(ctx context.Context, userID string, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM subscriptions WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *SubscriptionRepository) Pay(ctx context.Context, userID string, id string, paidAt time.Time, note string) (SubscriptionPayment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SubscriptionPayment{}, err
	}
	defer tx.Rollback(ctx)

	subscription, err := r.get(ctx, tx, userID, id, true)
	if err != nil {
		return SubscriptionPayment{}, err
	}
	if subscription.Status != SubscriptionStatusActive {
		return SubscriptionPayment{}, errInactiveSubscription
	}

	nextDueDate, err := AdvanceSubscriptionDueDate(subscription.NextDueDate, subscription.BillingCycle)
	if err != nil {
		return SubscriptionPayment{}, err
	}

	paymentNote := note
	if paymentNote == "" {
		paymentNote = "Subscription payment: " + subscription.Name
	}
	createdTx, err := r.transactionRepo.CreateInTx(ctx, tx, userID, transaction.TransactionInput{
		Type:           transaction.TransactionTypeExpense,
		WalletID:       subscription.WalletID,
		CategoryID:     subscription.CategoryID,
		AmountMinor:    subscription.AmountMinor,
		TransactionUTC: paidAt.UTC(),
		Note:           paymentNote,
	})
	if err != nil {
		return SubscriptionPayment{}, err
	}

	updated, err := scanSubscription(tx.QueryRow(ctx, `
		UPDATE subscriptions
		SET next_due_date = $3, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, name, account_detail, wallet_id::text, category_id::text,
			amount_minor, billing_cycle, next_due_date, status, note, created_at, updated_at
	`, userID, id, nextDueDate))
	if err != nil {
		return SubscriptionPayment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SubscriptionPayment{}, err
	}
	return SubscriptionPayment{Subscription: updated, Transaction: createdTx}, nil
}

func (r *SubscriptionRepository) get(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID string, id string, forUpdate bool) (Subscription, error) {
	sql := `
		SELECT id::text, user_id::text, name, account_detail, wallet_id::text, category_id::text,
			amount_minor, billing_cycle, next_due_date, status, note, created_at, updated_at
		FROM subscriptions
		WHERE user_id = $1 AND id = $2
	`
	if forUpdate {
		sql += ` FOR UPDATE`
	}
	return scanSubscription(q.QueryRow(ctx, sql, userID, id))
}

func scanSubscription(row rowScanner) (Subscription, error) {
	var subscription Subscription
	err := row.Scan(
		&subscription.ID,
		&subscription.UserID,
		&subscription.Name,
		&subscription.AccountDetail,
		&subscription.WalletID,
		&subscription.CategoryID,
		&subscription.AmountMinor,
		&subscription.BillingCycle,
		&subscription.NextDueDate,
		&subscription.Status,
		&subscription.Note,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
	)
	return subscription, err
}
