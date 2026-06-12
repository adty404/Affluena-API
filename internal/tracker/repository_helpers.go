package tracker

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var errInactiveSubscription = errors.New("subscription is not active")

type rowScanner interface {
	Scan(dest ...any) error
}

type refQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func ensureExpensePaymentRefs(ctx context.Context, q refQueryer, userID string, walletID string, categoryID string) error {
	if err := ensureWallet(ctx, q, userID, walletID); err != nil {
		return err
	}
	return ensureExpenseCategory(ctx, q, userID, categoryID)
}

func ensureWallet(ctx context.Context, q refQueryer, userID string, walletID string) error {
	var exists bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM wallets WHERE user_id = $1 AND id = $2)`, userID, walletID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return pgx.ErrNoRows
	}
	return nil
}

func ensureExpenseCategory(ctx context.Context, q refQueryer, userID string, categoryID string) error {
	var exists bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM categories
			WHERE user_id = $1 AND id = $2 AND type = 'expense'
		)
	`, userID, categoryID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return pgx.ErrNoRows
	}
	return nil
}
