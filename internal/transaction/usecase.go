package transaction

import (
	"context"
	"fmt"
	"time"

	"affluena-api/internal/activity"
	"affluena-api/internal/httpx"
	"affluena-api/internal/page"
	"affluena-api/internal/wallet"
)

type RepositoryPort interface {
	Create(ctx context.Context, userID string, input TransactionInput) (Transaction, error)
	List(ctx context.Context, userID string, filter TransactionFilter, pagination page.Params) (page.Result[Transaction], error)
	Get(ctx context.Context, userID string, id string) (Transaction, error)
	Update(ctx context.Context, userID string, id string, input TransactionInput) (Transaction, error)
	Delete(ctx context.Context, userID string, id string) error
}

type alertUseCase interface {
	CheckBudgetAndAlert(ctx context.Context, userID, categoryID string, transactionAt time.Time)
}

type UseCase struct {
	repo       RepositoryPort
	activityUC activity.UseCase
	alertUC    alertUseCase
	walletAC   wallet.AccessChecker
}

func NewUseCase(repo RepositoryPort, activityUC activity.UseCase, alertUC alertUseCase, walletAC wallet.AccessChecker) *UseCase {
	return &UseCase{repo: repo, activityUC: activityUC, alertUC: alertUC, walletAC: walletAC}
}

func (u *UseCase) Create(ctx context.Context, userID string, input TransactionInput) (Transaction, error) {
	if err := wallet.CheckMemberAccess(u.walletAC, ctx, userID, input.WalletID); err != nil {
		return Transaction{}, httpx.ErrNotFound
	}
	if input.ToWalletID != "" {
		if err := wallet.CheckMemberAccess(u.walletAC, ctx, userID, input.ToWalletID); err != nil {
			return Transaction{}, httpx.ErrNotFound
		}
	}
	if _, err := BalanceDeltas(input); err != nil {
		return Transaction{}, err
	}
	t, err := u.repo.Create(ctx, userID, input)
	if err == nil && u.activityUC != nil {
		desc := fmt.Sprintf("Mencatat transaksi %s sebesar %.2f", input.Type, float64(input.AmountMinor))
		u.activityUC.LogActivity(ctx, userID, "CREATE", "TRANSACTION", &t.ID, desc)
	}
	if err == nil && u.alertUC != nil && input.Type == TransactionTypeExpense && input.CategoryID != "" {
		// Run alert check asynchronously
		// Use context.Background() since the request context might be cancelled
		go u.alertUC.CheckBudgetAndAlert(context.Background(), userID, input.CategoryID, input.TransactionUTC)
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
