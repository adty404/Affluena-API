package debt

import (
	"testing"

	"affluena/internal/transaction"
)

func TestApplyPaymentTransitionsDebtStatus(t *testing.T) {
	first, err := ApplyPayment(PaymentState{
		PrincipalAmountMinor: 100_000,
		PaidAmountMinor:      0,
		Status:               DebtStatusOpen,
	}, 40_000)
	if err != nil {
		t.Fatalf("ApplyPayment first payment returned error: %v", err)
	}
	if first.PaidAmountMinor != 40_000 || first.RemainingAmountMinor != 60_000 || first.Status != DebtStatusPartial {
		t.Fatalf("expected partial state with 40k paid and 60k remaining, got %+v", first)
	}

	second, err := ApplyPayment(first, 60_000)
	if err != nil {
		t.Fatalf("ApplyPayment final payment returned error: %v", err)
	}
	if second.PaidAmountMinor != 100_000 || second.RemainingAmountMinor != 0 || second.Status != DebtStatusPaidOff {
		t.Fatalf("expected paid off state with zero remaining, got %+v", second)
	}
}

func TestApplyPaymentRejectsInvalidPayment(t *testing.T) {
	state := PaymentState{
		PrincipalAmountMinor: 100_000,
		PaidAmountMinor:      75_000,
		Status:               DebtStatusPartial,
	}
	if _, err := ApplyPayment(state, 0); err == nil {
		t.Fatal("expected zero payment to fail")
	}
	if _, err := ApplyPayment(state, 25_001); err == nil {
		t.Fatal("expected overpayment to fail")
	}

	cancelled := PaymentState{
		PrincipalAmountMinor: 100_000,
		PaidAmountMinor:      0,
		Status:               DebtStatusCancelled,
	}
	if _, err := ApplyPayment(cancelled, 10_000); err == nil {
		t.Fatal("expected cancelled debt payment to fail")
	}
}

func TestDebtTransactionTypes(t *testing.T) {
	tests := []struct {
		name     string
		debtType DebtType
		action   DebtAction
		wantType transaction.TransactionType
		wantErr  bool
	}{
		{
			name:     "receivable origination is expense",
			debtType: DebtTypeReceivable,
			action:   DebtActionOrigination,
			wantType: transaction.TransactionTypeExpense,
		},
		{
			name:     "receivable payment is income",
			debtType: DebtTypeReceivable,
			action:   DebtActionPayment,
			wantType: transaction.TransactionTypeIncome,
		},
		{
			name:     "payable origination is income",
			debtType: DebtTypePayable,
			action:   DebtActionOrigination,
			wantType: transaction.TransactionTypeIncome,
		},
		{
			name:     "payable payment is expense",
			debtType: DebtTypePayable,
			action:   DebtActionPayment,
			wantType: transaction.TransactionTypeExpense,
		},
		{
			name:     "invalid type fails",
			debtType: DebtType("bad"),
			action:   DebtActionPayment,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TransactionTypeFor(tt.debtType, tt.action)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("TransactionTypeFor returned error: %v", err)
			}
			if got != tt.wantType {
				t.Fatalf("expected %s, got %s", tt.wantType, got)
			}
		})
	}
}
