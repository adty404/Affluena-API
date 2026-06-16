package transaction

import (
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("resource not found")
	ErrUnauthorized = errors.New("forbidden: only transaction creator can modify or delete")
)

type Transaction struct {
	ID            string          `json:"id"`
	UserID        string          `json:"user_id"`
	Type          TransactionType `json:"type"`
	WalletID      string          `json:"wallet_id"`
	ToWalletID    string          `json:"to_wallet_id,omitempty"`
	CategoryID    string          `json:"category_id,omitempty"`
	AmountMinor   int64           `json:"amount_minor"`
	TagIDs        []string        `json:"tag_ids"`
	TransactionAt time.Time       `json:"transaction_at"`
	Note          string          `json:"note"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type TransactionFilter struct {
	Type       TransactionType
	WalletID   string
	CategoryID string
	TagID      string
	From       time.Time
	To         time.Time
}

func (t Transaction) Input() TransactionInput {
	return TransactionInput{
		Type:           t.Type,
		WalletID:       t.WalletID,
		ToWalletID:     t.ToWalletID,
		CategoryID:     t.CategoryID,
		AmountMinor:    t.AmountMinor,
		TagIDs:         t.TagIDs,
		TransactionUTC: t.TransactionAt,
		Note:           t.Note,
	}
}

func NotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
