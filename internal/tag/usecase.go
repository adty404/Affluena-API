package tag

import (
	"context"

	"affluena-api/internal/activity"
	"affluena-api/internal/page"
)

type RepositoryPort interface {
	Create(ctx context.Context, userID string, input CreateTagInput) (Tag, error)
	List(ctx context.Context, userID string, pagination page.Params) (page.Result[Tag], error)
	Get(ctx context.Context, userID string, id string) (Tag, error)
	Update(ctx context.Context, userID string, id string, input UpdateTagInput) (Tag, error)
	Delete(ctx context.Context, userID string, id string) error
}

type UseCase struct {
	repo       RepositoryPort
	activityUC activity.UseCase
}

func NewUseCase(repo RepositoryPort, activityUC activity.UseCase) *UseCase {
	return &UseCase{repo: repo, activityUC: activityUC}
}

func (u *UseCase) Create(ctx context.Context, userID string, input CreateTagInput) (Tag, error) {
	t, err := u.repo.Create(ctx, userID, input)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "CREATE", "TAG", &t.ID, "Membuat label baru: "+input.Name)
	}
	return t, err
}

func (u *UseCase) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Tag], error) {
	return u.repo.List(ctx, userID, pagination)
}

func (u *UseCase) Get(ctx context.Context, userID string, id string) (Tag, error) {
	return u.repo.Get(ctx, userID, id)
}

func (u *UseCase) Update(ctx context.Context, userID string, id string, input UpdateTagInput) (Tag, error) {
	t, err := u.repo.Update(ctx, userID, id, input)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "UPDATE", "TAG", &id, "Mengubah label: "+input.Name)
	}
	return t, err
}

func (u *UseCase) Delete(ctx context.Context, userID string, id string) error {
	err := u.repo.Delete(ctx, userID, id)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "DELETE", "TAG", &id, "Menghapus label")
	}
	return err
}
