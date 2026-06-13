package budget

import (
	"context"
	"errors"
	"testing"
	"time"

	"affluena-api/internal/page"
)

type fakeBudgetRepository struct {
	createInput CreateBudgetInput
	created     Budget
	listPage    page.Params
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

func (f *fakeBudgetRepository) List(ctx context.Context, userID string, month time.Time, pagination page.Params) (page.Result[BudgetSummary], error) {
	f.listPage = pagination
	if f.err != nil {
		return page.Result[BudgetSummary]{}, f.err
	}
	return page.NewResult(f.listed, pagination, len(f.listed)), nil
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
	uc := NewUseCase(repo, nil)

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
	uc := NewUseCase(&fakeBudgetRepository{}, nil)

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
	uc := NewUseCase(repo, nil)

	listed, err := uc.List(context.Background(), "user-1", "2026-06", page.Params{Limit: 10, Sort: "created_at_desc"})
	if err != nil || len(listed.Items) != 1 || listed.Items[0].ID != "budget-1" {
		t.Fatalf("unexpected List result %+v err=%v", listed, err)
	}
	if repo.listPage.Limit != 10 || repo.listPage.Sort != "created_at_desc" {
		t.Fatalf("expected repository to receive pagination, got %+v", repo.listPage)
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
