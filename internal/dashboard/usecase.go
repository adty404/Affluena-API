package dashboard

import (
	"context"
	"time"
)

type RepositoryPort interface {
	Summary(ctx context.Context, userID string, month time.Time) (Summary, error)
}

type UseCase struct {
	repo RepositoryPort
}

func NewUseCase(repo RepositoryPort) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) Summary(ctx context.Context, userID string, month time.Time) (Summary, error) {
	return u.repo.Summary(ctx, userID, month)
}
