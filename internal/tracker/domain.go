package tracker

import (
	"time"

	"affluena-api/internal/transaction"
)

type Installment struct {
	ID                 string            `json:"id"`
	UserID             string            `json:"user_id"`
	Name               string            `json:"name"`
	WalletID           string            `json:"wallet_id"`
	CategoryID         string            `json:"category_id"`
	TotalAmountMinor   int64             `json:"total_amount_minor"`
	MonthlyAmountMinor int64             `json:"monthly_amount_minor"`
	TenorMonths        int               `json:"tenor_months"`
	RemainingMonths    int               `json:"remaining_months"`
	StartDate          time.Time         `json:"start_date"`
	DueDay             int               `json:"due_day"`
	Status             InstallmentStatus `json:"status"`
	Note               string            `json:"note"`
	Color              string            `json:"color"`
	Icon               string            `json:"icon"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

type InstallmentPayment struct {
	Installment Installment             `json:"installment"`
	Transaction transaction.Transaction `json:"transaction"`
}

type Subscription struct {
	ID            string             `json:"id"`
	UserID        string             `json:"user_id"`
	Name          string             `json:"name"`
	AccountDetail string             `json:"account_detail"`
	WalletID      string             `json:"wallet_id"`
	CategoryID    string             `json:"category_id"`
	AmountMinor   int64              `json:"amount_minor"`
	BillingCycle  BillingCycle       `json:"billing_cycle"`
	NextDueDate   time.Time          `json:"next_due_date"`
	Status        SubscriptionStatus `json:"status"`
	Note          string             `json:"note"`
	Color         string             `json:"color"`
	Icon          string             `json:"icon"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

type SubscriptionPayment struct {
	Subscription Subscription            `json:"subscription"`
	Transaction  transaction.Transaction `json:"transaction"`
}
