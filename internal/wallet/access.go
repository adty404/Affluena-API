package wallet

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrAccessDenied = errors.New("access denied")
)

// AccessChecker provides wallet access validation.
type AccessChecker interface {
	GetAccessLevel(ctx context.Context, userID string, walletID string) (AccessLevel, error)
}

// AccessLevel represents user's access to a wallet.
type AccessLevel int

const (
	AccessNone   AccessLevel = iota // No access
	AccessViewer                    // Read only: can view, cannot record
	AccessMember                    // Read/write, cannot manage
	AccessOwner                     // Full control
)

// CheckOwnerAccess verifies user is wallet owner.
func CheckOwnerAccess(checker AccessChecker, ctx context.Context, userID, walletID string) error {
	level, err := checker.GetAccessLevel(ctx, userID, walletID)
	if err != nil {
		return err
	}
	if level != AccessOwner {
		return ErrAccessDenied
	}
	return nil
}

// CheckMemberAccess verifies user has at least member (read/write) access.
// Viewers (read-only) are rejected — use this to gate any write to a wallet.
func CheckMemberAccess(checker AccessChecker, ctx context.Context, userID, walletID string) error {
	level, err := checker.GetAccessLevel(ctx, userID, walletID)
	if err != nil {
		return err
	}
	if level < AccessMember {
		return ErrAccessDenied
	}
	return nil
}

// CheckViewAccess verifies user can at least view the wallet (viewer, member,
// or owner). Use this to gate read-only access.
func CheckViewAccess(checker AccessChecker, ctx context.Context, userID, walletID string) error {
	level, err := checker.GetAccessLevel(ctx, userID, walletID)
	if err != nil {
		return err
	}
	if level < AccessViewer {
		return ErrAccessDenied
	}
	return nil
}

// OwnerOnlyError wraps operation that requires owner access.
func OwnerOnlyError(operation string) error {
	return fmt.Errorf("only creator can %s wallet", operation)
}
