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
	cases := []InstallmentPaymentState{
		{RemainingMonths: 0, Status: InstallmentStatusPaidOff},
		{RemainingMonths: 1, Status: InstallmentStatusCancelled},
		{RemainingMonths: 0, Status: InstallmentStatusActive},
	}

	for _, tc := range cases {
		if _, err := ApplyInstallmentPayment(tc); err == nil {
			t.Fatalf("expected installment payment to fail for %#v", tc)
		}
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

func TestAdvanceSubscriptionDueDateRejectsInvalidCycle(t *testing.T) {
	if _, err := AdvanceSubscriptionDueDate(time.Now().UTC(), BillingCycle("yearly")); err == nil {
		t.Fatal("expected invalid billing cycle to fail")
	}
}

func TestValidateInstallmentPlan(t *testing.T) {
	if err := ValidateInstallmentPlan(600_000, 200_000, 3); err != nil {
		t.Fatalf("expected matching installment plan to pass: %v", err)
	}
	if err := ValidateInstallmentPlan(100_000, 60_000, 2); err == nil {
		t.Fatal("expected mismatched installment plan to fail")
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

	negative := -1
	if _, _, err := ResolveInstallmentRemainingAndStatus(3, &negative, InstallmentStatusActive); err == nil {
		t.Fatal("expected negative remaining months to fail")
	}

	tooMany := 4
	if _, _, err := ResolveInstallmentRemainingAndStatus(3, &tooMany, InstallmentStatusActive); err == nil {
		t.Fatal("expected remaining months above tenor to fail")
	}

	if _, _, err := ResolveInstallmentRemainingAndStatus(0, nil, InstallmentStatusActive); err == nil {
		t.Fatal("expected non-positive tenor to fail")
	}

	if _, _, err := ResolveInstallmentRemainingAndStatus(3, nil, InstallmentStatus("late")); err == nil {
		t.Fatal("expected invalid installment status to fail")
	}
}

func TestTrackerValidationHelpers(t *testing.T) {
	if !IsValidInstallmentStatus(InstallmentStatusActive) || !IsValidInstallmentStatus(InstallmentStatusPaidOff) || !IsValidInstallmentStatus(InstallmentStatusCancelled) {
		t.Fatal("expected known installment statuses to be valid")
	}
	if IsValidInstallmentStatus(InstallmentStatus("late")) {
		t.Fatal("expected unknown installment status to be invalid")
	}
	if !IsValidBillingCycle(BillingCycleWeekly) || !IsValidBillingCycle(BillingCycleMonthly) {
		t.Fatal("expected known billing cycles to be valid")
	}
	if IsValidBillingCycle(BillingCycle("yearly")) {
		t.Fatal("expected yearly billing cycle to be invalid")
	}
	if !IsValidSubscriptionStatus(SubscriptionStatusActive) || !IsValidSubscriptionStatus(SubscriptionStatusPaused) || !IsValidSubscriptionStatus(SubscriptionStatusCancelled) {
		t.Fatal("expected known subscription statuses to be valid")
	}
	if IsValidSubscriptionStatus(SubscriptionStatus("expired")) {
		t.Fatal("expected unknown subscription status to be invalid")
	}
}
