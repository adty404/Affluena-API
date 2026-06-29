package partner

import (
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("partner link not found")
	ErrNotAuthorized = errors.New("not authorized")
	ErrAlreadyLinked = errors.New("partner already exists")
	ErrSelfInvite    = errors.New("cannot invite yourself")
	ErrUserNotFound  = errors.New("user with that email not found")
	// ErrPartnerLimit means the caller has reached the maximum number of active
	// (pending or joined) people they can share their wallets with. An existing
	// link must be revoked before inviting someone else. See maxActiveShares.
	ErrPartnerLimit = errors.New("share limit reached")
)

// Direction of a link relative to the caller.
const (
	// DirectionOwned: the caller is the owner; the other party (partner) can
	// view the caller's wallets.
	DirectionOwned = "owned"
	// DirectionIncoming: the caller is the partner; they can view the other
	// party's (owner's) wallets.
	DirectionIncoming = "incoming"
)

// Link is a single partner_links row as seen by the caller, carrying the OTHER
// party's identity (so the client can render names without extra lookups).
type Link struct {
	ID        string    `json:"id"`
	Direction string    `json:"direction"`
	Status    string    `json:"status"`
	UserID    string    `json:"user_id"` // the other party
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type InviteInput struct {
	Email string `json:"email" binding:"required,email"`
}

type RespondInput struct {
	Status string `json:"status" binding:"required,oneof=joined rejected"`
}
