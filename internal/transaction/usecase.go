package transaction

import "context"

import (
	"fmt"

	"affluena-api/internal/activity"
	"affluena-api/internal/page"
)

type RepositoryPort interface {
	Create(ctx context.Context, userID string, input TransactionInput) (Transaction, error)
	List(ctx context.Context, userID string, filter TransactionFilter, pagination page.Params) (page.Result[Transaction], error)
	Get(ctx context.Context, userID string, id string) (Transaction, error)
	Update(ctx context.Context, userID string, id string, input TransactionInput) (Transaction, error)
	Delete(ctx context.Context, userID string, id string) error
}

type UseCase struct {
	repo       RepositoryPort
	activityUC activity.UseCase
}

func NewUseCase(repo RepositoryPort, activityUC activity.UseCase) *UseCase {
	return &UseCase{repo: repo, activityUC: activityUC}
}

func (u *UseCase) Create(ctx context.Context, userID string, input TransactionInput) (Transaction, error) {
	if _, err := BalanceDeltas(input); err != nil {
		return Transaction{}, err
	}
	t, err := u.repo.Create(ctx, userID, input)
	if err == nil && u.activityUC != nil {
		desc := fmt.Sprintf("Mencatat transaksi %s sebesar %.2f", input.Type, float64(input.AmountMinor))
		u.activityUC.LogActivity(ctx, userID, "CREATE", "TRANSACTION", &t.ID, desc)
	}
	return t, err
}

func (u *UseCase) List(ctx context.Context, userID string, filter TransactionFilter, pagination page.Params) (page.Result[Transaction], error) {
	return u.repo.List(ctx, userID, filter, pagination)
}

func (u *UseCase) Get(ctx context.Context, userID string, id string) (Transaction, error) {
	return u.repo.Get(ctx, userID, id)
}

func (u *UseCase) Update(ctx context.Context, userID string, id string, input TransactionInput) (Transaction, error) {
	if _, err := BalanceDeltas(input); err != nil {
		return Transaction{}, err
	}
	t, err := u.repo.Update(ctx, userID, id, input)
	if err == nil && u.activityUC != nil {
		desc := fmt.Sprintf("Mengubah transaksi %s menjadi %.2f", input.Type, float64(input.AmountMinor))
		u.activityUC.LogActivity(ctx, userID, "UPDATE", "TRANSACTION", &id, desc)
	}
	return t, err
}

func (u *UseCase) Delete(ctx context.Context, userID string, id string) error {
	err := u.repo.Delete(ctx, userID, id)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "DELETE", "TRANSACTION", &id, "Menghapus transaksi")
	}
	return err
}
