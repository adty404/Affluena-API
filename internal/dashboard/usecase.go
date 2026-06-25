package dashboard

import (
	"context"
	"time"
)

type RepositoryPort interface {
	Summary(ctx context.Context, userID string, month time.Time) (Summary, error)
	CashflowTrend(ctx context.Context, userID string, opts CashflowTrendOptions) ([]CashflowTrend, error)
	ExpenseDistribution(ctx context.Context, userID string, month time.Time) ([]ExpenseDistribution, error)
	Forecast(ctx context.Context, userID string, month time.Time) (Forecast, error)
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

func (u *UseCase) CashflowTrend(ctx context.Context, userID string, opts CashflowTrendOptions) ([]CashflowTrend, error) {
	return u.repo.CashflowTrend(ctx, userID, opts)
}

func (u *UseCase) ExpenseDistribution(ctx context.Context, userID string, month time.Time) ([]ExpenseDistribution, error) {
	return u.repo.ExpenseDistribution(ctx, userID, month)
}

func (u *UseCase) Forecast(ctx context.Context, userID string, month time.Time) (Forecast, error) {
	return u.repo.Forecast(ctx, userID, month)
}
