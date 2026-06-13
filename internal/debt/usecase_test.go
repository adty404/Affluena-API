package debt

import (
	"context"
	"errors"
	"testing"
	"time"

	"affluena-api/internal/page"
)

type fakeDebtRepository struct {
	createInput DebtInput
	updateInput DebtUpdate
	paidAmount  int64
	paidAt      time.Time
	payNote     string
	created     Debt
	listPage    page.Params
	listed      []Debt
	got         Debt
	updated     Debt
	paid        DebtPayment
	err         error
}

func (f *fakeDebtRepository) Create(ctx context.Context, userID string, input DebtInput) (Debt, error) {
	f.createInput = input
	if f.err != nil {
		return Debt{}, f.err
	}
	return f.created, nil
}

func (f *fakeDebtRepository) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Debt], error) {
	f.listPage = pagination
	if f.err != nil {
		return page.Result[Debt]{}, f.err
	}
	return page.NewResult(f.listed, pagination, len(f.listed)), nil
}

func (f *fakeDebtRepository) Get(ctx context.Context, userID string, id string) (Debt, error) {
	if f.err != nil {
		return Debt{}, f.err
	}
	return f.got, nil
}

func (f *fakeDebtRepository) Update(ctx context.Context, userID string, id string, update DebtUpdate) (Debt, error) {
	f.updateInput = update
	if f.err != nil {
		return Debt{}, f.err
	}
	return f.updated, nil
}

func (f *fakeDebtRepository) Delete(ctx context.Context, userID string, id string) error {
	return f.err
}

func (f *fakeDebtRepository) Pay(ctx context.Context, userID string, id string, amountMinor int64, paidAt time.Time, note string) (DebtPayment, error) {
	f.paidAmount = amountMinor
	f.paidAt = paidAt
	f.payNote = note
	if f.err != nil {
		return DebtPayment{}, f.err
	}
	return f.paid, nil
}

func TestDebtUseCaseRejectsInvalidCreateInput(t *testing.T) {
	uc := NewUseCase(&fakeDebtRepository{}, nil)

	if _, err := uc.Create(context.Background(), "user-1", DebtInput{Type: DebtType("bad"), PrincipalAmountMinor: 100}); err == nil {
		t.Fatal("expected invalid debt type error")
	}
	if _, err := uc.Create(context.Background(), "user-1", DebtInput{Type: DebtTypeReceivable}); err == nil {
		t.Fatal("expected invalid principal error")
	}
}

func TestDebtUseCaseCreateDelegatesValidInput(t *testing.T) {
	repo := &fakeDebtRepository{created: Debt{ID: "debt-1"}}
	uc := NewUseCase(repo, nil)

	input := DebtInput{
		Type:                 DebtTypeReceivable,
		CounterpartyName:     "Friend",
		PrincipalAmountMinor: 100_000,
	}
	created, err := uc.Create(context.Background(), "user-1", input)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID != "debt-1" {
		t.Fatalf("expected debt-1, got %+v", created)
	}
	if repo.createInput.CounterpartyName != "Friend" {
		t.Fatalf("expected create input to be delegated, got %+v", repo.createInput)
	}
}

func TestDebtUseCaseRejectsInvalidPayment(t *testing.T) {
	uc := NewUseCase(&fakeDebtRepository{}, nil)

	if _, err := uc.Pay(context.Background(), "user-1", "debt-1", 0, time.Now().UTC(), ""); err == nil {
		t.Fatal("expected invalid payment amount error")
	}
}

func TestDebtUseCaseRejectsInvalidUpdateStatus(t *testing.T) {
	uc := NewUseCase(&fakeDebtRepository{}, nil)

	if _, err := uc.Update(context.Background(), "user-1", "debt-1", DebtUpdate{Status: DebtStatus("lost")}); err == nil {
		t.Fatal("expected invalid debt status error")
	}
}

func TestDebtUseCasePayDelegatesPositivePayment(t *testing.T) {
	paidAt := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	repo := &fakeDebtRepository{paid: DebtPayment{ID: "payment-1"}}
	uc := NewUseCase(repo, nil)

	payment, err := uc.Pay(context.Background(), "user-1", "debt-1", 50_000, paidAt, "partial")
	if err != nil {
		t.Fatalf("Pay returned error: %v", err)
	}
	if payment.ID != "payment-1" {
		t.Fatalf("expected payment-1, got %+v", payment)
	}
	if repo.paidAmount != 50_000 || !repo.paidAt.Equal(paidAt) || repo.payNote != "partial" {
		t.Fatalf("unexpected delegated payment amount/time/note: %d %s %q", repo.paidAmount, repo.paidAt, repo.payNote)
	}
}

func TestDebtUseCaseDelegatesReadUpdateDelete(t *testing.T) {
	repo := &fakeDebtRepository{
		listed:  []Debt{{ID: "debt-1"}},
		got:     Debt{ID: "debt-1"},
		updated: Debt{ID: "debt-1", Status: DebtStatusCancelled},
	}
	uc := NewUseCase(repo, nil)

	listed, err := uc.List(context.Background(), "user-1", page.Params{Limit: 10, Sort: "opened_at_desc"})
	if err != nil || len(listed.Items) != 1 || listed.Items[0].ID != "debt-1" {
		t.Fatalf("unexpected List result %+v err=%v", listed, err)
	}
	if repo.listPage.Limit != 10 || repo.listPage.Sort != "opened_at_desc" {
		t.Fatalf("expected repository to receive pagination, got %+v", repo.listPage)
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

func TestDebtUseCasePropagatesRepositoryErrors(t *testing.T) {
	repoErr := errors.New("repo failed")
	uc := NewUseCase(&fakeDebtRepository{err: repoErr}, nil)

	if _, err := uc.Create(context.Background(), "user-1", DebtInput{Type: DebtTypeReceivable, PrincipalAmountMinor: 100_000}); !errors.Is(err, repoErr) {
		t.Fatalf("expected create repo error, got %v", err)
	}
	if _, err := uc.List(context.Background(), "user-1", page.Params{Limit: 100}); !errors.Is(err, repoErr) {
		t.Fatalf("expected list repo error, got %v", err)
	}
	if _, err := uc.Update(context.Background(), "user-1", "debt-1", DebtUpdate{Status: DebtStatusOpen}); !errors.Is(err, repoErr) {
		t.Fatalf("expected update repo error, got %v", err)
	}
	if err := uc.Delete(context.Background(), "user-1", "debt-1"); !errors.Is(err, repoErr) {
		t.Fatalf("expected delete repo error, got %v", err)
	}
	if _, err := uc.Pay(context.Background(), "user-1", "debt-1", 1, time.Now().UTC(), ""); !errors.Is(err, repoErr) {
		t.Fatalf("expected pay repo error, got %v", err)
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
