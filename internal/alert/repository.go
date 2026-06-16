package alert

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetUserEmail(ctx context.Context, userID string) (string, error)
	GetCategoryName(ctx context.Context, categoryID string) (string, error)
	HasAlertBeenSent(ctx context.Context, userID, categoryID, monthValue, alertType string) (bool, error)
	MarkAlertSent(ctx context.Context, userID, categoryID, monthValue, alertType string) error
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

// HasAlertBeenSent checks if an alert was already sent for the given parameters.
func (r *repository) HasAlertBeenSent(ctx context.Context, userID, categoryID, monthValue, alertType string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM sent_alerts
			WHERE user_id = $1 AND category_id = $2
			AND month_value = $3 AND alert_type = $4
		)
	`, userID, categoryID, monthValue, alertType).Scan(&exists)
	return exists, err
}

// MarkAlertSent records that an alert was sent.
func (r *repository) MarkAlertSent(ctx context.Context, userID, categoryID, monthValue, alertType string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sent_alerts (user_id, category_id, month_value, alert_type, sent_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, category_id, month_value, alert_type)
		DO UPDATE SET sent_at = $5
	`, userID, categoryID, monthValue, alertType, time.Now().UTC())
	return err
}
