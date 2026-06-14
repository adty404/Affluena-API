package activity

import (
	"context"
	"log/slog"
	"time"
)

type useCase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &useCase{repo: repo}
}

// LogActivity executes in a goroutine to avoid blocking the main request
func (u *useCase) LogActivity(ctx context.Context, userID, actionType, entityType string, entityID *string, description string) {
	// Create a new context with timeout for the background job
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		act := Activity{
			UserID:      userID,
			ActionType:  actionType,
			EntityType:  entityType,
			EntityID:    entityID,
			Description: description,
		}

		err := u.repo.Create(bgCtx, act)
		if err != nil {
			slog.Error("failed to log user activity", "user_id", userID, "action", actionType, "error", err)
		}
	}()
}

func (u *useCase) ListActivities(ctx context.Context, userID string, limit, offset int, sort string) ([]Activity, int, error) {
	return u.repo.List(ctx, userID, limit, offset, sort)
}
