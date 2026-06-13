package export

import (
	"time"
)

type ExportOptions struct {
	From time.Time
	To   time.Time
}

type TransactionExportRow struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	AmountMinor   int64     `json:"amount_minor"`
	TransactionAt time.Time `json:"transaction_at"`
	Note          string    `json:"note"`
	WalletName    string    `json:"wallet_name"`
	ToWalletName  string    `json:"to_wallet_name"`
	CategoryName  string    `json:"category_name"`
	Tags          string    `json:"tags"`
	CreatedAt     time.Time `json:"created_at"`
}
