package activity

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("activity not found")

type Activity struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	ActionType  string    `json:"action_type"`
	EntityType  string    `json:"entity_type"`
	EntityID    *string   `json:"entity_id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Repository interface {
	Create(ctx context.Context, activity Activity) error
	List(ctx context.Context, userID string, limit, offset int, sort string) ([]Activity, int, error)
	GetByID(ctx context.Context, userID string, id string) (*Activity, error)
}

type UseCase interface {
	LogActivity(ctx context.Context, userID, actionType, entityType string, entityID *string, description string)
	ListActivities(ctx context.Context, userID string, limit, offset int, sort string) ([]Activity, int, error)
	GetActivity(ctx context.Context, userID string, id string) (*Activity, error)
}
