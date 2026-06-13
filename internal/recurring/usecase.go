package recurring

import (
	"context"
	"time"
)

type RepositoryPort interface {
	Create(ctx context.Context, userID string, input RuleInput) (Rule, error)
	List(ctx context.Context, userID string) ([]Rule, error)
	Get(ctx context.Context, userID string, id string) (Rule, error)
	Update(ctx context.Context, userID string, id string, input RuleInput) (Rule, error)
	Delete(ctx context.Context, userID string, id string) error
	RunManual(ctx context.Context, userID string, id string, now time.Time) (Run, error)
	RunDue(ctx context.Context, now time.Time, limit int) ([]Run, error)
}

type UseCase struct {
	repo RepositoryPort
}

func NewUseCase(repo RepositoryPort) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) Create(ctx context.Context, userID string, input RuleInput) (Rule, error) {
	return u.repo.Create(ctx, userID, input)
}

func (u *UseCase) List(ctx context.Context, userID string) ([]Rule, error) {
	return u.repo.List(ctx, userID)
}

func (u *UseCase) Get(ctx context.Context, userID string, id string) (Rule, error) {
	return u.repo.Get(ctx, userID, id)
}

func (u *UseCase) Update(ctx context.Context, userID string, id string, input RuleInput) (Rule, error) {
	return u.repo.Update(ctx, userID, id, input)
}

func (u *UseCase) Delete(ctx context.Context, userID string, id string) error {
	return u.repo.Delete(ctx, userID, id)
}

func (u *UseCase) RunManual(ctx context.Context, userID string, id string, now time.Time) (Run, error) {
	return u.repo.RunManual(ctx, userID, id, now)
}

func (u *UseCase) RunDue(ctx context.Context, now time.Time, limit int) ([]Run, error) {
	return u.repo.RunDue(ctx, now, limit)
}
