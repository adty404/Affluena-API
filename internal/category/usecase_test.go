package category

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeCategoryRepository struct {
	createInput CreateCategoryInput
	created     Category
	listed      []Category
	got         Category
	updated     Category
	deletedID   string
	err         error
}

func (f *fakeCategoryRepository) Create(ctx context.Context, userID string, input CreateCategoryInput) (Category, error) {
	f.createInput = input
	if f.err != nil {
		return Category{}, f.err
	}
	return f.created, nil
}

func (f *fakeCategoryRepository) List(ctx context.Context, userID string) ([]Category, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.listed, nil
}

func (f *fakeCategoryRepository) Get(ctx context.Context, userID string, id string) (Category, error) {
	if f.err != nil {
		return Category{}, f.err
	}
	return f.got, nil
}

func (f *fakeCategoryRepository) Update(ctx context.Context, userID string, id string, input UpdateCategoryInput) (Category, error) {
	if f.err != nil {
		return Category{}, f.err
	}
	return f.updated, nil
}

func (f *fakeCategoryRepository) Delete(ctx context.Context, userID string, id string) error {
	f.deletedID = id
	return f.err
}

func TestCategoryUseCaseCreateDelegatesValidCategory(t *testing.T) {
	repo := &fakeCategoryRepository{created: Category{ID: "category-1"}}
	uc := NewUseCase(repo)

	created, err := uc.Create(context.Background(), "user-1", CreateCategoryInput{Name: "Salary", Type: "income"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID != "category-1" {
		t.Fatalf("expected created category, got %+v", created)
	}
	if repo.createInput.Type != "income" || repo.createInput.Name != "Salary" {
		t.Fatalf("unexpected create input %+v", repo.createInput)
	}
}

func TestCategoryUseCaseRejectsInvalidType(t *testing.T) {
	uc := NewUseCase(&fakeCategoryRepository{})

	if _, err := uc.Create(context.Background(), "user-1", CreateCategoryInput{Name: "Bad", Type: "transfer"}); err == nil {
		t.Fatal("expected invalid category type error")
	}
	if _, err := uc.Update(context.Background(), "user-1", "category-1", UpdateCategoryInput{Name: "Bad", Type: "transfer"}); err == nil {
		t.Fatal("expected invalid category type error")
	}
}

func TestCategoryUseCaseDelegatesReadAndDelete(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeCategoryRepository{
		listed: []Category{{ID: "category-1", CreatedAt: now}},
		got:    Category{ID: "category-1", CreatedAt: now},
	}
	uc := NewUseCase(repo)

	listed, err := uc.List(context.Background(), "user-1")
	if err != nil || len(listed) != 1 || listed[0].ID != "category-1" {
		t.Fatalf("unexpected List result %+v err=%v", listed, err)
	}
	got, err := uc.Get(context.Background(), "user-1", "category-1")
	if err != nil || got.ID != "category-1" {
		t.Fatalf("unexpected Get result %+v err=%v", got, err)
	}
	if err := uc.Delete(context.Background(), "user-1", "category-1"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if repo.deletedID != "category-1" {
		t.Fatalf("expected delete id category-1, got %q", repo.deletedID)
	}
}

func TestCategoryNotFoundRecognizesRepositoryNoRows(t *testing.T) {
	if !NotFound(ErrNotFound) {
		t.Fatal("expected ErrNotFound to be recognized")
	}
	if NotFound(errors.New("other")) {
		t.Fatal("did not expect unrelated error to be recognized")
	}
}
