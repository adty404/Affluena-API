package goal

import (
	"context"
	"errors"

	"affluena-api/internal/activity"
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
	repo       RepositoryPort
	activityUC activity.UseCase
}

func NewUsecase(repo RepositoryPort, activityUC activity.UseCase) *Usecase {
	return &Usecase{repo: repo, activityUC: activityUC}
}

func (u *Usecase) Create(ctx context.Context, userID string, input CreateGoalInput) (Goal, error) {
	g, err := u.repo.CreateWithOwnerWallet(ctx, userID, input)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "CREATE", "GOAL", &g.ID, "Membuat tabungan bersama: "+input.Name)
	}
	return g, err
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

	err = u.repo.AddMember(ctx, id, invitedUserID, "pending")
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "INVITE", "GOAL", &id, "Mengundang "+input.Email+" ke tabungan bersama")
	}
	return err
}

func (u *Usecase) RespondInvite(ctx context.Context, userID string, id string, memberID string, input RespondInviteInput) error {
	if memberID != userID {
		return ErrNotFound
	}
	err := u.repo.RespondInvite(ctx, id, userID, input.Status)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "UPDATE", "GOAL_MEMBER", &id, "Merespons undangan tabungan bersama: "+input.Status)
	}
	return err
}
