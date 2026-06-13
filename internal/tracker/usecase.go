package tracker

import (
	"context"
	"time"

	"affluena/internal/page"
)

type InstallmentRepositoryPort interface {
	Create(ctx context.Context, userID string, installment Installment) (Installment, error)
	List(ctx context.Context, userID string, pagination page.Params) (page.Result[Installment], error)
	Get(ctx context.Context, userID string, id string) (Installment, error)
	Update(ctx context.Context, userID string, id string, installment Installment) (Installment, error)
	Delete(ctx context.Context, userID string, id string) error
	Pay(ctx context.Context, userID string, id string, paidAt time.Time, note string) (InstallmentPayment, error)
}

type SubscriptionRepositoryPort interface {
	Create(ctx context.Context, userID string, subscription Subscription) (Subscription, error)
	List(ctx context.Context, userID string, pagination page.Params) (page.Result[Subscription], error)
	Get(ctx context.Context, userID string, id string) (Subscription, error)
	Update(ctx context.Context, userID string, id string, subscription Subscription) (Subscription, error)
	Delete(ctx context.Context, userID string, id string) error
	Pay(ctx context.Context, userID string, id string, paidAt time.Time, note string) (SubscriptionPayment, error)
}

type UseCase struct {
	installments  InstallmentRepositoryPort
	subscriptions SubscriptionRepositoryPort
}

func NewUseCase(installments InstallmentRepositoryPort, subscriptions SubscriptionRepositoryPort) *UseCase {
	return &UseCase{installments: installments, subscriptions: subscriptions}
}

func (u *UseCase) CreateInstallment(ctx context.Context, userID string, installment Installment) (Installment, error) {
	return u.installments.Create(ctx, userID, installment)
}

func (u *UseCase) ListInstallments(ctx context.Context, userID string, pagination page.Params) (page.Result[Installment], error) {
	return u.installments.List(ctx, userID, pagination)
}

func (u *UseCase) GetInstallment(ctx context.Context, userID string, id string) (Installment, error) {
	return u.installments.Get(ctx, userID, id)
}

func (u *UseCase) UpdateInstallment(ctx context.Context, userID string, id string, installment Installment) (Installment, error) {
	return u.installments.Update(ctx, userID, id, installment)
}

func (u *UseCase) DeleteInstallment(ctx context.Context, userID string, id string) error {
	return u.installments.Delete(ctx, userID, id)
}

func (u *UseCase) PayInstallment(ctx context.Context, userID string, id string, paidAt time.Time, note string) (InstallmentPayment, error) {
	return u.installments.Pay(ctx, userID, id, paidAt, note)
}

func (u *UseCase) CreateSubscription(ctx context.Context, userID string, subscription Subscription) (Subscription, error) {
	return u.subscriptions.Create(ctx, userID, subscription)
}

func (u *UseCase) ListSubscriptions(ctx context.Context, userID string, pagination page.Params) (page.Result[Subscription], error) {
	return u.subscriptions.List(ctx, userID, pagination)
}

func (u *UseCase) GetSubscription(ctx context.Context, userID string, id string) (Subscription, error) {
	return u.subscriptions.Get(ctx, userID, id)
}

func (u *UseCase) UpdateSubscription(ctx context.Context, userID string, id string, subscription Subscription) (Subscription, error) {
	return u.subscriptions.Update(ctx, userID, id, subscription)
}

func (u *UseCase) DeleteSubscription(ctx context.Context, userID string, id string) error {
	return u.subscriptions.Delete(ctx, userID, id)
}

func (u *UseCase) PaySubscription(ctx context.Context, userID string, id string, paidAt time.Time, note string) (SubscriptionPayment, error) {
	return u.subscriptions.Pay(ctx, userID, id, paidAt, note)
}
