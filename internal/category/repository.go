package category

import (
	"context"
	"errors"

	"affluena/internal/page"

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

func (r *Repository) List(ctx context.Context, userID string, categoryType string, pagination page.Params) (page.Result[Category], error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, user_id::text, name, type, created_at, updated_at
		FROM categories
		WHERE user_id = $1 AND ($2 = '' OR type = $2)
		ORDER BY `+categoryOrderBy(pagination.Sort)+`
		LIMIT $3 OFFSET $4
	`, userID, categoryType, pagination.Limit, pagination.Offset)
	if err != nil {
		return page.Result[Category]{}, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var category Category
		if err := rows.Scan(&category.ID, &category.UserID, &category.Name, &category.Type, &category.CreatedAt, &category.UpdatedAt); err != nil {
			return page.Result[Category]{}, err
		}
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return page.Result[Category]{}, err
	}
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM categories
		WHERE user_id = $1 AND ($2 = '' OR type = $2)
	`, userID, categoryType).Scan(&total); err != nil {
		return page.Result[Category]{}, err
	}
	return page.NewResult(categories, pagination, total), nil
}

func categoryOrderBy(sort string) string {
	switch sort {
	case "type_name_desc":
		return "type DESC, name DESC"
	case "name_asc":
		return "name ASC"
	case "name_desc":
		return "name DESC"
	case "created_at_desc":
		return "created_at DESC"
	case "created_at_asc":
		return "created_at ASC"
	default:
		return "type ASC, name ASC"
	}
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
