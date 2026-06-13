package splitbill

import (
	"context"
	"errors"

	"affluena-api/internal/activity"
	"affluena-api/internal/debt"
	"affluena-api/internal/transaction"
)

var (
	ErrInvalidSplitAmount = errors.New("total split amount exceeds transaction amount")
)

type transactionUseCase interface {
	Create(ctx context.Context, userID string, input transaction.TransactionInput) (transaction.Transaction, error)
}

type debtUseCase interface {
	Create(ctx context.Context, userID string, input debt.DebtInput) (debt.Debt, error)
}

type UseCase struct {
	transactionUC transactionUseCase
	debtUC        debtUseCase
	activityUC    activity.UseCase
}

func NewUseCase(transactionUC transactionUseCase, debtUC debtUseCase, activityUC activity.UseCase) *UseCase {
	return &UseCase{
		transactionUC: transactionUC,
		debtUC:        debtUC,
		activityUC:    activityUC,
	}
}

func (u *UseCase) SplitExpense(ctx context.Context, userID string, input SplitTransactionInput) (SplitTransactionResponse, error) {
	// 1. Calculate and validate amounts
	var totalSplitMinor int64 = 0
	for _, split := range input.Splits {
		if split.AmountMinor <= 0 {
			return SplitTransactionResponse{}, errors.New("split amount must be positive")
		}
		totalSplitMinor += split.AmountMinor
	}

	if totalSplitMinor >= input.TotalAmountMinor {
		return SplitTransactionResponse{}, ErrInvalidSplitAmount
	}

	userExpenseMinor := input.TotalAmountMinor - totalSplitMinor

	// 2. Create the main expense transaction for the user
	userTxInput := transaction.TransactionInput{
		Type:           transaction.TransactionTypeExpense,
		WalletID:       input.WalletID,
		CategoryID:     input.CategoryID,
		AmountMinor:    userExpenseMinor,
		TagIDs:         input.TagIDs,
		TransactionUTC: input.TransactionAt,
		Note:           input.Note,
	}

	userTx, err := u.transactionUC.Create(ctx, userID, userTxInput)
	if err != nil {
		return SplitTransactionResponse{}, err
	}

	// 3. Loop and create debts for counterparties
	var debtIDs []string
	for _, split := range input.Splits {
		debtInput := debt.DebtInput{
			Type:                   debt.DebtTypeReceivable,
			CounterpartyName:       split.CounterpartyName,
			WalletID:               input.WalletID,
			DisbursementCategoryID: split.DisbursementCategoryID,
			PaymentCategoryID:      split.PaymentCategoryID,
			PrincipalAmountMinor:   split.AmountMinor,
			OpenedAt:               input.TransactionAt,
			Note:                   input.Note + " (Split: " + split.CounterpartyName + ")",
		}

		createdDebt, err := u.debtUC.Create(ctx, userID, debtInput)
		if err != nil {
			// Note: Macro endpoint. We don't rollback the previously created transactions here to keep it simple.
			// The user can manually delete if something fails.
			return SplitTransactionResponse{TransactionID: userTx.ID, DebtIDs: debtIDs}, err
		}
		debtIDs = append(debtIDs, createdDebt.ID)
	}

	if u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "CREATE", "SPLIT_BILL", &userTx.ID, "Mencatat transaksi patungan (Split Bill) senilai total "+input.Note)
	}

	return SplitTransactionResponse{
		TransactionID: userTx.ID,
		DebtIDs:       debtIDs,
	}, nil
}
