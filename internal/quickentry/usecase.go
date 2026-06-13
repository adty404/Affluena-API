package quickentry

import (
	"context"
	"time"

	"affluena/internal/page"
	"affluena/internal/transaction"
)

type RepositoryPort interface {
	Create(ctx context.Context, userID string, template Template) (Template, error)
	List(ctx context.Context, userID string, pagination page.Params) (page.Result[Template], error)
	Get(ctx context.Context, userID string, id string) (Template, error)
	Update(ctx context.Context, userID string, id string, template Template) (Template, error)
	Delete(ctx context.Context, userID string, id string) error
}

type TransactionCreator interface {
	Create(ctx context.Context, userID string, input transaction.TransactionInput) (transaction.Transaction, error)
}

type UseCase struct {
	repo         RepositoryPort
	transactions TransactionCreator
}

func NewUseCase(repo RepositoryPort, transactions TransactionCreator) *UseCase {
	return &UseCase{repo: repo, transactions: transactions}
}

func (u *UseCase) Create(ctx context.Context, userID string, template Template) (Template, error) {
	if err := validateTemplate(template); err != nil {
		return Template{}, err
	}
	return u.repo.Create(ctx, userID, template)
}

func (u *UseCase) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Template], error) {
	return u.repo.List(ctx, userID, pagination)
}

func (u *UseCase) Get(ctx context.Context, userID string, id string) (Template, error) {
	return u.repo.Get(ctx, userID, id)
}

func (u *UseCase) Update(ctx context.Context, userID string, id string, template Template) (Template, error) {
	if err := validateTemplate(template); err != nil {
		return Template{}, err
	}
	return u.repo.Update(ctx, userID, id, template)
}

func (u *UseCase) Delete(ctx context.Context, userID string, id string) error {
	return u.repo.Delete(ctx, userID, id)
}

func (u *UseCase) Execute(ctx context.Context, userID string, id string, input ExecuteInput) (ExecuteResult, error) {
	template, err := u.repo.Get(ctx, userID, id)
	if err != nil {
		return ExecuteResult{}, err
	}
	if input.TransactionAt.IsZero() {
		input.TransactionAt = time.Now().UTC()
	}
	note := template.Note
	if input.Note != "" {
		note = input.Note
	}
	created, err := u.transactions.Create(ctx, userID, transaction.TransactionInput{
		Type:           transaction.TransactionType(template.Type),
		WalletID:       template.WalletID,
		ToWalletID:     template.ToWalletID,
		CategoryID:     template.CategoryID,
		AmountMinor:    template.AmountMinor,
		TransactionUTC: input.TransactionAt.UTC(),
		Note:           note,
	})
	if err != nil {
		return ExecuteResult{}, err
	}
	return ExecuteResult{Transaction: created}, nil
}

func validateTemplate(template Template) error {
	_, err := transaction.BalanceDeltas(transaction.TransactionInput{
		Type:           transaction.TransactionType(template.Type),
		WalletID:       template.WalletID,
		ToWalletID:     template.ToWalletID,
		CategoryID:     template.CategoryID,
		AmountMinor:    template.AmountMinor,
		TransactionUTC: time.Now().UTC(),
		Note:           template.Note,
	})
	return err
}
