package category

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

func (r *Repository) Create(ctx context.Context, userID string, input CreateCategoryInput) (Category, error) {
	var category Category
	err := r.pool.QueryRow(ctx, `
		INSERT INTO categories (user_id, name, type)
		VALUES ($1, $2, $3)
		RETURNING id::text, user_id::text, name, type, created_at, updated_at
	`, userID, input.Name, input.Type).Scan(&category.ID, &category.UserID, &category.Name, &category.Type, &category.CreatedAt, &category.UpdatedAt)
	return category, err
}

func (r *Repository) List(ctx context.Context, userID string, categoryType string) ([]Category, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, user_id::text, name, type, created_at, updated_at
		FROM categories
		WHERE user_id = $1 AND ($2 = '' OR type = $2)
		ORDER BY type, name
	`, userID, categoryType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var category Category
		if err := rows.Scan(&category.ID, &category.UserID, &category.Name, &category.Type, &category.CreatedAt, &category.UpdatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func (r *Repository) Get(ctx context.Context, userID string, id string) (Category, error) {
	var category Category
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, user_id::text, name, type, created_at, updated_at
		FROM categories
		WHERE user_id = $1 AND id = $2
	`, userID, id).Scan(&category.ID, &category.UserID, &category.Name, &category.Type, &category.CreatedAt, &category.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	return category, err
}

func (r *Repository) Update(ctx context.Context, userID string, id string, input UpdateCategoryInput) (Category, error) {
	var category Category
	err := r.pool.QueryRow(ctx, `
		UPDATE categories
		SET name = $3, type = $4, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, name, type, created_at, updated_at
	`, userID, id, input.Name, input.Type).Scan(&category.ID, &category.UserID, &category.Name, &category.Type, &category.CreatedAt, &category.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	return category, err
}

func (r *Repository) Delete(ctx context.Context, userID string, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM categories WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
