package export

import (
	"errors"
	"time"
)

var ErrJobNotFound = errors.New("export job not found")

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

type ExportJob struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Format    string     `json:"format"`
	FromAt    *time.Time `json:"from_at"`
	ToAt      *time.Time `json:"to_at"`
	RowCount  int        `json:"row_count"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}
