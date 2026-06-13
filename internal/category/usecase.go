package category

import "context"

type RepositoryPort interface {
	Create(ctx context.Context, userID string, input CreateCategoryInput) (Category, error)
	List(ctx context.Context, userID string, categoryType string) ([]Category, error)
	Get(ctx context.Context, userID string, id string) (Category, error)
	Update(ctx context.Context, userID string, id string, input UpdateCategoryInput) (Category, error)
	Delete(ctx context.Context, userID string, id string) error
}

type UseCase struct {
	repo RepositoryPort
}

func NewUseCase(repo RepositoryPort) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) Create(ctx context.Context, userID string, input CreateCategoryInput) (Category, error) {
	if !IsValidType(input.Type) {
		return Category{}, ErrInvalidCategoryType
	}
	return u.repo.Create(ctx, userID, input)
}

func (u *UseCase) List(ctx context.Context, userID string, categoryType string) ([]Category, error) {
	if categoryType != "" && !IsValidType(categoryType) {
		return nil, ErrInvalidCategoryType
	}
	return u.repo.List(ctx, userID, categoryType)
}

func (u *UseCase) Get(ctx context.Context, userID string, id string) (Category, error) {
	return u.repo.Get(ctx, userID, id)
}

func (u *UseCase) Update(ctx context.Context, userID string, id string, input UpdateCategoryInput) (Category, error) {
	if !IsValidType(input.Type) {
		return Category{}, ErrInvalidCategoryType
	}
	return u.repo.Update(ctx, userID, id, input)
}

func (u *UseCase) Delete(ctx context.Context, userID string, id string) error {
	return u.repo.Delete(ctx, userID, id)
}
