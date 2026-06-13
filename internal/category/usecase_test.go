package category

import (
	"context"
	"errors"
	"testing"
	"time"

	"affluena/internal/page"
)

type fakeCategoryRepository struct {
	createInput CreateCategoryInput
	created     Category
	listType    string
	listPage    page.Params
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

func (f *fakeCategoryRepository) List(ctx context.Context, userID string, categoryType string, pagination page.Params) (page.Result[Category], error) {
	f.listType = categoryType
	f.listPage = pagination
	if f.err != nil {
		return page.Result[Category]{}, f.err
	}
	return page.NewResult(f.listed, pagination, len(f.listed)), nil
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

func TestCategoryUseCaseAcceptsKnownTypes(t *testing.T) {
	tests := []string{"income", "expense"}

	for _, categoryType := range tests {
		t.Run(categoryType, func(t *testing.T) {
			repo := &fakeCategoryRepository{created: Category{ID: "category-1", Type: categoryType}}
			uc := NewUseCase(repo)

			created, err := uc.Create(context.Background(), "user-1", CreateCategoryInput{Name: "Category", Type: categoryType})
			if err != nil {
				t.Fatalf("Create returned error: %v", err)
			}
			if created.Type != categoryType {
				t.Fatalf("expected category type %s, got %+v", categoryType, created)
			}
		})
	}
}

func TestCategoryUseCasePropagatesRepositoryErrors(t *testing.T) {
	repoErr := errors.New("repo failed")
	uc := NewUseCase(&fakeCategoryRepository{err: repoErr})

	if _, err := uc.Create(context.Background(), "user-1", CreateCategoryInput{Name: "Salary", Type: "income"}); !errors.Is(err, repoErr) {
		t.Fatalf("expected create repo error, got %v", err)
	}
	if _, err := uc.List(context.Background(), "user-1", "", page.Params{Limit: 100}); !errors.Is(err, repoErr) {
		t.Fatalf("expected list repo error, got %v", err)
	}
	if _, err := uc.Update(context.Background(), "user-1", "category-1", UpdateCategoryInput{Name: "Salary", Type: "income"}); !errors.Is(err, repoErr) {
		t.Fatalf("expected update repo error, got %v", err)
	}
	if err := uc.Delete(context.Background(), "user-1", "category-1"); !errors.Is(err, repoErr) {
		t.Fatalf("expected delete repo error, got %v", err)
	}
}

func TestCategoryUseCaseDelegatesReadAndDelete(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeCategoryRepository{
		listed: []Category{{ID: "category-1", CreatedAt: now}},
		got:    Category{ID: "category-1", CreatedAt: now},
	}
	uc := NewUseCase(repo)

	listed, err := uc.List(context.Background(), "user-1", "", page.Params{Limit: 10, Sort: "type_name_asc"})
	if err != nil || len(listed.Items) != 1 || listed.Items[0].ID != "category-1" {
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

func TestCategoryUseCaseListAcceptsOptionalTypeFilter(t *testing.T) {
	repo := &fakeCategoryRepository{listed: []Category{{ID: "category-1", Type: "income"}}}
	uc := NewUseCase(repo)

	listed, err := uc.List(context.Background(), "user-1", "income", page.Params{Limit: 10, Sort: "type_name_asc"})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(listed.Items) != 1 || listed.Items[0].Type != "income" {
		t.Fatalf("unexpected List result %+v", listed)
	}
	if repo.listType != "income" {
		t.Fatalf("expected repository to receive income filter, got %q", repo.listType)
	}
	if repo.listPage.Limit != 10 || repo.listPage.Sort != "type_name_asc" {
		t.Fatalf("expected repository to receive pagination, got %+v", repo.listPage)
	}
}

func TestCategoryUseCaseListRejectsInvalidTypeFilter(t *testing.T) {
	repo := &fakeCategoryRepository{}
	uc := NewUseCase(repo)

	if _, err := uc.List(context.Background(), "user-1", "transfer", page.Params{Limit: 100}); err == nil {
		t.Fatal("expected invalid category type error")
	}
	if repo.listType != "" {
		t.Fatalf("expected invalid filter to skip repository, got %q", repo.listType)
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
