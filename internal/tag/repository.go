package tag

import (
	"context"
	"errors"

	"affluena-api/internal/page"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, userID string, input CreateTagInput) (Tag, error) {
	var tag Tag
	err := r.pool.QueryRow(ctx, `
		INSERT INTO tags (user_id, name)
		VALUES ($1, $2)
		RETURNING id::text, user_id::text, name, created_at, updated_at
	`, userID, input.Name).Scan(
		&tag.ID, &tag.UserID, &tag.Name, &tag.CreatedAt, &tag.UpdatedAt,
	)
	return tag, err
}

func (r *Repository) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Tag], error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, user_id::text, name, created_at, updated_at
		FROM tags
		WHERE user_id = $1
		ORDER BY `+tagOrderBy(pagination.Sort)+`
		LIMIT $2 OFFSET $3
	`, userID, pagination.Limit, pagination.Offset)
	if err != nil {
		return page.Result[Tag]{}, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.UserID, &tag.Name, &tag.CreatedAt, &tag.UpdatedAt); err != nil {
			return page.Result[Tag]{}, err
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return page.Result[Tag]{}, err
	}

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tags WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return page.Result[Tag]{}, err
	}
	return page.NewResult(tags, pagination, total), nil
}

func (r *Repository) Get(ctx context.Context, userID string, id string) (Tag, error) {
	var tag Tag
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, user_id::text, name, created_at, updated_at
		FROM tags
		WHERE user_id = $1 AND id = $2
	`, userID, id).Scan(&tag.ID, &tag.UserID, &tag.Name, &tag.CreatedAt, &tag.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Tag{}, ErrNotFound
	}
	return tag, err
}

func (r *Repository) Update(ctx context.Context, userID string, id string, input UpdateTagInput) (Tag, error) {
	var tag Tag
	err := r.pool.QueryRow(ctx, `
		UPDATE tags
		SET name = $3, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, name, created_at, updated_at
	`, userID, id, input.Name).Scan(
		&tag.ID, &tag.UserID, &tag.Name, &tag.CreatedAt, &tag.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Tag{}, ErrNotFound
	}
	return tag, err
}

func (r *Repository) Delete(ctx context.Context, userID string, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM tags WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func tagOrderBy(sort string) string {
	switch sort {
	case "created_at_asc":
		return "created_at ASC"
	case "name_asc":
		return "name ASC"
	case "name_desc":
		return "name DESC"
	default:
		return "created_at DESC"
	}
}
