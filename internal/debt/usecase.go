package debt

import (
	"context"
	"errors"
	"time"

	"affluena-api/internal/activity"
	"affluena-api/internal/page"
)

type RepositoryPort interface {
	Create(ctx context.Context, userID string, input DebtInput) (Debt, error)
	List(ctx context.Context, userID string, pagination page.Params) (page.Result[Debt], error)
	Get(ctx context.Context, userID string, id string) (Debt, error)
	Update(ctx context.Context, userID string, id string, update DebtUpdate) (Debt, error)
	Delete(ctx context.Context, userID string, id string) error
	Pay(ctx context.Context, userID string, id string, amountMinor int64, paidAt time.Time, note string) (DebtPayment, error)
}

type UseCase struct {
	repo       RepositoryPort
	activityUC activity.UseCase
}

func NewUseCase(repo RepositoryPort, activityUC activity.UseCase) *UseCase {
	return &UseCase{repo: repo, activityUC: activityUC}
}

func (u *UseCase) Create(ctx context.Context, userID string, input DebtInput) (Debt, error) {
	if !IsValidDebtType(input.Type) {
		return Debt{}, errInvalidDebtType
	}
	if input.PrincipalAmountMinor <= 0 {
		return Debt{}, errors.New("principal_amount_minor must be positive")
	}
	d, err := u.repo.Create(ctx, userID, input)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "CREATE", "DEBT", &d.ID, "Mencatat hutang/piutang baru: "+input.CounterpartyName)
	}
	return d, err
}

func (u *UseCase) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Debt], error) {
	return u.repo.List(ctx, userID, pagination)
}

func (u *UseCase) Get(ctx context.Context, userID string, id string) (Debt, error) {
	return u.repo.Get(ctx, userID, id)
}

func (u *UseCase) Update(ctx context.Context, userID string, id string, update DebtUpdate) (Debt, error) {
	if update.Status != "" && !IsValidDebtStatus(update.Status) {
		return Debt{}, errInvalidDebtStatus
	}
	d, err := u.repo.Update(ctx, userID, id, update)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "UPDATE", "DEBT", &id, "Mengubah rincian hutang/piutang")
	}
	return d, err
}

func (u *UseCase) Delete(ctx context.Context, userID string, id string) error {
	err := u.repo.Delete(ctx, userID, id)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "DELETE", "DEBT", &id, "Menghapus data hutang/piutang")
	}
	return err
}

func (u *UseCase) Pay(ctx context.Context, userID string, id string, amountMinor int64, paidAt time.Time, note string) (DebtPayment, error) {
	if amountMinor <= 0 {
		return DebtPayment{}, errInvalidPayment
	}
	pay, err := u.repo.Pay(ctx, userID, id, amountMinor, paidAt, note)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "UPDATE", "DEBT_PAYMENT", &id, "Membayar/menerima pelunasan hutang")
	}
	return pay, err
}
