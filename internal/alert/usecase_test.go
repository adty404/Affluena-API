package alert

import (
	"context"
	"testing"
	"time"

	"affluena-api/internal/budget"
	"affluena-api/internal/page"
)

type mockRepo struct {
	email       string
	err         error
	catName     string
	alreadySent bool
}

func (m *mockRepo) GetUserEmail(ctx context.Context, userID string) (string, error) {
	return m.email, m.err
}

func (m *mockRepo) GetCategoryName(ctx context.Context, categoryID string) (string, error) {
	return m.catName, nil
}

func (m *mockRepo) HasAlertBeenSent(ctx context.Context, userID, categoryID, monthValue, alertType string) (bool, error) {
	return m.alreadySent, nil
}

func (m *mockRepo) MarkAlertSent(ctx context.Context, userID, categoryID, monthValue, alertType string) error {
	return nil
}

type mockBudget struct {
	result     page.Result[budget.BudgetSummary]
	err        error
	monthValue string
}

func (m *mockBudget) List(ctx context.Context, userID string, monthValue string, pagination page.Params) (page.Result[budget.BudgetSummary], error) {
	m.monthValue = monthValue
	return m.result, m.err
}

type mockMailer struct {
	sentCount int
	lastSubj  string
}

func (m *mockMailer) SendEmail(ctx context.Context, to []string, subject string, htmlBody string) error {
	m.sentCount++
	m.lastSubj = subject
	return nil
}

func TestCheckBudgetAndAlert(t *testing.T) {
	tests := []struct {
		name       string
		spent      int64
		limit      int64
		expectSent bool
		expectType string
	}{
		{"Below threshold (50%)", 50, 100, false, ""},
		{"Exactly 80%", 80, 100, true, "WARNING"},
		{"Over 80%", 85, 100, true, "WARNING"},
		{"Exactly 100%", 100, 100, true, "EXCEEDED"},
		{"Over 100%", 150, 100, true, "EXCEEDED"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := budget.BudgetSummary{
				Budget: budget.Budget{
					CategoryID: "cat-1",
					LimitMinor: tc.limit,
				},
				SpentMinor: tc.spent,
			}
			provider := &mockBudget{
				result: page.Result[budget.BudgetSummary]{Items: []budget.BudgetSummary{b}},
			}
			repo := &mockRepo{email: "test@example.com"}
			mailer := &mockMailer{}

			uc := NewUseCase(repo, provider, mailer)
			uc.CheckBudgetAndAlert(context.Background(), "user-1", "cat-1", time.Now().UTC())

			if tc.expectSent {
				if mailer.sentCount != 1 {
					t.Errorf("expected 1 email sent, got %d", mailer.sentCount)
				}
				// Verify subject format roughly
				if tc.expectType == "EXCEEDED" && mailer.lastSubj == "" {
					t.Errorf("expected subject for EXCEEDED")
				}
			} else {
				if mailer.sentCount != 0 {
					t.Errorf("expected 0 emails sent, got %d", mailer.sentCount)
				}
			}
		})
	}
}

func TestCheckBudgetAndAlertUsesTransactionMonth(t *testing.T) {
	provider := &mockBudget{
		result: page.Result[budget.BudgetSummary]{
			Items: []budget.BudgetSummary{
				{
					Budget: budget.Budget{
						CategoryID: "cat-1",
						LimitMinor: 100_000,
					},
					SpentMinor: 90_000,
				},
			},
		},
	}
	repo := &mockRepo{email: "test@example.com"}
	mailer := &mockMailer{}
	uc := NewUseCase(repo, provider, mailer)

	transactionAt := time.Date(2026, time.May, 20, 15, 30, 0, 0, time.UTC)
	uc.CheckBudgetAndAlert(context.Background(), "user-1", "cat-1", transactionAt)

	if provider.monthValue != "2026-05" {
		t.Fatalf("expected alert to check budget month 2026-05, got %q", provider.monthValue)
	}
	if mailer.sentCount != 1 {
		t.Fatalf("expected alert email to be sent for matching transaction month, got %d", mailer.sentCount)
	}
}

func TestCheckBudgetAndAlertDeduplication(t *testing.T) {
	b := budget.BudgetSummary{
		Budget: budget.Budget{
			CategoryID: "cat-1",
			LimitMinor: 100,
		},
		SpentMinor: 80,
	}
	provider := &mockBudget{
		result: page.Result[budget.BudgetSummary]{Items: []budget.BudgetSummary{b}},
	}
	repo := &mockRepo{email: "test@example.com", alreadySent: true}
	mailer := &mockMailer{}
	uc := NewUseCase(repo, provider, mailer)

	uc.CheckBudgetAndAlert(context.Background(), "user-1", "cat-1", time.Now().UTC())

	if mailer.sentCount != 0 {
		t.Errorf("expected 0 emails sent when alert already sent, got %d", mailer.sentCount)
	}
}
