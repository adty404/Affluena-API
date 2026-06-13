package debt

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeDebtRepository struct {
	created Debt
	listed  []Debt
	got     Debt
	updated Debt
	paid    DebtPayment
	err     error
}

func (f *fakeDebtRepository) Create(ctx context.Context, userID string, input DebtInput) (Debt, error) {
	if f.err != nil {
		return Debt{}, f.err
	}
	return f.created, nil
}

func (f *fakeDebtRepository) List(ctx context.Context, userID string) ([]Debt, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.listed, nil
}

func (f *fakeDebtRepository) Get(ctx context.Context, userID string, id string) (Debt, error) {
	if f.err != nil {
		return Debt{}, f.err
	}
	return f.got, nil
}

func (f *fakeDebtRepository) Update(ctx context.Context, userID string, id string, update DebtUpdate) (Debt, error) {
	if f.err != nil {
		return Debt{}, f.err
	}
	return f.updated, nil
}

func (f *fakeDebtRepository) Delete(ctx context.Context, userID string, id string) error {
	return f.err
}

func (f *fakeDebtRepository) Pay(ctx context.Context, userID string, id string, amountMinor int64, paidAt time.Time, note string) (DebtPayment, error) {
	if f.err != nil {
		return DebtPayment{}, f.err
	}
	return f.paid, nil
}

func TestDebtUseCaseRejectsInvalidCreateInput(t *testing.T) {
	uc := NewUseCase(&fakeDebtRepository{})

	if _, err := uc.Create(context.Background(), "user-1", DebtInput{Type: DebtType("bad"), PrincipalAmountMinor: 100}); err == nil {
		t.Fatal("expected invalid debt type error")
	}
	if _, err := uc.Create(context.Background(), "user-1", DebtInput{Type: DebtTypeReceivable}); err == nil {
		t.Fatal("expected invalid principal error")
	}
}

func TestDebtUseCaseRejectsInvalidPayment(t *testing.T) {
	uc := NewUseCase(&fakeDebtRepository{})

	if _, err := uc.Pay(context.Background(), "user-1", "debt-1", 0, time.Now().UTC(), ""); err == nil {
		t.Fatal("expected invalid payment amount error")
	}
}

func TestDebtUseCaseDelegatesReadUpdateDelete(t *testing.T) {
	repo := &fakeDebtRepository{
		listed:  []Debt{{ID: "debt-1"}},
		got:     Debt{ID: "debt-1"},
		updated: Debt{ID: "debt-1", Status: DebtStatusCancelled},
	}
	uc := NewUseCase(repo)

	listed, err := uc.List(context.Background(), "user-1")
	if err != nil || len(listed) != 1 || listed[0].ID != "debt-1" {
		t.Fatalf("unexpected List result %+v err=%v", listed, err)
	}
	got, err := uc.Get(context.Background(), "user-1", "debt-1")
	if err != nil || got.ID != "debt-1" {
		t.Fatalf("unexpected Get result %+v err=%v", got, err)
	}
	updated, err := uc.Update(context.Background(), "user-1", "debt-1", DebtUpdate{CounterpartyName: "Friend", Status: DebtStatusCancelled})
	if err != nil || updated.Status != DebtStatusCancelled {
		t.Fatalf("unexpected Update result %+v err=%v", updated, err)
	}
	if err := uc.Delete(context.Background(), "user-1", "debt-1"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func TestDebtNotFoundRecognizesRepositoryNoRows(t *testing.T) {
	if !NotFound(ErrNotFound) {
		t.Fatal("expected ErrNotFound to be recognized")
	}
	if NotFound(errors.New("other")) {
		t.Fatal("did not expect unrelated error to be recognized")
	}
}
