package wallet

import (
	"context"
	"errors"

	"affluena-api/internal/page"
)

type RepositoryPort interface {
	Create(ctx context.Context, userID string, input CreateWalletInput) (Wallet, error)
	List(ctx context.Context, userID string, pagination page.Params) (page.Result[Wallet], error)
	Get(ctx context.Context, userID string, id string) (Wallet, error)
	Update(ctx context.Context, userID string, id string, input UpdateWalletInput) (Wallet, error)
	Delete(ctx context.Context, userID string, id string) error
	AddMember(ctx context.Context, walletID string, userID string, status string) error
	FindUserByEmail(ctx context.Context, email string) (string, error)
	RespondInvite(ctx context.Context, walletID string, userID string, status string) error
}

type UseCase struct {
	repo RepositoryPort
}

func NewUseCase(repo RepositoryPort) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) Create(ctx context.Context, userID string, input CreateWalletInput) (Wallet, error) {
	if !IsValidType(input.Type) {
		return Wallet{}, errors.New("invalid wallet type")
	}
	if input.CurrencyCode == "" {
		input.CurrencyCode = "IDR"
	}
	return u.repo.Create(ctx, userID, input)
}

func (u *UseCase) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Wallet], error) {
	return u.repo.List(ctx, userID, pagination)
}

func (u *UseCase) Get(ctx context.Context, userID string, id string) (Wallet, error) {
	return u.repo.Get(ctx, userID, id)
}

func (u *UseCase) Update(ctx context.Context, userID string, id string, input UpdateWalletInput) (Wallet, error) {
	if !IsValidType(input.Type) {
		return Wallet{}, errors.New("invalid wallet type")
	}
	return u.repo.Update(ctx, userID, id, input)
}

func (u *UseCase) Delete(ctx context.Context, userID string, id string) error {
	return u.repo.Delete(ctx, userID, id)
}

func (u *UseCase) InviteMember(ctx context.Context, userID string, id string, input InviteMemberInput) error {
	// check if user is the owner
	w, err := u.repo.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	if w.UserID != userID {
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

func (u *UseCase) RespondInvite(ctx context.Context, userID string, id string, memberID string, input RespondInviteInput) error {
	if memberID != userID {
		return ErrNotFound
	}
	return u.repo.RespondInvite(ctx, id, userID, input.Status)
}
