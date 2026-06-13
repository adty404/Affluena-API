package alert

import (
	"context"
	"testing"

	"affluena-api/internal/budget"
	"affluena-api/internal/page"
)

type mockRepo struct {
	email   string
	err     error
	catName string
}

func (m *mockRepo) GetUserEmail(ctx context.Context, userID string) (string, error) {
	return m.email, m.err
}

func (m *mockRepo) GetCategoryName(ctx context.Context, categoryID string) (string, error) {
	return m.catName, nil
}

type mockBudget struct {
	result page.Result[budget.BudgetSummary]
	err    error
}

func (m *mockBudget) List(ctx context.Context, userID string, monthValue string, pagination page.Params) (page.Result[budget.BudgetSummary], error) {
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
			uc.CheckBudgetAndAlert(context.Background(), "user-1", "cat-1")

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
