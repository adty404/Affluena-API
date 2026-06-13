package recurring

import (
	"context"
	"time"

	"affluena-api/internal/activity"
	"affluena-api/internal/page"
)

type RepositoryPort interface {
	Create(ctx context.Context, userID string, input RuleInput) (Rule, error)
	List(ctx context.Context, userID string, pagination page.Params) (page.Result[Rule], error)
	Get(ctx context.Context, userID string, id string) (Rule, error)
	Update(ctx context.Context, userID string, id string, input RuleInput) (Rule, error)
	Delete(ctx context.Context, userID string, id string) error
	RunManual(ctx context.Context, userID string, id string, now time.Time) (Run, error)
	RunDue(ctx context.Context, now time.Time, limit int) ([]Run, error)
}

type UseCase struct {
	repo       RepositoryPort
	activityUC activity.UseCase
}

func NewUseCase(repo RepositoryPort, activityUC activity.UseCase) *UseCase {
	return &UseCase{repo: repo, activityUC: activityUC}
}

func (u *UseCase) Create(ctx context.Context, userID string, input RuleInput) (Rule, error) {
	r, err := u.repo.Create(ctx, userID, input)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "CREATE", "RECURRING_RULE", &r.ID, "Membuat aturan transaksi berulang")
	}
	return r, err
}

func (u *UseCase) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Rule], error) {
	return u.repo.List(ctx, userID, pagination)
}

func (u *UseCase) Get(ctx context.Context, userID string, id string) (Rule, error) {
	return u.repo.Get(ctx, userID, id)
}

func (u *UseCase) Update(ctx context.Context, userID string, id string, input RuleInput) (Rule, error) {
	r, err := u.repo.Update(ctx, userID, id, input)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "UPDATE", "RECURRING_RULE", &id, "Mengubah aturan transaksi berulang")
	}
	return r, err
}

func (u *UseCase) Delete(ctx context.Context, userID string, id string) error {
	err := u.repo.Delete(ctx, userID, id)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "DELETE", "RECURRING_RULE", &id, "Menghapus aturan transaksi berulang")
	}
	return err
}

func (u *UseCase) RunManual(ctx context.Context, userID string, id string, now time.Time) (Run, error) {
	run, err := u.repo.RunManual(ctx, userID, id, now)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "EXECUTE", "RECURRING_RULE", &id, "Mengeksekusi transaksi berulang secara manual")
	}
	return run, err
}

func (u *UseCase) RunDue(ctx context.Context, now time.Time, limit int) ([]Run, error) {
	return u.repo.RunDue(ctx, now, limit)
}
