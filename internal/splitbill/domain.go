package splitbill

import "time"

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
