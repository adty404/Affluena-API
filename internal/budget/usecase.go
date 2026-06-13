package budget

import (
	"context"
	"errors"
	"time"

	"affluena-api/internal/activity"
	"affluena-api/internal/page"
)

type RepositoryPort interface {
	Create(ctx context.Context, userID string, input CreateBudgetInput) (Budget, error)
	List(ctx context.Context, userID string, month time.Time, pagination page.Params) (page.Result[BudgetSummary], error)
	Get(ctx context.Context, userID string, id string) (Budget, error)
	Update(ctx context.Context, userID string, id string, input UpdateBudgetInput) (Budget, error)
	Delete(ctx context.Context, userID string, id string) error
}

type UseCase struct {
	repo       RepositoryPort
	activityUC activity.UseCase
}

func NewUseCase(repo RepositoryPort, activityUC activity.UseCase) *UseCase {
	return &UseCase{repo: repo, activityUC: activityUC}
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
	b, err := u.repo.Create(ctx, userID, input)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "CREATE", "BUDGET", &b.ID, "Menetapkan budget baru")
	}
	return b, err
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
	b, err := u.repo.Update(ctx, userID, id, input)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "UPDATE", "BUDGET", &id, "Mengubah batas budget")
	}
	return b, err
}

func (u *UseCase) Delete(ctx context.Context, userID string, id string) error {
	err := u.repo.Delete(ctx, userID, id)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "DELETE", "BUDGET", &id, "Menghapus budget bulanan")
	}
	return err
}
