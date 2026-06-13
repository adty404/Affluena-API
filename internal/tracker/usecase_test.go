package tracker

import (
	"context"
	"testing"
	"time"
)

type fakeInstallmentRepository struct {
	created Installment
	listed  []Installment
	got     Installment
	updated Installment
	paid    InstallmentPayment
	err     error
}

func (f *fakeInstallmentRepository) Create(ctx context.Context, userID string, installment Installment) (Installment, error) {
	return f.created, f.err
}

func (f *fakeInstallmentRepository) List(ctx context.Context, userID string) ([]Installment, error) {
	return f.listed, f.err
}

func (f *fakeInstallmentRepository) Get(ctx context.Context, userID string, id string) (Installment, error) {
	return f.got, f.err
}

func (f *fakeInstallmentRepository) Update(ctx context.Context, userID string, id string, installment Installment) (Installment, error) {
	return f.updated, f.err
}

func (f *fakeInstallmentRepository) Delete(ctx context.Context, userID string, id string) error {
	return f.err
}

func (f *fakeInstallmentRepository) Pay(ctx context.Context, userID string, id string, paidAt time.Time, note string) (InstallmentPayment, error) {
	return f.paid, f.err
}

type fakeSubscriptionRepository struct {
	created Subscription
	listed  []Subscription
	got     Subscription
	updated Subscription
	paid    SubscriptionPayment
	err     error
}

func (f *fakeSubscriptionRepository) Create(ctx context.Context, userID string, subscription Subscription) (Subscription, error) {
	return f.created, f.err
}

func (f *fakeSubscriptionRepository) List(ctx context.Context, userID string) ([]Subscription, error) {
	return f.listed, f.err
}

func (f *fakeSubscriptionRepository) Get(ctx context.Context, userID string, id string) (Subscription, error) {
	return f.got, f.err
}

func (f *fakeSubscriptionRepository) Update(ctx context.Context, userID string, id string, subscription Subscription) (Subscription, error) {
	return f.updated, f.err
}

func (f *fakeSubscriptionRepository) Delete(ctx context.Context, userID string, id string) error {
	return f.err
}

func (f *fakeSubscriptionRepository) Pay(ctx context.Context, userID string, id string, paidAt time.Time, note string) (SubscriptionPayment, error) {
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
	if got, err := uc.ListInstallments(context.Background(), "user-1"); err != nil || len(got) != 1 {
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
	if got, err := uc.ListSubscriptions(context.Background(), "user-1"); err != nil || len(got) != 1 {
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
