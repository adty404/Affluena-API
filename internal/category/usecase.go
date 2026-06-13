package category

import (
	"context"

	"affluena-api/internal/activity"
	"affluena-api/internal/page"
)

type RepositoryPort interface {
	Create(ctx context.Context, userID string, input CreateCategoryInput) (Category, error)
	List(ctx context.Context, userID string, categoryType string, pagination page.Params) (page.Result[Category], error)
	Get(ctx context.Context, userID string, id string) (Category, error)
	Update(ctx context.Context, userID string, id string, input UpdateCategoryInput) (Category, error)
	Delete(ctx context.Context, userID string, id string) error
	CheckParent(ctx context.Context, userID string, categoryType string, parentID *string) error
	CheckDepth(ctx context.Context, parentID *string) error
	CheckCycle(ctx context.Context, categoryID string, newParentID *string) error
}

type UseCase struct {
	repo       RepositoryPort
	activityUC activity.UseCase
}

func NewUseCase(repo RepositoryPort, activityUC activity.UseCase) *UseCase {
	return &UseCase{repo: repo, activityUC: activityUC}
}

func (u *UseCase) Create(ctx context.Context, userID string, input CreateCategoryInput) (Category, error) {
	if !IsValidType(input.Type) {
		return Category{}, ErrInvalidCategoryType
	}
	if err := u.repo.CheckParent(ctx, userID, input.Type, input.ParentID); err != nil {
		return Category{}, err
	}
	if err := u.repo.CheckDepth(ctx, input.ParentID); err != nil {
		return Category{}, err
	}
	c, err := u.repo.Create(ctx, userID, input)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "CREATE", "CATEGORY", &c.ID, "Membuat kategori baru: "+input.Name)
	}
	return c, err
}

func (u *UseCase) List(ctx context.Context, userID string, categoryType string, pagination page.Params) (page.Result[Category], error) {
	if categoryType != "" && !IsValidType(categoryType) {
		return page.Result[Category]{}, ErrInvalidCategoryType
	}
	return u.repo.List(ctx, userID, categoryType, pagination)
}

func (u *UseCase) Get(ctx context.Context, userID string, id string) (Category, error) {
	return u.repo.Get(ctx, userID, id)
}

func (u *UseCase) Update(ctx context.Context, userID string, id string, input UpdateCategoryInput) (Category, error) {
	if !IsValidType(input.Type) {
		return Category{}, ErrInvalidCategoryType
	}
	if err := u.repo.CheckCycle(ctx, id, input.ParentID); err != nil {
		return Category{}, err
	}
	if err := u.repo.CheckParent(ctx, userID, input.Type, input.ParentID); err != nil {
		return Category{}, err
	}
	if err := u.repo.CheckDepth(ctx, input.ParentID); err != nil {
		return Category{}, err
	}
	c, err := u.repo.Update(ctx, userID, id, input)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "UPDATE", "CATEGORY", &id, "Mengubah kategori: "+input.Name)
	}
	return c, err
}

func (u *UseCase) Delete(ctx context.Context, userID string, id string) error {
	err := u.repo.Delete(ctx, userID, id)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "DELETE", "CATEGORY", &id, "Menghapus kategori")
	}
	return err
}
