package notification_test

import (
	"context"
	"testing"

	"affluena-api/internal/notification"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	rules []notification.NotificationRule
}

func (m *mockRepo) List(ctx context.Context, userID string) ([]notification.NotificationRule, error) {
	var res []notification.NotificationRule
	for _, r := range m.rules {
		if r.UserID == userID {
			res = append(res, r)
		}
	}
	return res, nil
}

func (m *mockRepo) Update(ctx context.Context, userID, id string, update notification.NotificationRuleUpdate) (notification.NotificationRule, error) {
	for i, r := range m.rules {
		if r.ID == id && r.UserID == userID {
			if update.Enabled != nil {
				m.rules[i].Enabled = *update.Enabled
			}
			if update.Channel != nil {
				m.rules[i].Channel = *update.Channel
			}
			return m.rules[i], nil
		}
	}
	return notification.NotificationRule{}, notification.ErrNotFound
}

func (m *mockRepo) EnsureDefaults(ctx context.Context, userID string) error {
	defaults := []notification.NotificationRule{
		{RuleKey: "budget-alert", Title: "Budget threshold alert", Description: "Notify when category budget reaches 80% and 100%.", Enabled: true, Channel: "both", Tone: "orange"},
		{RuleKey: "due-reminder", Title: "Due date reminder", Description: "Debt, installment, and subscription reminders at H-3 and H-1.", Enabled: true, Channel: "both", Tone: "blue"},
		{RuleKey: "recurring-run", Title: "Recurring run result", Description: "Notify when recurring transaction runs or fails.", Enabled: true, Channel: "in-app", Tone: "green"},
		{RuleKey: "security-alert", Title: "Security alert", Description: "Notify login from a new device or location.", Enabled: true, Channel: "email", Tone: "red"},
		{RuleKey: "weekly-summary", Title: "Weekly finance summary", Description: "Send a weekly cashflow, budget, and goal summary.", Enabled: false, Channel: "email", Tone: "purple"},
	}

	existing := make(map[string]bool)
	for _, r := range m.rules {
		if r.UserID == userID {
			existing[r.RuleKey] = true
		}
	}

	for _, d := range defaults {
		if !existing[d.RuleKey] {
			d.ID = uuid.NewString()
			d.UserID = userID
			m.rules = append(m.rules, d)
		}
	}
	return nil
}

func TestUseCase_List(t *testing.T) {
	repo := &mockRepo{}
	uc := notification.NewUseCase(repo)
	userID := uuid.NewString()

	// First call should seed defaults
	rules, err := uc.List(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, rules, 5)

	// Second call should return the same 5 rules (no duplicates)
	rules2, err := uc.List(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, rules2, 5)
}

func TestUseCase_Update(t *testing.T) {
	repo := &mockRepo{}
	uc := notification.NewUseCase(repo)
	userID := uuid.NewString()

	// Seed defaults
	rules, err := uc.List(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, rules, 5)

	ruleToUpdate := rules[0]

	// Update enabled
	enabled := false
	updated, err := uc.Update(context.Background(), userID, ruleToUpdate.ID, notification.NotificationRuleUpdate{
		Enabled: &enabled,
	})
	require.NoError(t, err)
	assert.False(t, updated.Enabled)

	// Update channel
	channel := "email"
	updated, err = uc.Update(context.Background(), userID, ruleToUpdate.ID, notification.NotificationRuleUpdate{
		Channel: &channel,
	})
	require.NoError(t, err)
	assert.Equal(t, "email", updated.Channel)

	// Invalid channel
	invalidChannel := "invalid"
	_, err = uc.Update(context.Background(), userID, ruleToUpdate.ID, notification.NotificationRuleUpdate{
		Channel: &invalidChannel,
	})
	assert.Error(t, err)
	assert.Equal(t, "invalid channel", err.Error())

	// 404 for other user's rule
	otherUserID := uuid.NewString()
	_, err = uc.Update(context.Background(), otherUserID, ruleToUpdate.ID, notification.NotificationRuleUpdate{
		Enabled: &enabled,
	})
	assert.ErrorIs(t, err, notification.ErrNotFound)
}
