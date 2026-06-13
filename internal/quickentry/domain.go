package quickentry

import (
	"errors"
	"time"

	"affluena/internal/transaction"
)

var ErrNotFound = errors.New("quick entry template not found")

type Template struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	WalletID    string    `json:"wallet_id"`
	ToWalletID  string    `json:"to_wallet_id,omitempty"`
	CategoryID  string    `json:"category_id,omitempty"`
	AmountMinor int64     `json:"amount_minor"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ExecuteInput struct {
	TransactionAt time.Time
	Note          string
}

type ExecuteResult struct {
	Transaction transaction.Transaction `json:"transaction"`
}

func NotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
