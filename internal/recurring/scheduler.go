package recurring

import (
	"context"
	"log/slog"
	"time"
)

type Scheduler struct {
	usecase   recurringSchedulerUseCase
	interval  time.Duration
	batchSize int
}

type recurringSchedulerUseCase interface {
	RunDue(ctx context.Context, now time.Time, limit int) ([]Run, error)
}

func NewScheduler(usecase recurringSchedulerUseCase, interval time.Duration, batchSize int) *Scheduler {
	if interval <= 0 {
		interval = time.Minute
	}
	if batchSize <= 0 {
		batchSize = 20
	}
	return &Scheduler{usecase: usecase, interval: interval, batchSize: batchSize}
}

func (s *Scheduler) Start(ctx context.Context) {
	go func() {
		s.run(ctx)

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("recurring scheduler stopped")
				return
			case <-ticker.C:
				s.run(ctx)
			}
		}
	}()
}

func (s *Scheduler) run(ctx context.Context) {
	runs, err := s.usecase.RunDue(ctx, time.Now().UTC(), s.batchSize)
	if err != nil {
		slog.Error("recurring scheduler run failed", "error", err)
		return
	}
	if len(runs) > 0 {
		slog.Info("recurring scheduler executed rules", "count", len(runs))
	}
}
