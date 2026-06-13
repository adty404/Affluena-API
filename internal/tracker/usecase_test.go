package tracker

import (
	"context"
	"errors"
	"testing"
	"time"

	"affluena/internal/page"
)

type fakeInstallmentRepository struct {
	createInput Installment
	updateInput Installment
	paidAt      time.Time
	payNote     string
	created     Installment
	listPage    page.Params
	listed      []Installment
	got         Installment
	updated     Installment
	paid        InstallmentPayment
	err         error
}

func (f *fakeInstallmentRepository) Create(ctx context.Context, userID string, installment Installment) (Installment, error) {
	f.createInput = installment
	return f.created, f.err
}

func (f *fakeInstallmentRepository) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Installment], error) {
	f.listPage = pagination
	if f.err != nil {
		return page.Result[Installment]{}, f.err
	}
	return page.NewResult(f.listed, pagination, len(f.listed)), nil
}

func (f *fakeInstallmentRepository) Get(ctx context.Context, userID string, id string) (Installment, error) {
	return f.got, f.err
}

func (f *fakeInstallmentRepository) Update(ctx context.Context, userID string, id string, installment Installment) (Installment, error) {
	f.updateInput = installment
	return f.updated, f.err
}

func (f *fakeInstallmentRepository) Delete(ctx context.Context, userID string, id string) error {
	return f.err
}

func (f *fakeInstallmentRepository) Pay(ctx context.Context, userID string, id string, paidAt time.Time, note string) (InstallmentPayment, error) {
	f.paidAt = paidAt
	f.payNote = note
	return f.paid, f.err
}

type fakeSubscriptionRepository struct {
	createInput Subscription
	updateInput Subscription
	paidAt      time.Time
	payNote     string
	created     Subscription
	listPage    page.Params
	listed      []Subscription
	got         Subscription
	updated     Subscription
	paid        SubscriptionPayment
	err         error
}

func (f *fakeSubscriptionRepository) Create(ctx context.Context, userID string, subscription Subscription) (Subscription, error) {
	f.createInput = subscription
	return f.created, f.err
}

func (f *fakeSubscriptionRepository) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Subscription], error) {
	f.listPage = pagination
	if f.err != nil {
		return page.Result[Subscription]{}, f.err
	}
	return page.NewResult(f.listed, pagination, len(f.listed)), nil
}

func (f *fakeSubscriptionRepository) Get(ctx context.Context, userID string, id string) (Subscription, error) {
	return f.got, f.err
}

func (f *fakeSubscriptionRepository) Update(ctx context.Context, userID string, id string, subscription Subscription) (Subscription, error) {
	f.updateInput = subscription
	return f.updated, f.err
}

func (f *fakeSubscriptionRepository) Delete(ctx context.Context, userID string, id string) error {
	return f.err
}

func (f *fakeSubscriptionRepository) Pay(ctx context.Context, userID string, id string, paidAt time.Time, note string) (SubscriptionPayment, error) {
	f.paidAt = paidAt
	f.payNote = note
	return f.paid, f.err
}

func TestTrackerUseCaseDelegatesInstallments(t *testing.T) {
	installments := &fakeInstallmentRepository{
		created: Installment{ID: "installment-1"},
		listed:  []Installment{{ID: "installment-1"}},
		got:     Installment{ID: "installment-1"},
		updated: Installment{ID: "installment-1", Status: InstallmentStatusActive},
		paid:    InstallmentPayment{Installment: Installment{ID: "installment-1", RemainingMonths: 1}},
	}
	uc := NewUseCase(installments, &fakeSubscriptionRepository{})

	if got, err := uc.CreateInstallment(context.Background(), "user-1", Installment{}); err != nil || got.ID != "installment-1" {
		t.Fatalf("unexpected CreateInstallment result %+v err=%v", got, err)
	}
	if got, err := uc.ListInstallments(context.Background(), "user-1", page.Params{Limit: 10, Sort: "created_at_desc"}); err != nil || len(got.Items) != 1 {
		t.Fatalf("unexpected ListInstallments result %+v err=%v", got, err)
	}
	if got, err := uc.GetInstallment(context.Background(), "user-1", "installment-1"); err != nil || got.ID != "installment-1" {
		t.Fatalf("unexpected GetInstallment result %+v err=%v", got, err)
	}
	if got, err := uc.UpdateInstallment(context.Background(), "user-1", "installment-1", Installment{}); err != nil || got.ID != "installment-1" {
		t.Fatalf("unexpected UpdateInstallment result %+v err=%v", got, err)
	}
	if err := uc.DeleteInstallment(context.Background(), "user-1", "installment-1"); err != nil {
		t.Fatalf("DeleteInstallment returned error: %v", err)
	}
	if got, err := uc.PayInstallment(context.Background(), "user-1", "installment-1", time.Now().UTC(), ""); err != nil || got.Installment.ID != "installment-1" {
		t.Fatalf("unexpected PayInstallment result %+v err=%v", got, err)
	}
}

func TestTrackerUseCasePassesInstallmentInputsThrough(t *testing.T) {
	installments := &fakeInstallmentRepository{created: Installment{ID: "installment-1"}, updated: Installment{ID: "installment-1"}, paid: InstallmentPayment{Installment: Installment{ID: "installment-1"}}}
	uc := NewUseCase(installments, &fakeSubscriptionRepository{})

	input := Installment{Name: "Laptop", RemainingMonths: 6}
	if _, err := uc.CreateInstallment(context.Background(), "user-1", input); err != nil {
		t.Fatalf("CreateInstallment returned error: %v", err)
	}
	if installments.createInput.Name != "Laptop" {
		t.Fatalf("expected create input to be captured, got %+v", installments.createInput)
	}

	input.Name = "Laptop updated"
	if _, err := uc.UpdateInstallment(context.Background(), "user-1", "installment-1", input); err != nil {
		t.Fatalf("UpdateInstallment returned error: %v", err)
	}
	if installments.updateInput.Name != "Laptop updated" {
		t.Fatalf("expected update input to be captured, got %+v", installments.updateInput)
	}

	paidAt := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	if _, err := uc.PayInstallment(context.Background(), "user-1", "installment-1", paidAt, "paid"); err != nil {
		t.Fatalf("PayInstallment returned error: %v", err)
	}
	if !installments.paidAt.Equal(paidAt) || installments.payNote != "paid" {
		t.Fatalf("expected payment input %s/paid, got %s/%q", paidAt, installments.paidAt, installments.payNote)
	}
}

func TestTrackerUseCaseDelegatesSubscriptions(t *testing.T) {
	subscriptions := &fakeSubscriptionRepository{
		created: Subscription{ID: "subscription-1"},
		listed:  []Subscription{{ID: "subscription-1"}},
		got:     Subscription{ID: "subscription-1"},
		updated: Subscription{ID: "subscription-1", Status: SubscriptionStatusActive},
		paid:    SubscriptionPayment{Subscription: Subscription{ID: "subscription-1"}},
	}
	uc := NewUseCase(&fakeInstallmentRepository{}, subscriptions)

	if got, err := uc.CreateSubscription(context.Background(), "user-1", Subscription{}); err != nil || got.ID != "subscription-1" {
		t.Fatalf("unexpected CreateSubscription result %+v err=%v", got, err)
	}
	if got, err := uc.ListSubscriptions(context.Background(), "user-1", page.Params{Limit: 10, Sort: "next_due_date_asc"}); err != nil || len(got.Items) != 1 {
		t.Fatalf("unexpected ListSubscriptions result %+v err=%v", got, err)
	}
	if got, err := uc.GetSubscription(context.Background(), "user-1", "subscription-1"); err != nil || got.ID != "subscription-1" {
		t.Fatalf("unexpected GetSubscription result %+v err=%v", got, err)
	}
	if got, err := uc.UpdateSubscription(context.Background(), "user-1", "subscription-1", Subscription{}); err != nil || got.ID != "subscription-1" {
		t.Fatalf("unexpected UpdateSubscription result %+v err=%v", got, err)
	}
	if err := uc.DeleteSubscription(context.Background(), "user-1", "subscription-1"); err != nil {
		t.Fatalf("DeleteSubscription returned error: %v", err)
	}
	if got, err := uc.PaySubscription(context.Background(), "user-1", "subscription-1", time.Now().UTC(), ""); err != nil || got.Subscription.ID != "subscription-1" {
		t.Fatalf("unexpected PaySubscription result %+v err=%v", got, err)
	}
}

func TestTrackerUseCasePassesSubscriptionInputsThrough(t *testing.T) {
	subscriptions := &fakeSubscriptionRepository{created: Subscription{ID: "subscription-1"}, updated: Subscription{ID: "subscription-1"}, paid: SubscriptionPayment{Subscription: Subscription{ID: "subscription-1"}}}
	uc := NewUseCase(&fakeInstallmentRepository{}, subscriptions)

	input := Subscription{Name: "Internet", AccountDetail: "personal@example.com", BillingCycle: BillingCycleMonthly}
	if _, err := uc.CreateSubscription(context.Background(), "user-1", input); err != nil {
		t.Fatalf("CreateSubscription returned error: %v", err)
	}
	if subscriptions.createInput.Name != "Internet" || subscriptions.createInput.AccountDetail != "personal@example.com" {
		t.Fatalf("expected create input to be captured, got %+v", subscriptions.createInput)
	}

	input.Name = "Internet updated"
	input.AccountDetail = "work@example.com"
	if _, err := uc.UpdateSubscription(context.Background(), "user-1", "subscription-1", input); err != nil {
		t.Fatalf("UpdateSubscription returned error: %v", err)
	}
	if subscriptions.updateInput.Name != "Internet updated" || subscriptions.updateInput.AccountDetail != "work@example.com" {
		t.Fatalf("expected update input to be captured, got %+v", subscriptions.updateInput)
	}

	paidAt := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	if _, err := uc.PaySubscription(context.Background(), "user-1", "subscription-1", paidAt, "paid"); err != nil {
		t.Fatalf("PaySubscription returned error: %v", err)
	}
	if !subscriptions.paidAt.Equal(paidAt) || subscriptions.payNote != "paid" {
		t.Fatalf("expected payment input %s/paid, got %s/%q", paidAt, subscriptions.paidAt, subscriptions.payNote)
	}
}

func TestTrackerUseCasePropagatesRepositoryErrors(t *testing.T) {
	repoErr := errors.New("repo failed")
	uc := NewUseCase(&fakeInstallmentRepository{err: repoErr}, &fakeSubscriptionRepository{err: repoErr})

	if _, err := uc.CreateInstallment(context.Background(), "user-1", Installment{}); !errors.Is(err, repoErr) {
		t.Fatalf("expected installment create error, got %v", err)
	}
	if _, err := uc.ListInstallments(context.Background(), "user-1", page.Params{Limit: 100}); !errors.Is(err, repoErr) {
		t.Fatalf("expected installment list error, got %v", err)
	}
	if err := uc.DeleteInstallment(context.Background(), "user-1", "installment-1"); !errors.Is(err, repoErr) {
		t.Fatalf("expected installment delete error, got %v", err)
	}
	if _, err := uc.CreateSubscription(context.Background(), "user-1", Subscription{}); !errors.Is(err, repoErr) {
		t.Fatalf("expected subscription create error, got %v", err)
	}
	if _, err := uc.ListSubscriptions(context.Background(), "user-1", page.Params{Limit: 100}); !errors.Is(err, repoErr) {
		t.Fatalf("expected subscription list error, got %v", err)
	}
	if err := uc.DeleteSubscription(context.Background(), "user-1", "subscription-1"); !errors.Is(err, repoErr) {
		t.Fatalf("expected subscription delete error, got %v", err)
	}
}
