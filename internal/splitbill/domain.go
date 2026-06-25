package splitbill

import (
	"errors"
	"time"
)

// ErrNotFound is returned when a split bill (origination transaction with debts)
// cannot be found for the user.
var ErrNotFound = errors.New("split bill not found")

type SplitTransactionRequest struct {
	WalletID         string             `json:"wallet_id" binding:"required"`
	CategoryID       string             `json:"category_id"`
	TotalAmountMinor int64              `json:"total_amount_minor" binding:"required,gt=0"`
	TransactionAt    string             `json:"transaction_at"`
	Note             string             `json:"note"`
	TagIDs           []string           `json:"tag_ids"`
	Splits           []TransactionSplit `json:"splits" binding:"required,dive,required"`
}

type TransactionSplit struct {
	CounterpartyName       string `json:"counterparty_name" binding:"required"`
	AmountMinor            int64  `json:"amount_minor" binding:"required,gt=0"`
	DisbursementCategoryID string `json:"disbursement_category_id" binding:"required"`
	PaymentCategoryID      string `json:"payment_category_id" binding:"required"`
}

type SplitTransactionResponse struct {
	TransactionID string   `json:"transaction_id"`
	DebtIDs       []string `json:"debt_ids"`
}

type SplitTransactionInput struct {
	WalletID         string
	CategoryID       string
	TotalAmountMinor int64
	TransactionAt    time.Time
	Note             string
	TagIDs           []string
	Splits           []TransactionSplit
}

// SplitBillStatus reflects whether any participant still owes money.
const (
	SplitBillStatusOngoing = "ongoing"
	SplitBillStatusSettled = "settled"
)

// SplitBillSummary aggregates one split bill (an origination expense transaction
// plus its receivable debts) for the list view.
type SplitBillSummary struct {
	TransactionID       string    `json:"transaction_id"`
	Note                string    `json:"note"`
	TotalAmountMinor    int64     `json:"total_amount_minor"`
	TransactionAt       time.Time `json:"transaction_at"`
	ParticipantCount    int       `json:"participant_count"`
	SettledCount        int       `json:"settled_count"`
	TotalOwedMinor      int64     `json:"total_owed_minor"`
	TotalRemainingMinor int64     `json:"total_remaining_minor"`
	Status              string    `json:"status"`
}

// SplitBillParticipant is one receivable debt within a split bill.
type SplitBillParticipant struct {
	DebtID               string     `json:"debt_id"`
	CounterpartyName     string     `json:"counterparty_name"`
	PrincipalAmountMinor int64      `json:"principal_amount_minor"`
	PaidAmountMinor      int64      `json:"paid_amount_minor"`
	RemainingAmountMinor int64      `json:"remaining_amount_minor"`
	Status               string     `json:"status"`
	DueDate              *time.Time `json:"due_date,omitempty"`
}

// SplitBillDetail is a single split bill with its participant debts.
type SplitBillDetail struct {
	TransactionID       string                 `json:"transaction_id"`
	Note                string                 `json:"note"`
	TotalAmountMinor    int64                  `json:"total_amount_minor"`
	TransactionAt       time.Time              `json:"transaction_at"`
	ParticipantCount    int                    `json:"participant_count"`
	SettledCount        int                    `json:"settled_count"`
	TotalOwedMinor      int64                  `json:"total_owed_minor"`
	TotalRemainingMinor int64                  `json:"total_remaining_minor"`
	Status              string                 `json:"status"`
	Participants        []SplitBillParticipant `json:"participants"`
}
