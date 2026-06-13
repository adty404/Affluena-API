package recurring

import (
	"context"
	"testing"
	"time"

	"affluena-api/internal/page"
)

type fakeRecurringRepository struct {
	createInput RuleInput
	updateInput RuleInput
	manualNow   time.Time
	dueNow      time.Time
	dueLimit    int
	created     Rule
	listPage    page.Params
	listed      []Rule
	got         Rule
	updated     Rule
	manual      Run
	due         []Run
	err         error
}

func (f *fakeRecurringRepository) Create(ctx context.Context, userID string, input RuleInput) (Rule, error) {
	f.createInput = input
	return f.created, f.err
}

func (f *fakeRecurringRepository) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Rule], error) {
	f.listPage = pagination
	if f.err != nil {
		return page.Result[Rule]{}, f.err
	}
	return page.NewResult(f.listed, pagination, len(f.listed)), nil
}

func (f *fakeRecurringRepository) Get(ctx context.Context, userID string, id string) (Rule, error) {
	return f.got, f.err
}

func (f *fakeRecurringRepository) Update(ctx context.Context, userID string, id string, input RuleInput) (Rule, error) {
	f.updateInput = input
	return f.updated, f.err
}

func (f *fakeRecurringRepository) Delete(ctx context.Context, userID string, id string) error {
	return f.err
}

func (f *fakeRecurringRepository) RunManual(ctx context.Context, userID string, id string, now time.Time) (Run, error) {
	f.manualNow = now
	return f.manual, f.err
}

func (f *fakeRecurringRepository) RunDue(ctx context.Context, now time.Time, limit int) ([]Run, error) {
	f.dueNow = now
	f.dueLimit = limit
	return f.due, f.err
}

func TestRecurringUseCaseDelegatesRuleActions(t *testing.T) {
	repo := &fakeRecurringRepository{
		created: Rule{ID: "rule-1"},
		listed:  []Rule{{ID: "rule-1"}},
		got:     Rule{ID: "rule-1"},
		updated: Rule{ID: "rule-1", Status: StatusActive},
		manual:  Run{ID: "run-1"},
		due:     []Run{{ID: "run-2"}},
	}
	uc := NewUseCase(repo, nil)

	if got, err := uc.Create(context.Background(), "user-1", RuleInput{}); err != nil || got.ID != "rule-1" {
		t.Fatalf("unexpected Create result %+v err=%v", got, err)
	}
	if got, err := uc.List(context.Background(), "user-1", page.Params{Limit: 10, Sort: "next_run_at_asc"}); err != nil || len(got.Items) != 1 {
		t.Fatalf("unexpected List result %+v err=%v", got, err)
	}
	if got, err := uc.Get(context.Background(), "user-1", "rule-1"); err != nil || got.ID != "rule-1" {
		t.Fatalf("unexpected Get result %+v err=%v", got, err)
	}
	if got, err := uc.Update(context.Background(), "user-1", "rule-1", RuleInput{}); err != nil || got.ID != "rule-1" {
		t.Fatalf("unexpected Update result %+v err=%v", got, err)
	}
	if err := uc.Delete(context.Background(), "user-1", "rule-1"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if got, err := uc.RunManual(context.Background(), "user-1", "rule-1", time.Now().UTC()); err != nil || got.ID != "run-1" {
		t.Fatalf("unexpected RunManual result %+v err=%v", got, err)
	}
	if got, err := uc.RunDue(context.Background(), time.Now().UTC(), 20); err != nil || len(got) != 1 {
		t.Fatalf("unexpected RunDue result %+v err=%v", got, err)
	}
}

func TestRecurringUseCasePassesInputsThrough(t *testing.T) {
	repo := &fakeRecurringRepository{created: Rule{ID: "rule-1"}, updated: Rule{ID: "rule-1"}, manual: Run{ID: "run-1"}, due: []Run{{ID: "run-2"}}}
	uc := NewUseCase(repo, nil)

	input := RuleInput{Name: "Rent", Frequency: FrequencyMonthly, IntervalCount: 1}
	if _, err := uc.Create(context.Background(), "user-1", input); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if repo.createInput.Name != "Rent" {
		t.Fatalf("expected create input to be captured, got %+v", repo.createInput)
	}

	input.Name = "Rent updated"
	if _, err := uc.Update(context.Background(), "user-1", "rule-1", input); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if repo.updateInput.Name != "Rent updated" {
		t.Fatalf("expected update input to be captured, got %+v", repo.updateInput)
	}

	now := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	if _, err := uc.RunManual(context.Background(), "user-1", "rule-1", now); err != nil {
		t.Fatalf("RunManual returned error: %v", err)
	}
	if !repo.manualNow.Equal(now) {
		t.Fatalf("expected manual now %s, got %s", now, repo.manualNow)
	}

	if _, err := uc.RunDue(context.Background(), now, 7); err != nil {
		t.Fatalf("RunDue returned error: %v", err)
	}
	if !repo.dueNow.Equal(now) || repo.dueLimit != 7 {
		t.Fatalf("expected due input %s/7, got %s/%d", now, repo.dueNow, repo.dueLimit)
	}
}

func TestRecurringUseCasePropagatesRepositoryErrors(t *testing.T) {
	repoErr := assertRecurringError{}
	uc := NewUseCase(&fakeRecurringRepository{err: repoErr}, nil)

	if _, err := uc.Create(context.Background(), "user-1", RuleInput{}); err != repoErr {
		t.Fatalf("expected create error, got %v", err)
	}
	if _, err := uc.List(context.Background(), "user-1", page.Params{Limit: 100}); err != repoErr {
		t.Fatalf("expected list error, got %v", err)
	}
	if err := uc.Delete(context.Background(), "user-1", "rule-1"); err != repoErr {
		t.Fatalf("expected delete error, got %v", err)
	}
	if _, err := uc.RunDue(context.Background(), time.Now().UTC(), 1); err != repoErr {
		t.Fatalf("expected run due error, got %v", err)
	}
}

type assertRecurringError struct{}

func (assertRecurringError) Error() string { return "recurring repository failed" }
