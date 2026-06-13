package wallet

import (
	"context"
	"errors"
)

type RepositoryPort interface {
	Create(ctx context.Context, userID string, input CreateWalletInput) (Wallet, error)
	List(ctx context.Context, userID string) ([]Wallet, error)
	Get(ctx context.Context, userID string, id string) (Wallet, error)
	Update(ctx context.Context, userID string, id string, input UpdateWalletInput) (Wallet, error)
	Delete(ctx context.Context, userID string, id string) error
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

func (u *UseCase) List(ctx context.Context, userID string) ([]Wallet, error) {
	return u.repo.List(ctx, userID)
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
