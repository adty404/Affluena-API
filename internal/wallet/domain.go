package wallet

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("wallet not found")

type Wallet struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	CurrencyCode string    `json:"currency_code"`
	BalanceMinor int64     `json:"balance_minor"`
	GoalID       *string   `json:"goal_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateWalletInput struct {
	Name         string
	Type         string
	CurrencyCode string
	BalanceMinor int64
	GoalID       *string
}

type UpdateWalletInput struct {
	Name         string
	Type         string
	CurrencyCode string
}

func IsValidType(walletType string) bool {
	switch walletType {
	case "cash", "bank", "e_wallet", "investment", "goal":
		return true
	default:
		return false
	}
}

func NotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
