package transaction

import "context"

import "affluena-api/internal/page"

type RepositoryPort interface {
	Create(ctx context.Context, userID string, input TransactionInput) (Transaction, error)
	List(ctx context.Context, userID string, filter TransactionFilter, pagination page.Params) (page.Result[Transaction], error)
	Get(ctx context.Context, userID string, id string) (Transaction, error)
	Update(ctx context.Context, userID string, id string, input TransactionInput) (Transaction, error)
	Delete(ctx context.Context, userID string, id string) error
}

type UseCase struct {
	repo RepositoryPort
}

func NewUseCase(repo RepositoryPort) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) Create(ctx context.Context, userID string, input TransactionInput) (Transaction, error) {
	if _, err := BalanceDeltas(input); err != nil {
		return Transaction{}, err
	}
	return u.repo.Create(ctx, userID, input)
}

func (u *UseCase) List(ctx context.Context, userID string, filter TransactionFilter, pagination page.Params) (page.Result[Transaction], error) {
	return u.repo.List(ctx, userID, filter, pagination)
}

func (u *UseCase) Get(ctx context.Context, userID string, id string) (Transaction, error) {
	return u.repo.Get(ctx, userID, id)
}

func (u *UseCase) Update(ctx context.Context, userID string, id string, input TransactionInput) (Transaction, error) {
	if _, err := BalanceDeltas(input); err != nil {
		return Transaction{}, err
	}
	return u.repo.Update(ctx, userID, id, input)
}

func (u *UseCase) Delete(ctx context.Context, userID string, id string) error {
	return u.repo.Delete(ctx, userID, id)
}
