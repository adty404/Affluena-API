package wallet

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

func (r *Repository) Create(ctx context.Context, userID string, input CreateWalletInput) (Wallet, error) {
	var wallet Wallet
	err := r.pool.QueryRow(ctx, `
		INSERT INTO wallets (user_id, name, type, currency_code, balance_minor)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, user_id::text, name, type, currency_code, balance_minor, created_at, updated_at
	`, userID, input.Name, input.Type, input.CurrencyCode, input.BalanceMinor).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Name, &wallet.Type, &wallet.CurrencyCode, &wallet.BalanceMinor, &wallet.CreatedAt, &wallet.UpdatedAt,
	)
	return wallet, err
}

func (r *Repository) List(ctx context.Context, userID string) ([]Wallet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, user_id::text, name, type, currency_code, balance_minor, created_at, updated_at
		FROM wallets
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []Wallet
	for rows.Next() {
		var wallet Wallet
		if err := rows.Scan(&wallet.ID, &wallet.UserID, &wallet.Name, &wallet.Type, &wallet.CurrencyCode, &wallet.BalanceMinor, &wallet.CreatedAt, &wallet.UpdatedAt); err != nil {
			return nil, err
		}
		wallets = append(wallets, wallet)
	}
	return wallets, rows.Err()
}

func (r *Repository) Get(ctx context.Context, userID string, id string) (Wallet, error) {
	var wallet Wallet
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, user_id::text, name, type, currency_code, balance_minor, created_at, updated_at
		FROM wallets
		WHERE user_id = $1 AND id = $2
	`, userID, id).Scan(&wallet.ID, &wallet.UserID, &wallet.Name, &wallet.Type, &wallet.CurrencyCode, &wallet.BalanceMinor, &wallet.CreatedAt, &wallet.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, ErrNotFound
	}
	return wallet, err
}

func (r *Repository) Update(ctx context.Context, userID string, id string, input UpdateWalletInput) (Wallet, error) {
	var wallet Wallet
	err := r.pool.QueryRow(ctx, `
		UPDATE wallets
		SET name = $3, type = $4, currency_code = $5, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, name, type, currency_code, balance_minor, created_at, updated_at
	`, userID, id, input.Name, input.Type, input.CurrencyCode).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Name, &wallet.Type, &wallet.CurrencyCode, &wallet.BalanceMinor, &wallet.CreatedAt, &wallet.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, ErrNotFound
	}
	return wallet, err
}

func (r *Repository) Delete(ctx context.Context, userID string, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM wallets WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
