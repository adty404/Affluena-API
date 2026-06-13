package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateUser(ctx context.Context, email string, passwordHash string) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id::text, email, password_hash, created_at, updated_at
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func (r *Repository) UserByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrInvalidCredentials
	}
	return user, err
}

func (r *Repository) UserByID(ctx context.Context, userID string) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func (r *Repository) StoreRefreshToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	return err
}

func (r *Repository) RefreshTokenUser(ctx context.Context, tokenHash string, now time.Time) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		SELECT u.id::text, u.email, u.password_hash, u.created_at, u.updated_at
		FROM refresh_tokens rt
		JOIN users u ON u.id = rt.user_id
		WHERE rt.token_hash = $1
			AND rt.expires_at > $2
			AND rt.revoked_at IS NULL
	`, tokenHash, now).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrInvalidRefreshToken
	}
	return user, err
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	return err
}
