package budget

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeBudgetRepository struct {
	createInput CreateBudgetInput
	created     Budget
	listed      []BudgetSummary
	got         Budget
	updated     Budget
	deletedID   string
	err         error
}

func (f *fakeBudgetRepository) Create(ctx context.Context, userID string, input CreateBudgetInput) (Budget, error) {
	f.createInput = input
	if f.err != nil {
		return Budget{}, f.err
	}
	return f.created, nil
}

func (f *fakeBudgetRepository) List(ctx context.Context, userID string, month time.Time) ([]BudgetSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.listed, nil
}

func (f *fakeBudgetRepository) Get(ctx context.Context, userID string, id string) (Budget, error) {
	if f.err != nil {
		return Budget{}, f.err
	}
	return f.got, nil
}

func (f *fakeBudgetRepository) Update(ctx context.Context, userID string, id string, input UpdateBudgetInput) (Budget, error) {
	if f.err != nil {
		return Budget{}, f.err
	}
	return f.updated, nil
}

func (f *fakeBudgetRepository) Delete(ctx context.Context, userID string, id string) error {
	f.deletedID = id
	return f.err
}

func TestBudgetUseCaseCreateParsesMonthAndDelegates(t *testing.T) {
	repo := &fakeBudgetRepository{created: Budget{ID: "budget-1"}}
	uc := NewUseCase(repo)

	created, err := uc.Create(context.Background(), "user-1", CreateBudgetInput{
		CategoryID: "category-1",
		Month:      "2026-06",
		LimitMinor: 200_000,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID != "budget-1" {
		t.Fatalf("expected created budget, got %+v", created)
	}
	wantMonth := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if !repo.createInput.MonthDate.Equal(wantMonth) {
		t.Fatalf("expected month %s, got %s", wantMonth, repo.createInput.MonthDate)
	}
}

func TestBudgetUseCaseRejectsInvalidLimit(t *testing.T) {
	uc := NewUseCase(&fakeBudgetRepository{})

	if _, err := uc.Create(context.Background(), "user-1", CreateBudgetInput{CategoryID: "category-1", Month: "2026-06"}); err == nil {
		t.Fatal("expected invalid limit error")
	}
	if _, err := uc.Update(context.Background(), "user-1", "budget-1", UpdateBudgetInput{CategoryID: "category-1", Month: "2026-06"}); err == nil {
		t.Fatal("expected invalid limit error")
	}
}

func TestBudgetUseCaseDelegatesReadAndDelete(t *testing.T) {
	repo := &fakeBudgetRepository{
		listed: []BudgetSummary{{Budget: Budget{ID: "budget-1"}}},
		got:    Budget{ID: "budget-1"},
	}
	uc := NewUseCase(repo)

	listed, err := uc.List(context.Background(), "user-1", "2026-06")
	if err != nil || len(listed) != 1 || listed[0].ID != "budget-1" {
		t.Fatalf("unexpected List result %+v err=%v", listed, err)
	}
	got, err := uc.Get(context.Background(), "user-1", "budget-1")
	if err != nil || got.ID != "budget-1" {
		t.Fatalf("unexpected Get result %+v err=%v", got, err)
	}
	if err := uc.Delete(context.Background(), "user-1", "budget-1"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if repo.deletedID != "budget-1" {
		t.Fatalf("expected delete id budget-1, got %q", repo.deletedID)
	}
}

func TestBudgetNotFoundRecognizesRepositoryNoRows(t *testing.T) {
	if !NotFound(ErrNotFound) {
		t.Fatal("expected ErrNotFound to be recognized")
	}
	if NotFound(errors.New("other")) {
		t.Fatal("did not expect unrelated error to be recognized")
	}
}
