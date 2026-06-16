package splitbill

import (
	"context"
	"testing"
	"time"

	"affluena-api/internal/debt"
	"affluena-api/internal/transaction"

	"github.com/jackc/pgx/v5"
)

// mockTransactionRepo is a dummy repo for testing usecase validation
type mockTransactionRepo struct{}

func (m *mockTransactionRepo) CreateInTx(ctx context.Context, tx pgx.Tx, userID string, input transaction.TransactionInput) (transaction.Transaction, error) {
	return transaction.Transaction{ID: "tx-id", AmountMinor: input.AmountMinor}, nil
}

// mockDebtRepo is a dummy repo for testing usecase validation
type mockDebtRepo struct{}

func (m *mockDebtRepo) CreateInTx(ctx context.Context, tx pgx.Tx, userID string, input debt.DebtInput) (debt.Debt, error) {
	return debt.Debt{ID: "debt-id"}, nil
}

func TestSplitExpenseValidation(t *testing.T) {
	uc := &UseCase{
		pool:            nil, // We will just test validation before DB tx
		transactionRepo: &mockTransactionRepo{},
		debtRepo:        &mockDebtRepo{},
		activityUC:      nil,
	}

	tests := []struct {
		name        string
		totalAmount int64
		splits      []int64
		wantErr     error
	}{
		{
			name:        "partial split valid",
			totalAmount: 300000,
			splits:      []int64{100000, 100000},
			wantErr:     nil,
		},
		{
			name:        "full split valid",
			totalAmount: 200000,
			splits:      []int64{100000, 100000},
			wantErr:     nil,
		},
		{
			name:        "over split invalid",
			totalAmount: 100000,
			splits:      []int64{60000, 50000},
			wantErr:     ErrInvalidSplitAmount,
		},
		{
			name:        "zero split invalid",
			totalAmount: 100000,
			splits:      []int64{0},
			wantErr:     ErrInvalidSplitParticipantAmount,
		},
		{
			name:        "negative split invalid",
			totalAmount: 100000,
			splits:      []int64{-10000},
			wantErr:     ErrInvalidSplitParticipantAmount,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var splitReqs []TransactionSplit
			for _, amt := range tc.splits {
				splitReqs = append(splitReqs, TransactionSplit{
					CounterpartyName:       "Friend",
					AmountMinor:            amt,
					DisbursementCategoryID: "cat-1",
					PaymentCategoryID:      "cat-2",
				})
			}

			input := SplitTransactionInput{
				WalletID:         "wallet-1",
				CategoryID:       "cat-food",
				TotalAmountMinor: tc.totalAmount,
				TransactionAt:    time.Now().UTC(),
				Note:             "Dinner",
				Splits:           splitReqs,
			}

			defer func() {
				r := recover()
				if tc.wantErr == nil && r == nil {
					t.Errorf("expected panic for valid cases due to nil db pool, got none")
				}
			}()

			_, err := uc.SplitExpense(context.Background(), "user-1", input)
			if err != tc.wantErr {
				t.Errorf("expected error %v, got %v", tc.wantErr, err)
			}
		})
	}
}
