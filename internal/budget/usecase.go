package budget

import (
	"context"
	"errors"
	"time"

	"affluena/internal/page"
)

type RepositoryPort interface {
	Create(ctx context.Context, userID string, input CreateBudgetInput) (Budget, error)
	List(ctx context.Context, userID string, month time.Time, pagination page.Params) (page.Result[BudgetSummary], error)
	Get(ctx context.Context, userID string, id string) (Budget, error)
	Update(ctx context.Context, userID string, id string, input UpdateBudgetInput) (Budget, error)
	Delete(ctx context.Context, userID string, id string) error
}

type UseCase struct {
	repo RepositoryPort
}

func NewUseCase(repo RepositoryPort) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) Create(ctx context.Context, userID string, input CreateBudgetInput) (Budget, error) {
	if input.LimitMinor <= 0 {
		return Budget{}, errors.New("limit_minor must be positive")
	}
	month, err := ParseBudgetMonth(input.Month)
	if err != nil {
		return Budget{}, err
	}
	input.MonthDate = month
	return u.repo.Create(ctx, userID, input)
}

func (u *UseCase) List(ctx context.Context, userID string, monthValue string, pagination page.Params) (page.Result[BudgetSummary], error) {
	month, err := ParseBudgetMonth(monthValue)
	if err != nil {
		return page.Result[BudgetSummary]{}, err
	}
	return u.repo.List(ctx, userID, month, pagination)
}

func (u *UseCase) Get(ctx context.Context, userID string, id string) (Budget, error) {
	return u.repo.Get(ctx, userID, id)
}

func (u *UseCase) Update(ctx context.Context, userID string, id string, input UpdateBudgetInput) (Budget, error) {
	if input.LimitMinor <= 0 {
		return Budget{}, errors.New("limit_minor must be positive")
	}
	month, err := ParseBudgetMonth(input.Month)
	if err != nil {
		return Budget{}, err
	}
	input.MonthDate = month
	return u.repo.Update(ctx, userID, id, input)
}

func (u *UseCase) Delete(ctx context.Context, userID string, id string) error {
	return u.repo.Delete(ctx, userID, id)
}
