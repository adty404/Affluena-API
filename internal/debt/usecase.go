package debt

import (
	"context"
	"errors"
	"time"
)

type RepositoryPort interface {
	Create(ctx context.Context, userID string, input DebtInput) (Debt, error)
	List(ctx context.Context, userID string) ([]Debt, error)
	Get(ctx context.Context, userID string, id string) (Debt, error)
	Update(ctx context.Context, userID string, id string, update DebtUpdate) (Debt, error)
	Delete(ctx context.Context, userID string, id string) error
	Pay(ctx context.Context, userID string, id string, amountMinor int64, paidAt time.Time, note string) (DebtPayment, error)
}

type UseCase struct {
	repo RepositoryPort
}

func NewUseCase(repo RepositoryPort) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) Create(ctx context.Context, userID string, input DebtInput) (Debt, error) {
	if !IsValidDebtType(input.Type) {
		return Debt{}, errInvalidDebtType
	}
	if input.PrincipalAmountMinor <= 0 {
		return Debt{}, errors.New("principal_amount_minor must be positive")
	}
	return u.repo.Create(ctx, userID, input)
}

func (u *UseCase) List(ctx context.Context, userID string) ([]Debt, error) {
	return u.repo.List(ctx, userID)
}

func (u *UseCase) Get(ctx context.Context, userID string, id string) (Debt, error) {
	return u.repo.Get(ctx, userID, id)
}

func (u *UseCase) Update(ctx context.Context, userID string, id string, update DebtUpdate) (Debt, error) {
	if update.Status != "" && !IsValidDebtStatus(update.Status) {
		return Debt{}, errInvalidDebtStatus
	}
	return u.repo.Update(ctx, userID, id, update)
}

func (u *UseCase) Delete(ctx context.Context, userID string, id string) error {
	return u.repo.Delete(ctx, userID, id)
}

func (u *UseCase) Pay(ctx context.Context, userID string, id string, amountMinor int64, paidAt time.Time, note string) (DebtPayment, error) {
	if amountMinor <= 0 {
		return DebtPayment{}, errInvalidPayment
	}
	return u.repo.Pay(ctx, userID, id, amountMinor, paidAt, note)
}
