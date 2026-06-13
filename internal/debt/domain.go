package debt

import (
	"errors"
	"time"

	"affluena/internal/transaction"
)

var ErrNotFound = errors.New("debt not found")

type Debt struct {
	ID                       string     `json:"id"`
	UserID                   string     `json:"user_id"`
	Type                     DebtType   `json:"type"`
	CounterpartyName         string     `json:"counterparty_name"`
	WalletID                 string     `json:"wallet_id"`
	DisbursementCategoryID   string     `json:"disbursement_category_id"`
	PaymentCategoryID        string     `json:"payment_category_id"`
	OriginationTransactionID string     `json:"origination_transaction_id"`
	PrincipalAmountMinor     int64      `json:"principal_amount_minor"`
	PaidAmountMinor          int64      `json:"paid_amount_minor"`
	RemainingAmountMinor     int64      `json:"remaining_amount_minor"`
	OpenedAt                 time.Time  `json:"opened_at"`
	DueDate                  *time.Time `json:"due_date,omitempty"`
	Status                   DebtStatus `json:"status"`
	Note                     string     `json:"note"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

type DebtInput struct {
	Type                   DebtType
	CounterpartyName       string
	WalletID               string
	DisbursementCategoryID string
	PaymentCategoryID      string
	PrincipalAmountMinor   int64
	OpenedAt               time.Time
	DueDate                *time.Time
	Note                   string
}

type DebtUpdate struct {
	CounterpartyName string
	DueDate          *time.Time
	Status           DebtStatus
	Note             string
}

type DebtPayment struct {
	ID            string                  `json:"id"`
	UserID        string                  `json:"user_id"`
	DebtID        string                  `json:"debt_id"`
	TransactionID string                  `json:"transaction_id"`
	AmountMinor   int64                   `json:"amount_minor"`
	PaidAt        time.Time               `json:"paid_at"`
	Note          string                  `json:"note"`
	CreatedAt     time.Time               `json:"created_at"`
	Debt          Debt                    `json:"debt"`
	Transaction   transaction.Transaction `json:"transaction"`
}

func NotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
