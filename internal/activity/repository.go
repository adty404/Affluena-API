package activity

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) Create(ctx context.Context, activity Activity) error {
	query := `
		INSERT INTO user_activities (user_id, action_type, entity_type, entity_id, description)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, query,
		activity.UserID,
		activity.ActionType,
		activity.EntityType,
		activity.EntityID,
		activity.Description,
	)
	if err != nil {
		slog.Error("failed to create user activity", "error", err)
	}
	return err
}

func (r *repository) List(ctx context.Context, userID string, limit, offset int, sort string) ([]Activity, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM user_activities WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, user_id, action_type, entity_type, entity_id, description, created_at
		FROM user_activities
		WHERE user_id = $1
		ORDER BY ` + activityOrderBy(sort) + `
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var activities []Activity
	for rows.Next() {
		var a Activity
		err := rows.Scan(
			&a.ID,
			&a.UserID,
			&a.ActionType,
			&a.EntityType,
			&a.EntityID,
			&a.Description,
			&a.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		activities = append(activities, a)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	if activities == nil {
		activities = []Activity{}
	}

	return activities, total, nil
}

func activityOrderBy(sort string) string {
	switch sort {
	case "created_at_asc":
		return "created_at ASC"
	default:
		return "created_at DESC"
	}
}

func (r *repository) GetByID(ctx context.Context, userID string, id string) (*Activity, error) {
	query := `
		SELECT id, user_id, action_type, entity_type, entity_id, description, created_at
		FROM user_activities
		WHERE id = $1 AND user_id = $2
	`
	var a Activity
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&a.ID,
		&a.UserID,
		&a.ActionType,
		&a.EntityType,
		&a.EntityID,
		&a.Description,
		&a.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}
