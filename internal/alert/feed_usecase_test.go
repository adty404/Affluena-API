package alert

import (
	"context"
	"testing"
	"time"

	"affluena-api/internal/activity"
	"affluena-api/internal/budget"
	"affluena-api/internal/debt"
	"affluena-api/internal/page"

	"github.com/stretchr/testify/assert"
)

type mockFeedRepo struct {
	debts      []debt.Debt
	activities []activity.Activity
}

func (m *mockFeedRepo) ListOverdueDebts(ctx context.Context, userID string) ([]debt.Debt, error) {
	return m.debts, nil
}

func (m *mockFeedRepo) ListRecentRecurringActivities(ctx context.Context, userID string, since time.Time) ([]activity.Activity, error) {
	return m.activities, nil
}

type mockBudgetProvider struct {
	budgets []budget.BudgetSummary
}

func (m *mockBudgetProvider) List(ctx context.Context, userID string, monthValue string, pagination page.Params) (page.Result[budget.BudgetSummary], error) {
	return page.Result[budget.BudgetSummary]{
		Items: m.budgets,
		Pagination: page.Metadata{
			Total: len(m.budgets),
		},
	}, nil
}

func TestFeedUseCase_List(t *testing.T) {
	now := time.Now().UTC()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	dueDate := yesterday
	entityID := "rec-1"

	repo := &mockFeedRepo{
		debts: []debt.Debt{
			{
				ID:               "debt-1",
				CounterpartyName: "John Doe",
				DueDate:          &dueDate,
			},
		},
		activities: []activity.Activity{
			{
				ID:          "act-1",
				ActionType:  "RUN_FAILED",
				Description: "Failed to run",
				CreatedAt:   twoDaysAgo,
				EntityID:    &entityID,
			},
		},
	}

	budgetUC := &mockBudgetProvider{
		budgets: []budget.BudgetSummary{
			{
				Budget: budget.Budget{
					ID:         "bud-1",
					LimitMinor: 1000,
				},
				UsagePercent: 85.0,
				SpentMinor:   850,
			},
			{
				Budget: budget.Budget{
					ID:         "bud-2",
					LimitMinor: 1000,
				},
				UsagePercent: 105.0,
				SpentMinor:   1050,
			},
			{
				Budget: budget.Budget{
					ID:         "bud-3",
					LimitMinor: 1000,
				},
				UsagePercent: 50.0,
				SpentMinor:   500,
			},
		},
	}

	uc := NewFeedUseCase(repo, budgetUC)

	alerts, err := uc.List(context.Background(), "user-1", "2023-10")
	assert.NoError(t, err)
	assert.Len(t, alerts, 4) // 2 budgets, 1 debt, 1 activity

	// Check sorting (newest first)
	// Budget alerts use time.Now().UTC() so they should be first
	assert.Equal(t, "budget", alerts[0].Type)
	assert.Equal(t, "budget", alerts[1].Type)

	// Debt alert uses DueDate (yesterday)
	assert.Equal(t, "debt", alerts[2].Type)
	assert.Equal(t, "debt-debt-1", alerts[2].ID)
	assert.Equal(t, "John Doe jatuh tempo", alerts[2].Title)
	assert.Equal(t, "warning", alerts[2].Severity)

	// Activity alert uses CreatedAt (twoDaysAgo)
	assert.Equal(t, "recurring", alerts[3].Type)
	assert.Equal(t, "recurring-act-1", alerts[3].ID)
	assert.Equal(t, "Aturan berulang gagal dijalankan", alerts[3].Title)
	assert.Equal(t, "warning", alerts[3].Severity)
	assert.Equal(t, "/recurring/rec-1/history", alerts[3].ActionPath)
}

func TestFeedUseCase_Get(t *testing.T) {
	dueDate := time.Now().UTC().Add(-24 * time.Hour)
	repo := &mockFeedRepo{
		debts: []debt.Debt{
			{
				ID:               "debt-1",
				CounterpartyName: "John Doe",
				DueDate:          &dueDate,
			},
		},
	}
	budgetUC := &mockBudgetProvider{
		budgets: []budget.BudgetSummary{
			{
				Budget: budget.Budget{
					ID: "bud-1",
				},
				UsagePercent: 85.0,
			},
		},
	}

	uc := NewFeedUseCase(repo, budgetUC)

	// Test Get Debt
	alert, err := uc.Get(context.Background(), "user-1", "debt-debt-1")
	assert.NoError(t, err)
	assert.Equal(t, "debt-debt-1", alert.ID)

	// Test Get Budget
	monthValue := time.Now().UTC().Format("2006-01")
	budgetID := "budget-bud-1-" + monthValue
	alert, err = uc.Get(context.Background(), "user-1", budgetID)
	assert.NoError(t, err)
	assert.Equal(t, budgetID, alert.ID)

	// Test Not Found
	_, err = uc.Get(context.Background(), "user-1", "not-found")
	assert.ErrorIs(t, err, ErrAlertNotFound)
}
