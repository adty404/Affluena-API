package wallet

import (
	"context"
	"errors"

	"affluena-api/internal/activity"
	"affluena-api/internal/page"
)

type RepositoryPort interface {
	Create(ctx context.Context, userID string, input CreateWalletInput) (Wallet, error)
	List(ctx context.Context, userID string, pagination page.Params) (page.Result[Wallet], error)
	Get(ctx context.Context, userID string, id string) (Wallet, error)
	Update(ctx context.Context, userID string, id string, input UpdateWalletInput) (Wallet, error)
	Delete(ctx context.Context, userID string, id string) error
	AddMember(ctx context.Context, walletID string, userID string, status string, role string) error
	FindUserByEmail(ctx context.Context, email string) (string, error)
	RespondInvite(ctx context.Context, walletID string, userID string, status string) error
	GetAccessLevel(ctx context.Context, userID string, walletID string) (AccessLevel, error)
	GetMembers(ctx context.Context, walletID string) ([]WalletMember, error)
	GetAnalytics(ctx context.Context, userID string, walletID string, month string) (WalletAnalytics, error)
}

type UseCase struct {
	repo       RepositoryPort
	activityUC activity.UseCase
}

func NewUseCase(repo RepositoryPort, activityUC activity.UseCase) *UseCase {
	return &UseCase{repo: repo, activityUC: activityUC}
}

func (u *UseCase) Create(ctx context.Context, userID string, input CreateWalletInput) (Wallet, error) {
	if !IsPublicType(input.Type) {
		return Wallet{}, errors.New("invalid wallet type")
	}
	if input.CurrencyCode == "" {
		input.CurrencyCode = "IDR"
	}
	w, err := u.repo.Create(ctx, userID, input)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "CREATE", "WALLET", &w.ID, "Membuat dompet baru: "+input.Name)
	}
	return w, err
}

func (u *UseCase) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Wallet], error) {
	return u.repo.List(ctx, userID, pagination)
}

func (u *UseCase) Get(ctx context.Context, userID string, id string) (Wallet, error) {
	return u.repo.Get(ctx, userID, id)
}

func (u *UseCase) Update(ctx context.Context, userID string, id string, input UpdateWalletInput) (Wallet, error) {
	if !IsPublicType(input.Type) {
		return Wallet{}, errors.New("invalid wallet type")
	}
	current, err := u.repo.Get(ctx, userID, id)
	if err != nil {
		return Wallet{}, err
	}
	if current.GoalID != nil {
		return Wallet{}, ErrGoalWalletReadOnly
	}
	w, err := u.repo.Update(ctx, userID, id, input)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "UPDATE", "WALLET", &id, "Mengubah dompet: "+input.Name)
	}
	return w, err
}

func (u *UseCase) Delete(ctx context.Context, userID string, id string) error {
	current, err := u.repo.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	if current.GoalID != nil {
		return ErrGoalWalletReadOnly
	}
	err = u.repo.Delete(ctx, userID, id)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "DELETE", "WALLET", &id, "Menghapus dompet")
	}
	return err
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

	role := input.Role
	if role == "" {
		role = "member"
	}
	err = u.repo.AddMember(ctx, id, invitedUserID, "pending", role)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "INVITE", "WALLET", &id, "Mengundang "+input.Email+" ke dompet bersama")
	}
	return err
}

func (u *UseCase) RespondInvite(ctx context.Context, userID string, id string, memberID string, input RespondInviteInput) error {
	if memberID != userID {
		return ErrNotFound
	}
	err := u.repo.RespondInvite(ctx, id, userID, input.Status)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "UPDATE", "WALLET_MEMBER", &id, "Merespons undangan dompet bersama: "+input.Status)
	}
	return err
}

// GetAccessLevel returns user's access level for a wallet.
func (u *UseCase) GetAccessLevel(ctx context.Context, userID string, walletID string) (AccessLevel, error) {
	return u.repo.GetAccessLevel(ctx, userID, walletID)
}

// GetMembers returns the member list (owner + invitees) for a wallet.
// Caller must have access (owner or joined member).
func (u *UseCase) GetMembers(ctx context.Context, userID string, walletID string) ([]WalletMember, error) {
	access, err := u.repo.GetAccessLevel(ctx, userID, walletID)
	if err != nil {
		return nil, err
	}
	if access == AccessNone {
		return nil, ErrNotFound
	}
	return u.repo.GetMembers(ctx, walletID)
}

// GetAnalytics returns monthly inflow/outflow + last activity for a wallet.
// Caller must have access.
func (u *UseCase) GetAnalytics(ctx context.Context, userID string, walletID string, month string) (WalletAnalytics, error) {
	return u.repo.GetAnalytics(ctx, userID, walletID, month)
}
