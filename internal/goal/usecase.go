package goal

import (
	"context"
	"errors"
)

type RepositoryPort interface {
	CreateWithOwnerWallet(ctx context.Context, userID string, input CreateGoalInput) (Goal, error)
	List(ctx context.Context, userID string) ([]Goal, error)
	Get(ctx context.Context, userID string, id string) (Goal, error)
	AddMember(ctx context.Context, goalID string, userID string, status string) error
	FindUserByEmail(ctx context.Context, email string) (string, error)
	RespondInvite(ctx context.Context, goalID string, userID string, status string) error
}

type Usecase struct {
	repo RepositoryPort
}

func NewUsecase(repo RepositoryPort) *Usecase {
	return &Usecase{repo: repo}
}

func (u *Usecase) Create(ctx context.Context, userID string, input CreateGoalInput) (Goal, error) {
	return u.repo.CreateWithOwnerWallet(ctx, userID, input)
}

func (u *Usecase) List(ctx context.Context, userID string) ([]Goal, error) {
	return u.repo.List(ctx, userID)
}

func (u *Usecase) Get(ctx context.Context, userID string, id string) (Goal, error) {
	return u.repo.Get(ctx, userID, id)
}

func (u *Usecase) InviteMember(ctx context.Context, userID string, id string, input InviteMemberInput) error {
	// check if user is the owner
	g, err := u.repo.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	if g.UserID != userID {
		return ErrNotAuthorized
	}

	// find the invited user
	invitedUserID, err := u.repo.FindUserByEmail(ctx, input.Email)
	if err != nil {
		return errors.New("user with that email not found")
	}
	if invitedUserID == userID {
		return errors.New("cannot invite yourself")
	}

	return u.repo.AddMember(ctx, id, invitedUserID, "pending")
}

func (u *Usecase) RespondInvite(ctx context.Context, userID string, id string, memberID string, input RespondInviteInput) error {
	if memberID != userID {
		return ErrNotFound
	}
	return u.repo.RespondInvite(ctx, id, userID, input.Status)
}
