package quickentry

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Template struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	WalletID    string    `json:"wallet_id"`
	ToWalletID  string    `json:"to_wallet_id,omitempty"`
	CategoryID  string    `json:"category_id,omitempty"`
	AmountMinor int64     `json:"amount_minor"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, userID string, template Template) (Template, error) {
	if err := r.ensureRefs(ctx, userID, template); err != nil {
		return Template{}, err
	}

	return scanTemplate(r.pool.QueryRow(ctx, `
		INSERT INTO quick_entry_templates (user_id, name, type, wallet_id, to_wallet_id, category_id, amount_minor, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text, user_id::text, name, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, note, created_at, updated_at
	`, userID, template.Name, template.Type, template.WalletID, nullableUUID(template.ToWalletID), nullableUUID(template.CategoryID), template.AmountMinor, template.Note))
}

func (r *Repository) List(ctx context.Context, userID string) ([]Template, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, user_id::text, name, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, note, created_at, updated_at
		FROM quick_entry_templates
		WHERE user_id = $1
		ORDER BY name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		template, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}
	return templates, rows.Err()
}

func (r *Repository) Get(ctx context.Context, userID string, id string) (Template, error) {
	return scanTemplate(r.pool.QueryRow(ctx, `
		SELECT id::text, user_id::text, name, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, note, created_at, updated_at
		FROM quick_entry_templates
		WHERE user_id = $1 AND id = $2
	`, userID, id))
}

func (r *Repository) Update(ctx context.Context, userID string, id string, template Template) (Template, error) {
	if err := r.ensureRefs(ctx, userID, template); err != nil {
		return Template{}, err
	}

	return scanTemplate(r.pool.QueryRow(ctx, `
		UPDATE quick_entry_templates
		SET name = $3, type = $4, wallet_id = $5, to_wallet_id = $6, category_id = $7,
			amount_minor = $8, note = $9, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, name, type, wallet_id::text, COALESCE(to_wallet_id::text, ''),
			COALESCE(category_id::text, ''), amount_minor, note, created_at, updated_at
	`, userID, id, template.Name, template.Type, template.WalletID, nullableUUID(template.ToWalletID), nullableUUID(template.CategoryID), template.AmountMinor, template.Note))
}

func (r *Repository) Delete(ctx context.Context, userID string, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM quick_entry_templates WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ensureRefs(ctx context.Context, userID string, template Template) error {
	if err := r.ensureWallet(ctx, userID, template.WalletID); err != nil {
		return err
	}
	if template.ToWalletID != "" {
		if err := r.ensureWallet(ctx, userID, template.ToWalletID); err != nil {
			return err
		}
	}
	if template.CategoryID != "" {
		return r.ensureCategory(ctx, userID, template.CategoryID, template.Type)
	}
	return nil
}

func (r *Repository) ensureWallet(ctx context.Context, userID string, walletID string) error {
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM wallets WHERE user_id = $1 AND id = $2)`, userID, walletID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ensureCategory(ctx context.Context, userID string, categoryID string, transactionType string) error {
	categoryType := transactionType
	if transactionType != "income" {
		categoryType = "expense"
	}

	var exists bool
	if err := r.pool.QueryRow(ctx, `
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

func scanTemplate(row rowScanner) (Template, error) {
	var template Template
	err := row.Scan(
		&template.ID,
		&template.UserID,
		&template.Name,
		&template.Type,
		&template.WalletID,
		&template.ToWalletID,
		&template.CategoryID,
		&template.AmountMinor,
		&template.Note,
		&template.CreatedAt,
		&template.UpdatedAt,
	)
	return template, err
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
