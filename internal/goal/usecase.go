package goal

import (
	"context"
	"errors"
	"fmt"

	"affluena-api/internal/wallet"
)

type Usecase struct {
	repo       *Repository
	walletRepo *wallet.Repository
}

func NewUsecase(repo *Repository, walletRepo *wallet.Repository) *Usecase {
	return &Usecase{repo: repo, walletRepo: walletRepo}
}

func (u *Usecase) Create(ctx context.Context, userID string, input CreateGoalInput) (Goal, error) {
	g, err := u.repo.Create(ctx, userID, input)
	if err != nil {
		return Goal{}, err
	}

	// Add owner as member
	if err := u.repo.AddMember(ctx, g.ID, userID, "joined"); err != nil {
		return Goal{}, err
	}

	// Create a Wallet for the owner
	goalID := g.ID
	walletInput := wallet.CreateWalletInput{
		Name:         fmt.Sprintf("[Goal] %s", g.Name),
		Type:         "goal",
		CurrencyCode: "IDR",
		BalanceMinor: 0,
		GoalID:       &goalID,
	}
	if _, err := u.walletRepo.Create(ctx, userID, walletInput); err != nil {
		return Goal{}, err
	}

	return u.repo.Get(ctx, userID, g.ID)
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

func (u *Usecase) RespondInvite(ctx context.Context, userID string, id string, input RespondInviteInput) error {
	// check if member exists
	members, err := u.repo.GetMembers(ctx, id)
	if err != nil {
		return err
	}
	var isMember bool
	for _, m := range members {
		if m.UserID == userID {
			if m.Status == "joined" && input.Status == "joined" {
				return errors.New("already joined")
			}
			isMember = true
			break
		}
	}
	if !isMember {
		return ErrNotFound
	}

	if err := u.repo.UpdateMemberStatus(ctx, id, userID, input.Status); err != nil {
		return err
	}

	if input.Status == "joined" {
		// Create wallet for the new joined member
		g, err := u.repo.Get(ctx, userID, id)
		if err == nil {
			goalID := g.ID
			walletInput := wallet.CreateWalletInput{
				Name:         fmt.Sprintf("[Goal] %s", g.Name),
				Type:         "goal",
				CurrencyCode: "IDR",
				BalanceMinor: 0,
				GoalID:       &goalID,
			}
			// ignore error if they already have one with same name
			u.walletRepo.Create(ctx, userID, walletInput)
		}
	}
	return nil
}
