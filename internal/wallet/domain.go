package wallet

import (
	"errors"
	"time"
)

var (
	ErrNotFound               = errors.New("wallet not found")
	ErrNotAuthorized          = errors.New("not authorized")
	ErrOwnerCannotRespond     = errors.New("owner cannot respond to invite")
	ErrInviteAlreadyJoined    = errors.New("already joined")
	ErrInviteAlreadyRejected  = errors.New("invite already rejected")
	ErrInviteMemberIDMismatch = errors.New("invitation member does not match authenticated user")
	ErrGoalWalletReadOnly     = errors.New("goal wallets are managed by financial goals")
)

type Wallet struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	CurrencyCode string    `json:"currency_code"`
	BalanceMinor int64     `json:"balance_minor"`
	GoalID       *string   `json:"goal_id,omitempty"`
	Role         string    `json:"role,omitempty"`
	ShareStatus  string    `json:"share_status,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Members []WalletMember `json:"members,omitempty"`
}

type WalletMember struct {
	WalletID  string    `json:"wallet_id"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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

type InviteMemberInput struct {
	Email string `json:"email" binding:"required,email"`
}

type RespondInviteInput struct {
	Status string `json:"status" binding:"required,oneof=joined rejected"`
}

func IsValidType(walletType string) bool {
	switch walletType {
	case "cash", "bank", "e_wallet", "investment", "goal":
		return true
	default:
		return false
	}
}

func IsPublicType(walletType string) bool {
	switch walletType {
	case "cash", "bank", "e_wallet", "investment":
		return true
	default:
		return false
	}
}

func NotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
