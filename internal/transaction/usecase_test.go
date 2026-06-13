package transaction

import (
	"context"
	"errors"
	"testing"
	"time"

	"affluena-api/internal/page"
)

type fakeTransactionRepository struct {
	createInput TransactionInput
	updateInput TransactionInput
	listFilter  TransactionFilter
	listPage    page.Params
	created     Transaction
	listed      []Transaction
	got         Transaction
	updated     Transaction
	deletedID   string
	err         error
}

func (f *fakeTransactionRepository) Create(ctx context.Context, userID string, input TransactionInput) (Transaction, error) {
	f.createInput = input
	if f.err != nil {
		return Transaction{}, f.err
	}
	return f.created, nil
}

func (f *fakeTransactionRepository) List(ctx context.Context, userID string, filter TransactionFilter, pagination page.Params) (page.Result[Transaction], error) {
	f.listFilter = filter
	f.listPage = pagination
	if f.err != nil {
		return page.Result[Transaction]{}, f.err
	}
	return page.NewResult(f.listed, pagination, len(f.listed)), nil
}

func (f *fakeTransactionRepository) Get(ctx context.Context, userID string, id string) (Transaction, error) {
	if f.err != nil {
		return Transaction{}, f.err
	}
	return f.got, nil
}

func (f *fakeTransactionRepository) Update(ctx context.Context, userID string, id string, input TransactionInput) (Transaction, error) {
	f.updateInput = input
	if f.err != nil {
		return Transaction{}, f.err
	}
	return f.updated, nil
}

func (f *fakeTransactionRepository) Delete(ctx context.Context, userID string, id string) error {
	f.deletedID = id
	return f.err
}

func TestTransactionUseCaseCreateValidatesAndDelegates(t *testing.T) {
	repo := &fakeTransactionRepository{created: Transaction{ID: "tx-1"}}
	uc := NewUseCase(repo, nil, nil)
	input := TransactionInput{
		Type:           TransactionTypeIncome,
		WalletID:       "wallet-1",
		CategoryID:     "category-1",
		AmountMinor:    100_000,
		TransactionUTC: time.Now().UTC(),
	}

	created, err := uc.Create(context.Background(), "user-1", input)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID != "tx-1" {
		t.Fatalf("expected tx-1, got %+v", created)
	}
	if repo.createInput.Type != TransactionTypeIncome {
		t.Fatalf("expected delegated input, got %+v", repo.createInput)
	}
}

func TestTransactionUseCaseRejectsInvalidInput(t *testing.T) {
	uc := NewUseCase(&fakeTransactionRepository{}, nil, nil)

	if _, err := uc.Create(context.Background(), "user-1", TransactionInput{Type: TransactionTypeIncome}); err == nil {
		t.Fatal("expected invalid transaction input error")
	}
	if _, err := uc.Update(context.Background(), "user-1", "tx-1", TransactionInput{Type: TransactionTypeIncome}); err == nil {
		t.Fatal("expected invalid transaction input error")
	}
}

func TestTransactionUseCaseUpdateValidatesAndDelegates(t *testing.T) {
	repo := &fakeTransactionRepository{updated: Transaction{ID: "tx-1"}}
	uc := NewUseCase(repo, nil, nil)
	input := TransactionInput{
		Type:           TransactionTypeTransfer,
		WalletID:       "wallet-1",
		ToWalletID:     "wallet-2",
		AmountMinor:    100_000,
		TransactionUTC: time.Now().UTC(),
	}

	updated, err := uc.Update(context.Background(), "user-1", "tx-1", input)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.ID != "tx-1" {
		t.Fatalf("expected tx-1, got %+v", updated)
	}
	if repo.updateInput.ToWalletID != "wallet-2" {
		t.Fatalf("expected update input to be delegated, got %+v", repo.updateInput)
	}
}

func TestTransactionUseCaseDelegatesReadAndDelete(t *testing.T) {
	repo := &fakeTransactionRepository{
		listed: []Transaction{{ID: "tx-1"}},
		got:    Transaction{ID: "tx-1"},
	}
	uc := NewUseCase(repo, nil, nil)

	listed, err := uc.List(context.Background(), "user-1", TransactionFilter{Type: TransactionTypeExpense}, page.Params{Limit: 10, Sort: "transaction_at_desc"})
	if err != nil || len(listed.Items) != 1 || listed.Items[0].ID != "tx-1" {
		t.Fatalf("unexpected List result %+v err=%v", listed, err)
	}
	if repo.listFilter.Type != TransactionTypeExpense {
		t.Fatalf("expected list filter to be delegated, got %+v", repo.listFilter)
	}
	if repo.listPage.Limit != 10 {
		t.Fatalf("expected list pagination to be delegated, got %+v", repo.listPage)
	}
	got, err := uc.Get(context.Background(), "user-1", "tx-1")
	if err != nil || got.ID != "tx-1" {
		t.Fatalf("unexpected Get result %+v err=%v", got, err)
	}
	if err := uc.Delete(context.Background(), "user-1", "tx-1"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if repo.deletedID != "tx-1" {
		t.Fatalf("expected delete id tx-1, got %q", repo.deletedID)
	}
}

func TestTransactionUseCasePropagatesRepositoryErrors(t *testing.T) {
	repoErr := errors.New("repo failed")
	uc := NewUseCase(&fakeTransactionRepository{err: repoErr}, nil, nil)
	valid := TransactionInput{
		Type:           TransactionTypeIncome,
		WalletID:       "wallet-1",
		CategoryID:     "category-1",
		AmountMinor:    100_000,
		TransactionUTC: time.Now().UTC(),
	}

	if _, err := uc.Create(context.Background(), "user-1", valid); !errors.Is(err, repoErr) {
		t.Fatalf("expected create repo error, got %v", err)
	}
	if _, err := uc.List(context.Background(), "user-1", TransactionFilter{}, page.Params{Limit: 100}); !errors.Is(err, repoErr) {
		t.Fatalf("expected list repo error, got %v", err)
	}
	if _, err := uc.Update(context.Background(), "user-1", "tx-1", valid); !errors.Is(err, repoErr) {
		t.Fatalf("expected update repo error, got %v", err)
	}
	if err := uc.Delete(context.Background(), "user-1", "tx-1"); !errors.Is(err, repoErr) {
		t.Fatalf("expected delete repo error, got %v", err)
	}
}

func TestTransactionNotFoundRecognizesRepositoryNoRows(t *testing.T) {
	if !NotFound(ErrNotFound) {
		t.Fatal("expected ErrNotFound to be recognized")
	}
	if NotFound(errors.New("other")) {
		t.Fatal("did not expect unrelated error to be recognized")
	}
}
