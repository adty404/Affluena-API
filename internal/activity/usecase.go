package activity

import (
	"context"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cespare/xxhash/v2"
)

const (
	maxDescriptionLength = 500
	dedupeWindow         = 5 * time.Minute
)

type useCase struct {
	repo   Repository
	dedupe map[uint64]time.Time
}

func NewUseCase(repo Repository) UseCase {
	return &useCase{
		repo:   repo,
		dedupe: make(map[uint64]time.Time),
	}
}

// LogActivity executes in a goroutine to avoid blocking the main request.
// Description is truncated to maxDescriptionLength.
// Duplicate activities within dedupeWindow are ignored.
func (u *useCase) LogActivity(ctx context.Context, userID, actionType, entityType string, entityID *string, description string) {
	var entityIDCopy *string
	if entityID != nil {
		copied := *entityID
		entityIDCopy = &copied
	}

	// Truncate description to prevent bloat
	description = truncateDescription(description)

	// Generate dedupe key
	key := dedupeKey(userID, actionType, entityType, entityIDCopy, description)

	// Check dedupe window (safe for concurrent use)
	u.cleanupDedupe()
	if lastSeen, exists := u.dedupe[key]; exists && time.Since(lastSeen) < dedupeWindow {
		return
	}
	u.dedupe[key] = time.Now()

	// Create a new context with timeout for the background job
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		act := Activity{
			UserID:      userID,
			ActionType:  actionType,
			EntityType:  entityType,
			EntityID:    entityIDCopy,
			Description: description,
		}

		err := u.repo.Create(bgCtx, act)
		if err != nil {
			slog.Error("failed to log user activity", "user_id", userID, "action", actionType, "error", err)
		}
	}()
}

// truncateDescription limits description length to maxDescriptionLength.
func truncateDescription(desc string) string {
	if utf8.RuneCountInString(desc) <= maxDescriptionLength {
		return desc
	}
	runes := []rune(desc)
	if len(runes) > maxDescriptionLength {
		return string(runes[:maxDescriptionLength]) + "..."
	}
	return desc
}

// dedupeKey generates a hash key for deduplication.
func dedupeKey(userID, actionType, entityType string, entityID *string, description string) uint64 {
	var sb strings.Builder
	sb.WriteString(userID)
	sb.WriteString(":")
	sb.WriteString(actionType)
	sb.WriteString(":")
	sb.WriteString(entityType)
	if entityID != nil {
		sb.WriteString(":")
		sb.WriteString(*entityID)
	}
	return xxhash.Sum64String(sb.String())
}

// cleanupDedupe removes expired entries from dedupe map.
func (u *useCase) cleanupDedupe() {
	now := time.Now()
	for key, timestamp := range u.dedupe {
		if now.Sub(timestamp) > dedupeWindow {
			delete(u.dedupe, key)
		}
	}
}

func (u *useCase) ListActivities(ctx context.Context, userID string, limit, offset int, sort string) ([]Activity, int, error) {
	return u.repo.List(ctx, userID, limit, offset, sort)
}
