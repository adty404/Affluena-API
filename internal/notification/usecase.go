package notification

import (
	"context"
	"errors"
)

type UseCase interface {
	List(ctx context.Context, userID string) ([]NotificationRule, error)
	Update(ctx context.Context, userID, id string, update NotificationRuleUpdate) (NotificationRule, error)
}

type useCase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &useCase{repo: repo}
}

func (u *useCase) List(ctx context.Context, userID string) ([]NotificationRule, error) {
	if err := u.repo.EnsureDefaults(ctx, userID); err != nil {
		return nil, err
	}
	return u.repo.List(ctx, userID)
}

func (u *useCase) Update(ctx context.Context, userID, id string, update NotificationRuleUpdate) (NotificationRule, error) {
	if update.Channel != nil && !IsValidChannel(*update.Channel) {
		return NotificationRule{}, errors.New("invalid channel")
	}
	return u.repo.Update(ctx, userID, id, update)
}
