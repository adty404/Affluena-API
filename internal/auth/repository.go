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
	// seedDefaults runs inside CreateUser's transaction to give new accounts
	// their onboarding defaults. A field (not a direct call) so integration
	// tests can inject a failure and prove the user row rolls back with it.
	seedDefaults func(ctx context.Context, tx pgx.Tx, userID string) error
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, seedDefaults: seedOnboardingDefaults}
}

// CreateUser inserts the user row AND its onboarding defaults (8 categories +
// the starter wallet, see onboarding.go) in one transaction: either the
// account arrives fully set up, or nothing is written at all.
func (r *Repository) CreateUser(ctx context.Context, email string, passwordHash string) (User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	var user User
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id::text, email, name, avatar_url, password_hash, created_at, updated_at
	`, email, passwordHash).Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return User{}, err
	}

	if err := r.seedDefaults(ctx, tx, user.ID); err != nil {
		return User{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}

func (r *Repository) UserByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, email, name, avatar_url, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrInvalidCredentials
	}
	return user, err
}

func (r *Repository) UserByID(ctx context.Context, userID string) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, email, name, avatar_url, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func (r *Repository) UpdateUserProfile(ctx context.Context, userID string, name string, avatarURL string) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		UPDATE users
		SET name = $2, avatar_url = $3, updated_at = now()
		WHERE id = $1
		RETURNING id::text, email, name, avatar_url, password_hash, created_at, updated_at
	`, userID, name, avatarURL).Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func (r *Repository) ChangePassword(ctx context.Context, userID string, newPasswordHash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $2, password_changed_at = now(), updated_at = now()
		WHERE id = $1
	`, userID, newPasswordHash)
	return err
}

func (r *Repository) RevokeAllSessionsExcept(ctx context.Context, userID string, exceptTokenHash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL AND token_hash != $2
	`, userID, exceptTokenHash)
	return err
}

func (r *Repository) ListSessions(ctx context.Context, userID string) ([]Session, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, user_id::text,
			SUBSTRING(token_hash FROM 1 FOR 12) AS token_suffix,
			user_agent, ip_address, expires_at, created_at, revoked_at, last_used_at
		FROM refresh_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.UserID, &s.TokenSuffix, &s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.CreatedAt, &s.RevokedAt, &s.LastUsedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *Repository) RevokeSessionByID(ctx context.Context, userID string, sessionID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE user_id = $1 AND id = $2 AND revoked_at IS NULL
	`, userID, sessionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
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

// ConsumeRefreshToken atomically revokes and returns the user if the token is valid.
// Returns ErrInvalidRefreshToken if the token doesn't exist, is already revoked, or is expired.
func (r *Repository) ConsumeRefreshToken(ctx context.Context, tokenHash string, now time.Time) (string, error) {
	var userID string
	err := r.pool.QueryRow(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1
			AND revoked_at IS NULL
			AND expires_at > $2
		RETURNING user_id::text
	`, tokenHash, now).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvalidRefreshToken
	}
	return userID, err
}

func (r *Repository) CreatePasswordResetToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	return err
}

func (r *Repository) ConsumePasswordResetToken(ctx context.Context, tokenHash string, now time.Time) (string, error) {
	var userID string
	err := r.pool.QueryRow(ctx, `
		UPDATE password_reset_tokens
		SET used_at = now()
		WHERE token_hash = $1
			AND used_at IS NULL
			AND expires_at > $2
		RETURNING user_id::text
	`, tokenHash, now).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvalidResetToken
	}
	return userID, err
}
