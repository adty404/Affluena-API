package quickentry

import (
	"context"
	"errors"
	"testing"
	"time"

	"affluena/internal/transaction"
)

type fakeTemplateRepository struct {
	created   Template
	listed    []Template
	got       Template
	updated   Template
	deletedID string
	err       error
}

func (f *fakeTemplateRepository) Create(ctx context.Context, userID string, template Template) (Template, error) {
	if f.err != nil {
		return Template{}, f.err
	}
	return f.created, nil
}

func (f *fakeTemplateRepository) List(ctx context.Context, userID string) ([]Template, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.listed, nil
}

func (f *fakeTemplateRepository) Get(ctx context.Context, userID string, id string) (Template, error) {
	if f.err != nil {
		return Template{}, f.err
	}
	return f.got, nil
}

func (f *fakeTemplateRepository) Update(ctx context.Context, userID string, id string, template Template) (Template, error) {
	if f.err != nil {
		return Template{}, f.err
	}
	return f.updated, nil
}

func (f *fakeTemplateRepository) Delete(ctx context.Context, userID string, id string) error {
	f.deletedID = id
	return f.err
}

type fakeTransactionCreator struct {
	input   transaction.TransactionInput
	created transaction.Transaction
	err     error
}

func (f *fakeTransactionCreator) Create(ctx context.Context, userID string, input transaction.TransactionInput) (transaction.Transaction, error) {
	f.input = input
	if f.err != nil {
		return transaction.Transaction{}, f.err
	}
	return f.created, nil
}

func TestQuickEntryUseCaseRejectsInvalidTemplate(t *testing.T) {
	uc := NewUseCase(&fakeTemplateRepository{}, &fakeTransactionCreator{})

	_, err := uc.Create(context.Background(), "user-1", Template{
		Name:        "Bad",
		Type:        string(transaction.TransactionTypeIncome),
		WalletID:    "wallet-1",
		AmountMinor: 100,
	})
	if err == nil {
		t.Fatal("expected missing category validation error")
	}
}

func TestQuickEntryUseCaseExecuteCreatesTransactionFromTemplate(t *testing.T) {
	executedAt := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	templates := &fakeTemplateRepository{got: Template{
		ID:          "template-1",
		Type:        string(transaction.TransactionTypeExpense),
		WalletID:    "wallet-1",
		CategoryID:  "category-1",
		AmountMinor: 50_000,
		Note:        "Default note",
	}}
	transactions := &fakeTransactionCreator{created: transaction.Transaction{ID: "tx-1"}}
	uc := NewUseCase(templates, transactions)

	result, err := uc.Execute(context.Background(), "user-1", "template-1", ExecuteInput{
		TransactionAt: executedAt,
		Note:          "Override note",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Transaction.ID != "tx-1" {
		t.Fatalf("expected tx-1, got %+v", result)
	}
	if transactions.input.Note != "Override note" || !transactions.input.TransactionUTC.Equal(executedAt) {
		t.Fatalf("unexpected transaction input %+v", transactions.input)
	}
}

func TestQuickEntryNotFoundRecognizesRepositoryNoRows(t *testing.T) {
	if !NotFound(ErrNotFound) {
		t.Fatal("expected ErrNotFound to be recognized")
	}
	if NotFound(errors.New("other")) {
		t.Fatal("did not expect unrelated error to be recognized")
	}
}
