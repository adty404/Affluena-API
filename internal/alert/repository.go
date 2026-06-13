package alert

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetUserEmail(ctx context.Context, userID string) (string, error)
	GetCategoryName(ctx context.Context, categoryID string) (string, error)
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) GetUserEmail(ctx context.Context, userID string) (string, error) {
	var email string
	err := r.pool.QueryRow(ctx, "SELECT email FROM users WHERE id = $1", userID).Scan(&email)
	return email, err
}

func (r *repository) GetCategoryName(ctx context.Context, categoryID string) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx, "SELECT name FROM categories WHERE id = $1", categoryID).Scan(&name)
	return name, err
}
