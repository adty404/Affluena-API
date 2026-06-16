package splitbill

import (
	"context"
	"errors"

	"affluena-api/internal/activity"
	"affluena-api/internal/debt"
	"affluena-api/internal/transaction"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Split bill validation rules:
// - Partial split: total split amounts < transaction amount (user pays the difference)
// - Full split: total split amounts == transaction amount (user pays 0, all amounts are split)
// - Invalid: total split amounts > transaction amount (reject)
// - All individual split amounts must be positive

var (
	ErrInvalidSplitAmount            = errors.New("total split amount exceeds transaction amount")
	ErrInvalidSplitParticipantAmount = errors.New("split amount must be positive")
)

type transactionRepository interface {
	CreateInTx(ctx context.Context, tx pgx.Tx, userID string, input transaction.TransactionInput) (transaction.Transaction, error)
}

type debtRepository interface {
	CreateInTx(ctx context.Context, tx pgx.Tx, userID string, input debt.DebtInput) (debt.Debt, error)
}

type UseCase struct {
	pool            *pgxpool.Pool
	transactionRepo transactionRepository
	debtRepo        debtRepository
	activityUC      activity.UseCase
}

func NewUseCase(pool *pgxpool.Pool, transactionRepo transactionRepository, debtRepo debtRepository, activityUC activity.UseCase) *UseCase {
	return &UseCase{
		pool:            pool,
		transactionRepo: transactionRepo,
		debtRepo:        debtRepo,
		activityUC:      activityUC,
	}
}

func (u *UseCase) SplitExpense(ctx context.Context, userID string, input SplitTransactionInput) (SplitTransactionResponse, error) {
	var totalSplitMinor int64 = 0
	for _, split := range input.Splits {
		if split.AmountMinor <= 0 {
			return SplitTransactionResponse{}, ErrInvalidSplitParticipantAmount
		}
		totalSplitMinor += split.AmountMinor
	}

	if totalSplitMinor > input.TotalAmountMinor {
		return SplitTransactionResponse{}, ErrInvalidSplitAmount
	}

	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return SplitTransactionResponse{}, err
	}
	defer tx.Rollback(ctx)

	// The user pays the full amount upfront, so the transaction records the total amount.
	// The portions owed by others are recorded as receivable debts.
	userTxInput := transaction.TransactionInput{
		Type:           transaction.TransactionTypeExpense,
		WalletID:       input.WalletID,
		CategoryID:     input.CategoryID,
		AmountMinor:    input.TotalAmountMinor,
		TagIDs:         input.TagIDs,
		TransactionUTC: input.TransactionAt,
		Note:           input.Note,
	}

	userTx, err := u.transactionRepo.CreateInTx(ctx, tx, userID, userTxInput)
	if err != nil {
		return SplitTransactionResponse{}, err
	}

	var debtIDs []string
	for _, split := range input.Splits {
		debtInput := debt.DebtInput{
			Type:                     debt.DebtTypeReceivable,
			CounterpartyName:         split.CounterpartyName,
			WalletID:                 input.WalletID,
			DisbursementCategoryID:   split.DisbursementCategoryID,
			PaymentCategoryID:        split.PaymentCategoryID,
			OriginationTransactionID: &userTx.ID,
			PrincipalAmountMinor:     split.AmountMinor,
			OpenedAt:                 input.TransactionAt,
			Note:                     input.Note + " (Split: " + split.CounterpartyName + ")",
		}

		createdDebt, err := u.debtRepo.CreateInTx(ctx, tx, userID, debtInput)
		if err != nil {
			return SplitTransactionResponse{}, err
		}
		debtIDs = append(debtIDs, createdDebt.ID)
	}
	if err := tx.Commit(ctx); err != nil {
		return SplitTransactionResponse{}, err
	}

	if u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "CREATE", "SPLIT_BILL", &userTx.ID, "Mencatat transaksi patungan (Split Bill) senilai total "+input.Note)
	}

	return SplitTransactionResponse{
		TransactionID: userTx.ID,
		DebtIDs:       debtIDs,
	}, nil
}
