package alert

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetUserEmail(ctx context.Context, userID string) (string, error)
	GetCategoryName(ctx context.Context, categoryID string) (string, error)
	TryInsertSentAlert(ctx context.Context, userID, categoryID, monthValue, alertType string) (bool, error)
	DeleteSentAlert(ctx context.Context, userID, categoryID, monthValue, alertType string) error
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
// Deprecated: Use TryInsertSentAlert for atomic check-and-insert.
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

// TryInsertSentAlert atomically attempts to insert a sent alert record.
// Returns true if the insert succeeded (alert should be sent), false if it already exists.
func (r *repository) TryInsertSentAlert(ctx context.Context, userID, categoryID, monthValue, alertType string) (bool, error) {
	result, err := r.pool.Exec(ctx, `
		INSERT INTO sent_alerts (user_id, category_id, month_value, alert_type, sent_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, category_id, month_value, alert_type)
		DO NOTHING
	`, userID, categoryID, monthValue, alertType, time.Now().UTC())
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

// DeleteSentAlert removes a sent alert record, allowing it to be retried later.
func (r *repository) DeleteSentAlert(ctx context.Context, userID, categoryID, monthValue, alertType string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM sent_alerts
		WHERE user_id = $1 AND category_id = $2 AND month_value = $3 AND alert_type = $4
	`, userID, categoryID, monthValue, alertType)
	return err
}

// MarkAlertSent records that an alert was sent.
// Deprecated: Use TryInsertSentAlert for atomic operation before sending email.
func (r *repository) MarkAlertSent(ctx context.Context, userID, categoryID, monthValue, alertType string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sent_alerts (user_id, category_id, month_value, alert_type, sent_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, category_id, month_value, alert_type)
		DO UPDATE SET sent_at = $5
	`, userID, categoryID, monthValue, alertType, time.Now().UTC())
	return err
}
