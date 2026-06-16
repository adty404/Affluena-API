package alert

import (
	"context"
	"testing"
	"time"

	"affluena-api/internal/budget"
	"affluena-api/internal/page"
)

type mockRepo struct {
	email      string
	err        error
	catName    string
	shouldSend bool
}

func (m *mockRepo) GetUserEmail(ctx context.Context, userID string) (string, error) {
	return m.email, m.err
}

func (m *mockRepo) GetCategoryName(ctx context.Context, categoryID string) (string, error) {
	return m.catName, nil
}

func (m *mockRepo) TryInsertSentAlert(ctx context.Context, userID, categoryID, monthValue, alertType string) (bool, error) {
	return m.shouldSend, m.err
}

func (m *mockRepo) DeleteSentAlert(ctx context.Context, userID, categoryID, monthValue, alertType string) error {
	return nil
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
	err       error
}

func (m *mockMailer) SendEmail(ctx context.Context, to []string, subject string, htmlBody string) error {
	if m.err != nil {
		return m.err
	}
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
			repo := &mockRepo{email: "test@example.com", shouldSend: true}
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
	repo := &mockRepo{email: "test@example.com", shouldSend: true}
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

type fakeAtomicRepo struct {
	email    string
	catName  string
	inserted bool
	deleted  bool
}

func (m *fakeAtomicRepo) GetUserEmail(ctx context.Context, userID string) (string, error) {
	return m.email, nil
}
func (m *fakeAtomicRepo) GetCategoryName(ctx context.Context, categoryID string) (string, error) {
	return m.catName, nil
}
func (m *fakeAtomicRepo) TryInsertSentAlert(ctx context.Context, userID, categoryID, monthValue, alertType string) (bool, error) {
	if m.inserted {
		return false, nil // Already inserted, atomic dedup prevents duplicate
	}
	m.inserted = true
	return true, nil
}
func (m *fakeAtomicRepo) DeleteSentAlert(ctx context.Context, userID, categoryID, monthValue, alertType string) error {
	m.deleted = true
	m.inserted = false
	return nil
}
func (m *fakeAtomicRepo) MarkAlertSent(ctx context.Context, userID, categoryID, monthValue, alertType string) error {
	return nil
}

func TestCheckBudgetAndAlertAtomicDeduplication(t *testing.T) {
	b := budget.BudgetSummary{
		Budget: budget.Budget{
			CategoryID: "cat-1",
			LimitMinor: 100,
		},
		SpentMinor: 80, // 80% used -> triggers 80% alert
	}
	provider := &mockBudget{
		result: page.Result[budget.BudgetSummary]{Items: []budget.BudgetSummary{b}},
	}

	repo := &fakeAtomicRepo{email: "test@example.com", catName: "Food"}
	mailer := &mockMailer{}
	uc := NewUseCase(repo, provider, mailer)

	// First call should succeed and send email
	uc.CheckBudgetAndAlert(context.Background(), "user-1", "cat-1", time.Now().UTC())

	if mailer.sentCount != 1 {
		t.Errorf("expected 1 email sent, got %d", mailer.sentCount)
	}
	if !repo.inserted {
		t.Errorf("expected alert to be inserted atomically")
	}

	// Second call with same state should be deduped by TryInsertSentAlert returning false
	uc.CheckBudgetAndAlert(context.Background(), "user-1", "cat-1", time.Now().UTC())

	if mailer.sentCount != 1 {
		t.Errorf("expected email count to remain 1 due to atomic dedup, got %d", mailer.sentCount)
	}

	// Simulate failure policy: if mailer fails, it deletes the alert
	mailerWithError := &mockMailer{err: context.DeadlineExceeded}
	repoForFailure := &fakeAtomicRepo{email: "test@example.com", catName: "Food"}
	ucWithErr := NewUseCase(repoForFailure, provider, mailerWithError)

	ucWithErr.CheckBudgetAndAlert(context.Background(), "user-1", "cat-1", time.Now().UTC())

	if !repoForFailure.deleted {
		t.Errorf("expected alert to be deleted after email failure for retry")
	}
}
