package tracker

import (
	"testing"
	"time"
)

func TestApplyInstallmentPaymentMarksPaidOffAtZeroRemaining(t *testing.T) {
	next, err := ApplyInstallmentPayment(InstallmentPaymentState{
		RemainingMonths: 1,
		Status:          InstallmentStatusActive,
	})
	if err != nil {
		t.Fatalf("ApplyInstallmentPayment returned error: %v", err)
	}

	if next.RemainingMonths != 0 {
		t.Fatalf("expected remaining months 0, got %d", next.RemainingMonths)
	}
	if next.Status != InstallmentStatusPaidOff {
		t.Fatalf("expected status paid_off, got %s", next.Status)
	}
}

func TestApplyInstallmentPaymentRejectsPaidOffInstallment(t *testing.T) {
	_, err := ApplyInstallmentPayment(InstallmentPaymentState{
		RemainingMonths: 0,
		Status:          InstallmentStatusPaidOff,
	})
	if err == nil {
		t.Fatal("expected paid-off installment payment to fail")
	}
}

func TestAdvanceSubscriptionDueDate(t *testing.T) {
	due := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	monthly, err := AdvanceSubscriptionDueDate(due, BillingCycleMonthly)
	if err != nil {
		t.Fatalf("AdvanceSubscriptionDueDate monthly returned error: %v", err)
	}
	if !monthly.Equal(time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected monthly due date to clamp to 2026-02-28, got %s", monthly)
	}

	weekly, err := AdvanceSubscriptionDueDate(due, BillingCycleWeekly)
	if err != nil {
		t.Fatalf("AdvanceSubscriptionDueDate weekly returned error: %v", err)
	}
	if !weekly.Equal(time.Date(2026, 2, 7, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected weekly due date 2026-02-07, got %s", weekly)
	}
}

func TestResolveInstallmentRemainingAndStatus(t *testing.T) {
	remaining, status, err := ResolveInstallmentRemainingAndStatus(3, nil, "")
	if err != nil {
		t.Fatalf("ResolveInstallmentRemainingAndStatus returned error: %v", err)
	}
	if remaining != 3 || status != InstallmentStatusActive {
		t.Fatalf("expected active installment with 3 remaining months, got %d/%s", remaining, status)
	}

	zero := 0
	remaining, status, err = ResolveInstallmentRemainingAndStatus(3, &zero, InstallmentStatusPaidOff)
	if err != nil {
		t.Fatalf("ResolveInstallmentRemainingAndStatus paid off returned error: %v", err)
	}
	if remaining != 0 || status != InstallmentStatusPaidOff {
		t.Fatalf("expected paid off installment with 0 remaining months, got %d/%s", remaining, status)
	}
}

func TestResolveInstallmentRemainingAndStatusRejectsInconsistentState(t *testing.T) {
	zero := 0
	if _, _, err := ResolveInstallmentRemainingAndStatus(3, &zero, InstallmentStatusActive); err == nil {
		t.Fatal("expected active installment with zero remaining months to fail")
	}

	one := 1
	if _, _, err := ResolveInstallmentRemainingAndStatus(3, &one, InstallmentStatusPaidOff); err == nil {
		t.Fatal("expected paid off installment with remaining months to fail")
	}
}
