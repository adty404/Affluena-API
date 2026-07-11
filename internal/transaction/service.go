package transaction

import (
	"errors"
	"time"
)

type TransactionType string

const (
	TransactionTypeIncome     TransactionType = "income"
	TransactionTypeExpense    TransactionType = "expense"
	TransactionTypeTransfer   TransactionType = "transfer"
	TransactionTypeAdjustment TransactionType = "adjustment"
)

func IsValidType(transactionType TransactionType) bool {
	switch transactionType {
	case TransactionTypeIncome, TransactionTypeExpense, TransactionTypeTransfer, TransactionTypeAdjustment:
		return true
	default:
		return false
	}
}

type TransactionInput struct {
	Type        TransactionType
	WalletID    string
	ToWalletID  string
	CategoryID  string
	AmountMinor int64
	// FeeMinor is an optional bank admin fee charged to the SOURCE wallet on
	// top of AmountMinor. It is only meaningful for transfers; non-transfer
	// types must leave it 0 (enforced at the handler). Must be >= 0.
	FeeMinor       int64
	TagIDs         []string
	TransactionUTC time.Time
	Note           string
}

type BalanceDelta struct {
	WalletID    string
	AmountMinor int64
}

func BalanceDeltas(input TransactionInput) ([]BalanceDelta, error) {
	if input.WalletID == "" {
		return nil, errors.New("wallet_id is required")
	}
	if input.AmountMinor == 0 {
		return nil, errors.New("amount_minor must not be zero")
	}
	if input.TransactionUTC.IsZero() {
		return nil, errors.New("transaction_at is required")
	}

	switch input.Type {
	case TransactionTypeIncome:
		if input.AmountMinor < 0 {
			return nil, errors.New("income amount_minor must be positive")
		}
		if input.CategoryID == "" {
			return nil, errors.New("category_id is required")
		}
		return []BalanceDelta{{WalletID: input.WalletID, AmountMinor: input.AmountMinor}}, nil
	case TransactionTypeExpense:
		if input.AmountMinor < 0 {
			return nil, errors.New("expense amount_minor must be positive")
		}
		if input.CategoryID == "" {
			return nil, errors.New("category_id is required")
		}
		return []BalanceDelta{{WalletID: input.WalletID, AmountMinor: -input.AmountMinor}}, nil
	case TransactionTypeTransfer:
		if input.AmountMinor < 0 {
			return nil, errors.New("transfer amount_minor must be positive")
		}
		if input.FeeMinor < 0 {
			return nil, errors.New("fee_minor must not be negative")
		}
		if input.ToWalletID == "" {
			return nil, errors.New("to_wallet_id is required")
		}
		if input.WalletID == input.ToWalletID {
			return nil, errors.New("wallet_id and to_wallet_id must differ")
		}
		// The admin fee is a real loss to the bank: the source loses
		// amount+fee while the destination only gains amount, so the sum of
		// all wallet balances drops by exactly fee (net-worth-correct).
		return []BalanceDelta{
			{WalletID: input.WalletID, AmountMinor: -(input.AmountMinor + input.FeeMinor)},
			{WalletID: input.ToWalletID, AmountMinor: input.AmountMinor},
		}, nil
	case TransactionTypeAdjustment:
		return []BalanceDelta{{WalletID: input.WalletID, AmountMinor: input.AmountMinor}}, nil
	default:
		return nil, errors.New("invalid transaction type")
	}
}
