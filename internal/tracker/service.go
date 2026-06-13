package tracker

import (
	"errors"
	"time"

	"affluena-api/internal/caldate"
)

type InstallmentStatus string

const (
	InstallmentStatusActive    InstallmentStatus = "active"
	InstallmentStatusPaidOff   InstallmentStatus = "paid_off"
	InstallmentStatusCancelled InstallmentStatus = "cancelled"
)

type BillingCycle string

const (
	BillingCycleWeekly  BillingCycle = "weekly"
	BillingCycleMonthly BillingCycle = "monthly"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusPaused    SubscriptionStatus = "paused"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
)

type InstallmentPaymentState struct {
	RemainingMonths int
	Status          InstallmentStatus
}

func ApplyInstallmentPayment(state InstallmentPaymentState) (InstallmentPaymentState, error) {
	if state.Status != InstallmentStatusActive {
		return InstallmentPaymentState{}, errors.New("installment is not active")
	}
	if state.RemainingMonths <= 0 {
		return InstallmentPaymentState{}, errors.New("installment has no remaining months")
	}

	state.RemainingMonths--
	if state.RemainingMonths == 0 {
		state.Status = InstallmentStatusPaidOff
	}
	return state, nil
}

func AdvanceSubscriptionDueDate(due time.Time, cycle BillingCycle) (time.Time, error) {
	switch cycle {
	case BillingCycleWeekly:
		return due.AddDate(0, 0, 7), nil
	case BillingCycleMonthly:
		return caldate.AddMonthsClamped(due, 1), nil
	default:
		return time.Time{}, errors.New("invalid billing cycle")
	}
}

func ResolveInstallmentRemainingAndStatus(tenorMonths int, remainingMonths *int, status InstallmentStatus) (int, InstallmentStatus, error) {
	if tenorMonths <= 0 {
		return 0, "", errors.New("tenor_months must be positive")
	}
	if status == "" {
		status = InstallmentStatusActive
	}
	if !IsValidInstallmentStatus(status) {
		return 0, "", errors.New("invalid installment status")
	}

	remaining := tenorMonths
	if remainingMonths != nil {
		remaining = *remainingMonths
	}
	if remaining < 0 || remaining > tenorMonths {
		return 0, "", errors.New("remaining_months must be between 0 and tenor_months")
	}
	if status == InstallmentStatusActive && remaining == 0 {
		return 0, "", errors.New("active installment must have remaining months")
	}
	if status == InstallmentStatusPaidOff && remaining != 0 {
		return 0, "", errors.New("paid off installment cannot have remaining months")
	}
	return remaining, status, nil
}

func IsValidInstallmentStatus(status InstallmentStatus) bool {
	switch status {
	case InstallmentStatusActive, InstallmentStatusPaidOff, InstallmentStatusCancelled:
		return true
	default:
		return false
	}
}

func IsValidBillingCycle(cycle BillingCycle) bool {
	switch cycle {
	case BillingCycleWeekly, BillingCycleMonthly:
		return true
	default:
		return false
	}
}

func IsValidSubscriptionStatus(status SubscriptionStatus) bool {
	switch status {
	case SubscriptionStatusActive, SubscriptionStatusPaused, SubscriptionStatusCancelled:
		return true
	default:
		return false
	}
}
