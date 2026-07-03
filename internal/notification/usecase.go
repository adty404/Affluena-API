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
	rules, err := u.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	// The DB stores English title/description; present them in Indonesian
	// (keyed on the stable rule_key) so the settings screen is fully id_ID.
	for i := range rules {
		rules[i] = LocalizeID(rules[i])
	}
	return rules, nil
}

func (u *useCase) Update(ctx context.Context, userID, id string, update NotificationRuleUpdate) (NotificationRule, error) {
	if update.Channel != nil && !IsValidChannel(*update.Channel) {
		return NotificationRule{}, errors.New("invalid channel")
	}
	rule, err := u.repo.Update(ctx, userID, id, update)
	if err != nil {
		return NotificationRule{}, err
	}
	return LocalizeID(rule), nil
}
