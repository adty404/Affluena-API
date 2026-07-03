package apilog

import (
	"context"
	"log/slog"
	"time"
)

// pruner is the minimal repository surface the retention job needs. It is
// satisfied by *repository (and any test double), keeping the job decoupled from
// the full Repository interface.
type pruner interface {
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

// RetentionScheduler periodically prunes api_logs rows older than retentionDays.
// api_logs stores full request + response payloads for every call, so without a
// retention job the table grows unbounded. Mirrors recurring.NewScheduler's
// ticker lifecycle (an immediate run on start, then every interval until the
// context is cancelled).
type RetentionScheduler struct {
	repo          pruner
	interval      time.Duration
	retentionDays int
}

// NewRetentionScheduler builds a retention job. interval and retentionDays fall
// back to safe defaults (6h / 30d) when non-positive.
func NewRetentionScheduler(repo pruner, interval time.Duration, retentionDays int) *RetentionScheduler {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	if retentionDays <= 0 {
		retentionDays = 30
	}
	return &RetentionScheduler{repo: repo, interval: interval, retentionDays: retentionDays}
}

// Start launches the retention loop in a goroutine. It runs once immediately so a
// long-running deployment does not wait a full interval before its first prune,
// then on every tick until ctx is cancelled.
func (s *RetentionScheduler) Start(ctx context.Context) {
	go func() {
		s.run(ctx)

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("api_logs retention scheduler stopped")
				return
			case <-ticker.C:
				s.run(ctx)
			}
		}
	}()
}

func (s *RetentionScheduler) run(ctx context.Context) {
	cutoff := time.Now().UTC().AddDate(0, 0, -s.retentionDays)
	deleted, err := s.repo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		slog.Error("api_logs retention prune failed", "error", err)
		return
	}
	if deleted > 0 {
		slog.Info("api_logs retention pruned rows", "deleted", deleted, "older_than", cutoff, "retention_days", s.retentionDays)
	}
}
