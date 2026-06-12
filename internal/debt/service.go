package debt

import (
	"errors"

	"affluena/internal/transaction"
)

type DebtType string

const (
	DebtTypeReceivable DebtType = "receivable"
	DebtTypePayable    DebtType = "payable"
)

type DebtStatus string

const (
	DebtStatusOpen      DebtStatus = "open"
	DebtStatusPartial   DebtStatus = "partial"
	DebtStatusPaidOff   DebtStatus = "paid_off"
	DebtStatusCancelled DebtStatus = "cancelled"
)

type DebtAction string

const (
	DebtActionOrigination DebtAction = "origination"
	DebtActionPayment     DebtAction = "payment"
)

var (
	errInvalidDebtType   = errors.New("invalid debt type")
	errInvalidDebtStatus = errors.New("invalid debt status")
	errInvalidDebtAction = errors.New("invalid debt action")
	errInvalidPayment    = errors.New("payment amount must be positive")
	errOverpayment       = errors.New("payment amount exceeds remaining amount")
	errInactiveDebt      = errors.New("debt is not payable")
)

type PaymentState struct {
	PrincipalAmountMinor int64      `json:"principal_amount_minor"`
	PaidAmountMinor      int64      `json:"paid_amount_minor"`
	RemainingAmountMinor int64      `json:"remaining_amount_minor"`
	Status               DebtStatus `json:"status"`
}

func IsValidDebtType(debtType DebtType) bool {
	switch debtType {
	case DebtTypeReceivable, DebtTypePayable:
		return true
	default:
		return false
	}
}

func IsValidDebtStatus(status DebtStatus) bool {
	switch status {
	case DebtStatusOpen, DebtStatusPartial, DebtStatusPaidOff, DebtStatusCancelled:
		return true
	default:
		return false
	}
}

func ResolveStatus(principalAmountMinor int64, paidAmountMinor int64, requested DebtStatus) (PaymentState, error) {
	if principalAmountMinor <= 0 {
		return PaymentState{}, errors.New("principal_amount_minor must be positive")
	}
	if paidAmountMinor < 0 || paidAmountMinor > principalAmountMinor {
		return PaymentState{}, errors.New("paid_amount_minor must be between 0 and principal_amount_minor")
	}
	if requested == "" {
		requested = statusForAmounts(principalAmountMinor, paidAmountMinor)
	}
	if !IsValidDebtStatus(requested) {
		return PaymentState{}, errInvalidDebtStatus
	}
	if requested != DebtStatusCancelled && requested != statusForAmounts(principalAmountMinor, paidAmountMinor) {
		return PaymentState{}, errors.New("debt status does not match payment progress")
	}
	return PaymentState{
		PrincipalAmountMinor: principalAmountMinor,
		PaidAmountMinor:      paidAmountMinor,
		RemainingAmountMinor: principalAmountMinor - paidAmountMinor,
		Status:               requested,
	}, nil
}

func ApplyPayment(state PaymentState, amountMinor int64) (PaymentState, error) {
	resolved, err := ResolveStatus(state.PrincipalAmountMinor, state.PaidAmountMinor, state.Status)
	if err != nil {
		return PaymentState{}, err
	}
	if resolved.Status == DebtStatusCancelled || resolved.Status == DebtStatusPaidOff {
		return PaymentState{}, errInactiveDebt
	}
	if amountMinor <= 0 {
		return PaymentState{}, errInvalidPayment
	}
	if amountMinor > resolved.RemainingAmountMinor {
		return PaymentState{}, errOverpayment
	}
	return ResolveStatus(resolved.PrincipalAmountMinor, resolved.PaidAmountMinor+amountMinor, "")
}

func TransactionTypeFor(debtType DebtType, action DebtAction) (transaction.TransactionType, error) {
	if !IsValidDebtType(debtType) {
		return "", errInvalidDebtType
	}
	switch action {
	case DebtActionOrigination:
		if debtType == DebtTypeReceivable {
			return transaction.TransactionTypeExpense, nil
		}
		return transaction.TransactionTypeIncome, nil
	case DebtActionPayment:
		if debtType == DebtTypeReceivable {
			return transaction.TransactionTypeIncome, nil
		}
		return transaction.TransactionTypeExpense, nil
	default:
		return "", errInvalidDebtAction
	}
}

func statusForAmounts(principalAmountMinor int64, paidAmountMinor int64) DebtStatus {
	switch {
	case paidAmountMinor == 0:
		return DebtStatusOpen
	case paidAmountMinor == principalAmountMinor:
		return DebtStatusPaidOff
	default:
		return DebtStatusPartial
	}
}
