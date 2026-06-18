package goal

import (
	"errors"
	"time"
)

var (
	ErrNotFound               = errors.New("goal not found")
	ErrNotAuthorized          = errors.New("not authorized to access this goal")
	ErrOwnerCannotRespond     = errors.New("forbidden: goal owner cannot respond to their own invitation")
	ErrInviteAlreadyJoined    = errors.New("cannot respond: already joined")
	ErrInviteAlreadyRejected  = errors.New("cannot respond: invitation already rejected")
	ErrInviteMemberIDMismatch = errors.New("forbidden: invitation member does not match authenticated user")
)

type Goal struct {
	ID                   string     `json:"id"`
	UserID               string     `json:"user_id"`
	Name                 string     `json:"name"`
	TargetAmountMinor    int64      `json:"target_amount_minor"`
	CollectedAmountMinor int64      `json:"collected_amount_minor"` // Computed from wallets
	Deadline             *time.Time `json:"deadline"`
	Status               string     `json:"status"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`

	// Optional relationships
	Members []GoalMember `json:"members,omitempty"`
}

type GoalMember struct {
	GoalID    string    `json:"goal_id"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateGoalInput struct {
	Name              string     `json:"name" binding:"required"`
	TargetAmountMinor int64      `json:"target_amount_minor" binding:"required,gt=0"`
	Deadline          *time.Time `json:"deadline" binding:"required"`
}

type UpdateGoalInput struct {
	Name              string     `json:"name" binding:"required"`
	TargetAmountMinor int64      `json:"target_amount_minor" binding:"required,gt=0"`
	Deadline          *time.Time `json:"deadline" binding:"required"`
}

type InviteMemberInput struct {
	Email string `json:"email" binding:"required,email"`
}

type RespondInviteInput struct {
	Status string `json:"status" binding:"required,oneof=joined rejected"`
}

func NotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
