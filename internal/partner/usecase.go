package partner

import (
	"context"

	"affluena-api/internal/activity"
)

type RepositoryPort interface {
	FindUserByEmail(ctx context.Context, email string) (string, error)
	Invite(ctx context.Context, ownerID, partnerID string) error
	Respond(ctx context.Context, linkID, partnerID, status string) error
	List(ctx context.Context, userID string) ([]Link, error)
	Revoke(ctx context.Context, linkID, userID string) error
}

type UseCase struct {
	repo       RepositoryPort
	activityUC activity.UseCase
}

func NewUseCase(repo RepositoryPort, activityUC activity.UseCase) *UseCase {
	return &UseCase{repo: repo, activityUC: activityUC}
}

// Invite links a partner (by email) so they can view all of the caller's
// wallets once they accept.
func (u *UseCase) Invite(ctx context.Context, userID string, input InviteInput) error {
	partnerID, err := u.repo.FindUserByEmail(ctx, input.Email)
	if err != nil {
		return err
	}
	if partnerID == userID {
		return ErrSelfInvite
	}
	err = u.repo.Invite(ctx, userID, partnerID)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "INVITE", "PARTNER", nil, "Mengundang "+input.Email+" sebagai pasangan")
	}
	return err
}

// Respond accepts or rejects an incoming partner invite (caller must be the
// invited partner). Accepting grants viewer access to all the owner's wallets.
func (u *UseCase) Respond(ctx context.Context, userID, linkID string, input RespondInput) error {
	err := u.repo.Respond(ctx, linkID, userID, input.Status)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "UPDATE", "PARTNER", &linkID, "Merespons undangan pasangan: "+input.Status)
	}
	return err
}

func (u *UseCase) List(ctx context.Context, userID string) ([]Link, error) {
	return u.repo.List(ctx, userID)
}

// Revoke removes a partner link (either party may do so) and the viewer access
// it granted.
func (u *UseCase) Revoke(ctx context.Context, userID, linkID string) error {
	err := u.repo.Revoke(ctx, linkID, userID)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "DELETE", "PARTNER", &linkID, "Memutus tautan pasangan")
	}
	return err
}
