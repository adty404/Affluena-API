package notification

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, userID string) ([]NotificationRule, error)
	Update(ctx context.Context, userID, id string, update NotificationRuleUpdate) (NotificationRule, error)
	EnsureDefaults(ctx context.Context, userID string) error
}

// DeliveryRepository backs the notification scheduler + gating: it looks up a
// user's rule preference, records deliveries (with de-dupe), lists candidate
// users, and pulls the due-window / cashflow data the notifications need.
type DeliveryRepository interface {
	// GetRule returns (enabled, channel) for the user's rule_key. found=false
	// when the user has no such rule row (the caller must fail closed).
	GetRule(ctx context.Context, userID, ruleKey string) (enabled bool, channel string, found bool, err error)
	// RecordDelivery inserts a delivery row; inserted=false means a row with the
	// same (user_id, rule_key, dedupe_key) already existed (de-dupe hit).
	RecordDelivery(ctx context.Context, d Delivery) (inserted bool, err error)
	// UserEmail returns the user's email for the email channel.
	UserEmail(ctx context.Context, userID string) (string, error)
	// ListUserIDsWithRuleEnabled returns the ids of users whose rule_key row is
	// enabled — the candidate set the scheduler iterates.
	ListUserIDsWithRuleEnabled(ctx context.Context, ruleKey string) ([]string, error)
	// DueItemsForUser returns subscriptions/installments/debts due exactly
	// daysAhead days from asOf (used for H-3 and H-1 windows).
	DueItemsForUser(ctx context.Context, userID string, asOf time.Time, daysAhead int) ([]DueItem, error)
	// WeeklyCashflowForUser sums income/expense over [from, to) for the user.
	WeeklyCashflowForUser(ctx context.Context, userID string, from, to time.Time) (CashflowSummary, error)
}

// Delivery is a row written to notification_deliveries.
type Delivery struct {
	UserID     string
	RuleKey    string
	DedupeKey  string
	Channel    string
	Title      string
	Message    string
	Severity   string
	ActionPath string
}

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{db: db}
}

// NewDeliveryRepository exposes the same underlying *repository as the
// scheduler/gating surface. Kept separate from Repository so the HTTP handler
// dependency stays the small preferences interface.
func NewDeliveryRepository(db *pgxpool.Pool) DeliveryRepository {
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

func (r *repository) GetRule(ctx context.Context, userID, ruleKey string) (bool, string, bool, error) {
	var enabled bool
	var channel string
	err := r.db.QueryRow(ctx,
		`SELECT enabled, channel FROM notification_rules WHERE user_id = $1 AND rule_key = $2`,
		userID, ruleKey,
	).Scan(&enabled, &channel)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", false, nil
	}
	if err != nil {
		return false, "", false, err
	}
	return enabled, channel, true, nil
}

func (r *repository) RecordDelivery(ctx context.Context, d Delivery) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO notification_deliveries
			(user_id, rule_key, dedupe_key, channel, title, message, severity, action_path)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, rule_key, dedupe_key) DO NOTHING
	`, d.UserID, d.RuleKey, d.DedupeKey, d.Channel, d.Title, d.Message, d.Severity, d.ActionPath)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *repository) UserEmail(ctx context.Context, userID string) (string, error) {
	var email string
	err := r.db.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email)
	return email, err
}

func (r *repository) ListUserIDsWithRuleEnabled(ctx context.Context, ruleKey string) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT user_id::text FROM notification_rules WHERE rule_key = $1 AND enabled = true`,
		ruleKey,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// DueItemsForUser finds subscriptions, installments, and debts whose due date is
// exactly `daysAhead` calendar days from asOf (the H-3 / H-1 windows). Dates are
// compared at day granularity so a tick at any time of day matches. The queries
// mirror the dashboard "upcoming" selects but pin the window to a single day.
func (r *repository) DueItemsForUser(ctx context.Context, userID string, asOf time.Time, daysAhead int) ([]DueItem, error) {
	target := time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, daysAhead)

	var items []DueItem

	// Subscriptions: next_due_date is a DATE.
	subRows, err := r.db.Query(ctx, `
		SELECT id::text, name, amount_minor, next_due_date
		FROM subscriptions
		WHERE user_id = $1 AND status = 'active' AND next_due_date::date = $2::date
	`, userID, target)
	if err != nil {
		return nil, err
	}
	for subRows.Next() {
		var it DueItem
		if err := subRows.Scan(&it.EntityID, &it.Name, &it.AmountMinor, &it.DueDate); err != nil {
			subRows.Close()
			return nil, err
		}
		it.EntityType = "subscription"
		it.DaysUntilDue = daysAhead
		items = append(items, it)
	}
	subRows.Close()
	if err := subRows.Err(); err != nil {
		return nil, err
	}

	// Installments: due date is computed from due_day within the target month
	// (clamped to the month length), matching the dashboard's installment_due CTE.
	instRows, err := r.db.Query(ctx, `
		WITH installment_due AS (
			SELECT id::text AS id, name, monthly_amount_minor,
				make_date(
					EXTRACT(year FROM $2::date)::int,
					EXTRACT(month FROM $2::date)::int,
					LEAST(due_day, EXTRACT(day FROM (date_trunc('month', $2::date) + interval '1 month' - interval '1 day'))::int)
				) AS due_date
			FROM installments
			WHERE user_id = $1 AND status = 'active'
		)
		SELECT id, name, monthly_amount_minor, due_date
		FROM installment_due
		WHERE due_date = $2::date
	`, userID, target)
	if err != nil {
		return nil, err
	}
	for instRows.Next() {
		var it DueItem
		if err := instRows.Scan(&it.EntityID, &it.Name, &it.AmountMinor, &it.DueDate); err != nil {
			instRows.Close()
			return nil, err
		}
		it.EntityType = "installment"
		it.DaysUntilDue = daysAhead
		items = append(items, it)
	}
	instRows.Close()
	if err := instRows.Err(); err != nil {
		return nil, err
	}

	// Debts: payable debts with a due_date, still open/partial.
	debtRows, err := r.db.Query(ctx, `
		SELECT id::text, counterparty_name, (principal_amount_minor - paid_amount_minor), due_date
		FROM debts
		WHERE user_id = $1 AND type = 'payable' AND status IN ('open', 'partial')
			AND due_date IS NOT NULL AND due_date::date = $2::date
	`, userID, target)
	if err != nil {
		return nil, err
	}
	for debtRows.Next() {
		var it DueItem
		if err := debtRows.Scan(&it.EntityID, &it.Name, &it.AmountMinor, &it.DueDate); err != nil {
			debtRows.Close()
			return nil, err
		}
		it.EntityType = "debt"
		it.DaysUntilDue = daysAhead
		items = append(items, it)
	}
	debtRows.Close()
	if err := debtRows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *repository) WeeklyCashflowForUser(ctx context.Context, userID string, from, to time.Time) (CashflowSummary, error) {
	var s CashflowSummary
	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount_minor ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount_minor ELSE 0 END), 0)
		FROM transactions
		WHERE user_id = $1
			AND type IN ('income', 'expense')
			AND transaction_at >= $2
			AND transaction_at < $3
	`, userID, from, to).Scan(&s.IncomeMinor, &s.ExpenseMinor)
	if err != nil {
		return CashflowSummary{}, err
	}
	s.NetMinor = s.IncomeMinor - s.ExpenseMinor
	return s, nil
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
