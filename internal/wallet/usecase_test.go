package wallet

import (
	"context"
	"errors"
	"testing"
	"time"

	"affluena-api/internal/page"
)

type fakeWalletRepository struct {
	createInput  CreateWalletInput
	listPage     page.Params
	created      Wallet
	listed       []Wallet
	got          Wallet
	updated      Wallet
	updateCalled bool
	deletedID    string
	err          error
}

func (f *fakeWalletRepository) Create(ctx context.Context, userID string, input CreateWalletInput) (Wallet, error) {
	f.createInput = input
	if f.err != nil {
		return Wallet{}, f.err
	}
	return f.created, nil
}

func (f *fakeWalletRepository) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Wallet], error) {
	f.listPage = pagination
	if f.err != nil {
		return page.Result[Wallet]{}, f.err
	}
	return page.NewResult(f.listed, pagination, len(f.listed)), nil
}

func (f *fakeWalletRepository) Get(ctx context.Context, userID string, id string) (Wallet, error) {
	if f.err != nil {
		return Wallet{}, f.err
	}
	return f.got, nil
}

func (f *fakeWalletRepository) Update(ctx context.Context, userID string, id string, input UpdateWalletInput) (Wallet, error) {
	f.updateCalled = true
	if f.err != nil {
		return Wallet{}, f.err
	}
	return f.updated, nil
}

func (f *fakeWalletRepository) Delete(ctx context.Context, userID string, id string) error {
	f.deletedID = id
	return f.err
}

func (f *fakeWalletRepository) AddMember(ctx context.Context, walletID string, userID string, status string, role string) error {
	return f.err
}

func (f *fakeWalletRepository) FindUserByEmail(ctx context.Context, email string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "user-2", nil
}

func (f *fakeWalletRepository) RespondInvite(ctx context.Context, walletID string, userID string, status string) error {
	return f.err
}

func (f *fakeWalletRepository) GetAccessLevel(ctx context.Context, userID string, walletID string) (AccessLevel, error) {
	if f.err != nil {
		return AccessNone, f.err
	}
	return AccessOwner, nil
}

func (f *fakeWalletRepository) GetMembers(ctx context.Context, walletID string) ([]WalletMember, error) {
	if f.err != nil {
		return nil, f.err
	}
	return nil, nil
}

func (f *fakeWalletRepository) GetAnalytics(ctx context.Context, userID string, walletID string, month string) (WalletAnalytics, error) {
	if f.err != nil {
		return WalletAnalytics{}, f.err
	}
	return WalletAnalytics{WalletID: walletID, Month: month}, nil
}

func TestWalletUseCaseCreateDefaultsCurrency(t *testing.T) {
	repo := &fakeWalletRepository{created: Wallet{ID: "wallet-1"}}
	uc := NewUseCase(repo, nil)

	created, err := uc.Create(context.Background(), "user-1", CreateWalletInput{
		Name:         "Cash",
		Type:         "cash",
		BalanceMinor: 1000,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID != "wallet-1" {
		t.Fatalf("expected created wallet, got %+v", created)
	}
	if repo.createInput.CurrencyCode != "IDR" {
		t.Fatalf("expected default currency IDR, got %q", repo.createInput.CurrencyCode)
	}
}

func TestWalletUseCaseCreatePreservesExplicitCurrency(t *testing.T) {
	repo := &fakeWalletRepository{created: Wallet{ID: "wallet-1"}}
	uc := NewUseCase(repo, nil)

	_, err := uc.Create(context.Background(), "user-1", CreateWalletInput{
		Name:         "USD cash",
		Type:         "cash",
		CurrencyCode: "USD",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if repo.createInput.CurrencyCode != "USD" {
		t.Fatalf("expected explicit currency USD to be preserved, got %q", repo.createInput.CurrencyCode)
	}
}

func TestWalletUseCaseRejectsInvalidType(t *testing.T) {
	uc := NewUseCase(&fakeWalletRepository{}, nil)

	if _, err := uc.Create(context.Background(), "user-1", CreateWalletInput{Name: "Bad", Type: "crypto"}); err == nil {
		t.Fatal("expected invalid wallet type error")
	}
	if _, err := uc.Update(context.Background(), "user-1", "wallet-1", UpdateWalletInput{Name: "Bad", Type: "crypto", CurrencyCode: "IDR"}); err == nil {
		t.Fatal("expected invalid wallet type error")
	}
}

func TestWalletUseCaseRejectsPublicGoalWalletWrites(t *testing.T) {
	uc := NewUseCase(&fakeWalletRepository{}, nil)

	if _, err := uc.Create(context.Background(), "user-1", CreateWalletInput{Name: "Goal", Type: "goal"}); err == nil {
		t.Fatal("expected public goal wallet create to fail")
	}
	if _, err := uc.Update(context.Background(), "user-1", "wallet-1", UpdateWalletInput{Name: "Goal", Type: "goal", CurrencyCode: "IDR"}); err == nil {
		t.Fatal("expected public goal wallet update to fail")
	}
}

func TestWalletUseCaseUpdateDelegatesValidInput(t *testing.T) {
	repo := &fakeWalletRepository{
		got:     Wallet{ID: "wallet-1"},
		updated: Wallet{ID: "wallet-1", Name: "Updated"},
	}
	uc := NewUseCase(repo, nil)

	updated, err := uc.Update(context.Background(), "user-1", "wallet-1", UpdateWalletInput{
		Name:         "Updated",
		Type:         "bank",
		CurrencyCode: "IDR",
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Name != "Updated" {
		t.Fatalf("expected updated wallet, got %+v", updated)
	}
}

func TestWalletUseCaseRejectsManagedGoalWalletMutation(t *testing.T) {
	goalID := "goal-1"
	repo := &fakeWalletRepository{got: Wallet{ID: "wallet-1", GoalID: &goalID}}
	uc := NewUseCase(repo, nil)

	if _, err := uc.Update(context.Background(), "user-1", "wallet-1", UpdateWalletInput{Name: "Goal as bank", Type: "bank", CurrencyCode: "IDR"}); !errors.Is(err, ErrGoalWalletReadOnly) {
		t.Fatalf("expected ErrGoalWalletReadOnly on update, got %v", err)
	}
	if repo.updateCalled {
		t.Fatal("expected update repository call to be skipped for managed goal wallet")
	}
	if err := uc.Delete(context.Background(), "user-1", "wallet-1"); !errors.Is(err, ErrGoalWalletReadOnly) {
		t.Fatalf("expected ErrGoalWalletReadOnly on delete, got %v", err)
	}
	if repo.deletedID != "" {
		t.Fatalf("expected delete repository call to be skipped for managed goal wallet, got %q", repo.deletedID)
	}
}

func TestWalletUseCasePropagatesRepositoryErrors(t *testing.T) {
	repoErr := errors.New("repo failed")
	uc := NewUseCase(&fakeWalletRepository{err: repoErr}, nil)

	if _, err := uc.Create(context.Background(), "user-1", CreateWalletInput{Name: "Cash", Type: "cash"}); !errors.Is(err, repoErr) {
		t.Fatalf("expected create repo error, got %v", err)
	}
	if _, err := uc.List(context.Background(), "user-1", page.Params{Limit: 100}); !errors.Is(err, repoErr) {
		t.Fatalf("expected list repo error, got %v", err)
	}
	if _, err := uc.Get(context.Background(), "user-1", "wallet-1"); !errors.Is(err, repoErr) {
		t.Fatalf("expected get repo error, got %v", err)
	}
	if err := uc.Delete(context.Background(), "user-1", "wallet-1"); !errors.Is(err, repoErr) {
		t.Fatalf("expected delete repo error, got %v", err)
	}
}

func TestWalletUseCaseDelegatesReadAndDelete(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeWalletRepository{
		listed: []Wallet{{ID: "wallet-1", CreatedAt: now}},
		got:    Wallet{ID: "wallet-1", CreatedAt: now},
	}
	uc := NewUseCase(repo, nil)

	listed, err := uc.List(context.Background(), "user-1", page.Params{Limit: 10, Sort: "created_at_desc"})
	if err != nil || len(listed.Items) != 1 || listed.Items[0].ID != "wallet-1" {
		t.Fatalf("unexpected List result %+v err=%v", listed, err)
	}
	if repo.listPage.Limit != 10 {
		t.Fatalf("expected pagination to be delegated, got %+v", repo.listPage)
	}
	got, err := uc.Get(context.Background(), "user-1", "wallet-1")
	if err != nil || got.ID != "wallet-1" {
		t.Fatalf("unexpected Get result %+v err=%v", got, err)
	}
	if err := uc.Delete(context.Background(), "user-1", "wallet-1"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if repo.deletedID != "wallet-1" {
		t.Fatalf("expected delete id wallet-1, got %q", repo.deletedID)
	}
}

func TestWalletNotFoundRecognizesRepositoryNoRows(t *testing.T) {
	if !NotFound(ErrNotFound) {
		t.Fatal("expected ErrNotFound to be recognized")
	}
	if NotFound(errors.New("other")) {
		t.Fatal("did not expect unrelated error to be recognized")
	}
}
