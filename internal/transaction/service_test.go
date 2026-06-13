package transaction

import "testing"

func TestBalanceDeltasForIncomeAndExpense(t *testing.T) {
	income, err := BalanceDeltas(TransactionInput{
		Type:           TransactionTypeIncome,
		WalletID:       "wallet-1",
		AmountMinor:    125_000,
		CategoryID:     "category-1",
		TransactionUTC: mustTestTime(t, "2026-06-12T10:00:00Z"),
	})
	if err != nil {
		t.Fatalf("income BalanceDeltas returned error: %v", err)
	}
	if len(income) != 1 || income[0].WalletID != "wallet-1" || income[0].AmountMinor != 125_000 {
		t.Fatalf("unexpected income deltas: %#v", income)
	}

	expense, err := BalanceDeltas(TransactionInput{
		Type:           TransactionTypeExpense,
		WalletID:       "wallet-1",
		AmountMinor:    50_000,
		CategoryID:     "category-1",
		TransactionUTC: mustTestTime(t, "2026-06-12T10:00:00Z"),
	})
	if err != nil {
		t.Fatalf("expense BalanceDeltas returned error: %v", err)
	}
	if len(expense) != 1 || expense[0].WalletID != "wallet-1" || expense[0].AmountMinor != -50_000 {
		t.Fatalf("unexpected expense deltas: %#v", expense)
	}
}

func TestBalanceDeltasForTransferAndAdjustment(t *testing.T) {
	transfer, err := BalanceDeltas(TransactionInput{
		Type:           TransactionTypeTransfer,
		WalletID:       "source-wallet",
		ToWalletID:     "destination-wallet",
		AmountMinor:    75_000,
		TransactionUTC: mustTestTime(t, "2026-06-12T10:00:00Z"),
	})
	if err != nil {
		t.Fatalf("transfer BalanceDeltas returned error: %v", err)
	}
	if len(transfer) != 2 {
		t.Fatalf("expected 2 transfer deltas, got %#v", transfer)
	}
	if transfer[0].WalletID != "source-wallet" || transfer[0].AmountMinor != -75_000 {
		t.Fatalf("unexpected source delta: %#v", transfer[0])
	}
	if transfer[1].WalletID != "destination-wallet" || transfer[1].AmountMinor != 75_000 {
		t.Fatalf("unexpected destination delta: %#v", transfer[1])
	}

	adjustment, err := BalanceDeltas(TransactionInput{
		Type:           TransactionTypeAdjustment,
		WalletID:       "wallet-1",
		AmountMinor:    -20_000,
		TransactionUTC: mustTestTime(t, "2026-06-12T10:00:00Z"),
	})
	if err != nil {
		t.Fatalf("adjustment BalanceDeltas returned error: %v", err)
	}
	if len(adjustment) != 1 || adjustment[0].WalletID != "wallet-1" || adjustment[0].AmountMinor != -20_000 {
		t.Fatalf("unexpected adjustment deltas: %#v", adjustment)
	}
}

func TestBalanceDeltasRejectInvalidInputs(t *testing.T) {
	cases := []TransactionInput{
		{Type: TransactionTypeIncome, WalletID: "", AmountMinor: 1, CategoryID: "category-1", TransactionUTC: mustTestTime(t, "2026-06-12T10:00:00Z")},
		{Type: TransactionTypeExpense, WalletID: "wallet-1", AmountMinor: 0, CategoryID: "category-1", TransactionUTC: mustTestTime(t, "2026-06-12T10:00:00Z")},
		{Type: TransactionTypeIncome, WalletID: "wallet-1", AmountMinor: -1, CategoryID: "category-1", TransactionUTC: mustTestTime(t, "2026-06-12T10:00:00Z")},
		{Type: TransactionTypeExpense, WalletID: "wallet-1", AmountMinor: -1, CategoryID: "category-1", TransactionUTC: mustTestTime(t, "2026-06-12T10:00:00Z")},
		{Type: TransactionTypeTransfer, WalletID: "wallet-1", ToWalletID: "wallet-2", AmountMinor: -1, TransactionUTC: mustTestTime(t, "2026-06-12T10:00:00Z")},
		{Type: TransactionTypeTransfer, WalletID: "wallet-1", ToWalletID: "", AmountMinor: 1, TransactionUTC: mustTestTime(t, "2026-06-12T10:00:00Z")},
		{Type: TransactionTypeTransfer, WalletID: "same-wallet", ToWalletID: "same-wallet", AmountMinor: 1, TransactionUTC: mustTestTime(t, "2026-06-12T10:00:00Z")},
		{Type: TransactionTypeIncome, WalletID: "wallet-1", AmountMinor: 1, CategoryID: "", TransactionUTC: mustTestTime(t, "2026-06-12T10:00:00Z")},
		{Type: TransactionTypeExpense, WalletID: "wallet-1", AmountMinor: 1, CategoryID: "", TransactionUTC: mustTestTime(t, "2026-06-12T10:00:00Z")},
		{Type: TransactionType("refund"), WalletID: "wallet-1", AmountMinor: 1, TransactionUTC: mustTestTime(t, "2026-06-12T10:00:00Z")},
		{Type: TransactionTypeAdjustment, WalletID: "wallet-1", AmountMinor: 1},
	}

	for _, tc := range cases {
		if _, err := BalanceDeltas(tc); err == nil {
			t.Fatalf("expected validation error for %#v", tc)
		}
	}
}

func TestBalanceDeltasAllowsPositiveAndNegativeAdjustments(t *testing.T) {
	tests := []struct {
		name        string
		amountMinor int64
	}{
		{name: "increase balance", amountMinor: 25_000},
		{name: "decrease balance", amountMinor: -25_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deltas, err := BalanceDeltas(TransactionInput{
				Type:           TransactionTypeAdjustment,
				WalletID:       "wallet-1",
				AmountMinor:    tt.amountMinor,
				TransactionUTC: mustTestTime(t, "2026-06-12T10:00:00Z"),
			})
			if err != nil {
				t.Fatalf("BalanceDeltas returned error: %v", err)
			}
			if len(deltas) != 1 || deltas[0].AmountMinor != tt.amountMinor {
				t.Fatalf("expected adjustment delta %d, got %#v", tt.amountMinor, deltas)
			}
		})
	}
}
