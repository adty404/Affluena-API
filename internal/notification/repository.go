package notification

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, userID string) ([]NotificationRule, error)
	Update(ctx context.Context, userID, id string, update NotificationRuleUpdate) (NotificationRule, error)
	EnsureDefaults(ctx context.Context, userID string) error
}

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{db: db}
}

func (r *repository) List(ctx context.Context, userID string) ([]NotificationRule, error) {
	query := `
		SELECT id, user_id, rule_key, title, description, enabled, channel, tone, created_at, updated_at
		FROM notification_rules
		WHERE user_id = $1
		ORDER BY rule_key ASC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []NotificationRule
	for rows.Next() {
		var rule NotificationRule
		if err := rows.Scan(
			&rule.ID, &rule.UserID, &rule.RuleKey, &rule.Title, &rule.Description,
			&rule.Enabled, &rule.Channel, &rule.Tone, &rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if rules == nil {
		rules = []NotificationRule{}
	}

	return rules, nil
}

func (r *repository) Update(ctx context.Context, userID, id string, update NotificationRuleUpdate) (NotificationRule, error) {
	query := `
		UPDATE notification_rules
		SET 
			enabled = COALESCE($1, enabled),
			channel = COALESCE($2, channel),
			updated_at = now()
		WHERE id = $3 AND user_id = $4
		RETURNING id, user_id, rule_key, title, description, enabled, channel, tone, created_at, updated_at
	`

	var rule NotificationRule
	err := r.db.QueryRow(ctx, query, update.Enabled, update.Channel, id, userID).Scan(
		&rule.ID, &rule.UserID, &rule.RuleKey, &rule.Title, &rule.Description,
		&rule.Enabled, &rule.Channel, &rule.Tone, &rule.CreatedAt, &rule.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return NotificationRule{}, ErrNotFound
		}
		return NotificationRule{}, err
	}

	return rule, nil
}

func (r *repository) EnsureDefaults(ctx context.Context, userID string) error {
	query := `
		INSERT INTO notification_rules (user_id, rule_key, title, description, enabled, channel, tone)
		VALUES 
			($1, 'budget-alert', 'Budget threshold alert', 'Notify when category budget reaches 80% and 100%.', true, 'both', 'orange'),
			($1, 'due-reminder', 'Due date reminder', 'Debt, installment, and subscription reminders at H-3 and H-1.', true, 'both', 'blue'),
			($1, 'recurring-run', 'Recurring run result', 'Notify when recurring transaction runs or fails.', true, 'in-app', 'green'),
			($1, 'security-alert', 'Security alert', 'Notify login from a new device or location.', true, 'email', 'red'),
			($1, 'weekly-summary', 'Weekly finance summary', 'Send a weekly cashflow, budget, and goal summary.', false, 'email', 'purple')
		ON CONFLICT (user_id, rule_key) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}
