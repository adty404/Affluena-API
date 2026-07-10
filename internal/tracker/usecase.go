package tracker

import (
	"context"
	"time"

	"affluena-api/internal/activity"
	"affluena-api/internal/page"
)

type InstallmentRepositoryPort interface {
	Create(ctx context.Context, userID string, installment Installment) (Installment, error)
	List(ctx context.Context, userID string, pagination page.Params) (page.Result[Installment], error)
	Get(ctx context.Context, userID string, id string) (Installment, error)
	Update(ctx context.Context, userID string, id string, installment Installment) (Installment, error)
	Delete(ctx context.Context, userID string, id string) error
	Pay(ctx context.Context, userID string, id string, paidAt time.Time, note string) (InstallmentPayment, error)
	ListPayments(ctx context.Context, userID string, id string) ([]InstallmentPaymentRecord, error)
}

type SubscriptionRepositoryPort interface {
	Create(ctx context.Context, userID string, subscription Subscription) (Subscription, error)
	List(ctx context.Context, userID string, pagination page.Params) (page.Result[Subscription], error)
	Get(ctx context.Context, userID string, id string) (Subscription, error)
	Update(ctx context.Context, userID string, id string, subscription Subscription) (Subscription, error)
	Delete(ctx context.Context, userID string, id string) error
	Pay(ctx context.Context, userID string, id string, paidAt time.Time, note string) (SubscriptionPayment, error)
	ListPayments(ctx context.Context, userID string, id string) ([]SubscriptionPaymentRecord, error)
}

type UseCase struct {
	installments  InstallmentRepositoryPort
	subscriptions SubscriptionRepositoryPort
	activityUC    activity.UseCase
}

func NewUseCase(installments InstallmentRepositoryPort, subscriptions SubscriptionRepositoryPort, activityUC activity.UseCase) *UseCase {
	return &UseCase{installments: installments, subscriptions: subscriptions, activityUC: activityUC}
}

func (u *UseCase) CreateInstallment(ctx context.Context, userID string, installment Installment) (Installment, error) {
	inst, err := u.installments.Create(ctx, userID, installment)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "CREATE", "INSTALLMENT", &inst.ID, "Membuat cicilan baru: "+installment.Name)
	}
	return inst, err
}

func (u *UseCase) ListInstallments(ctx context.Context, userID string, pagination page.Params) (page.Result[Installment], error) {
	return u.installments.List(ctx, userID, pagination)
}

func (u *UseCase) GetInstallment(ctx context.Context, userID string, id string) (Installment, error) {
	return u.installments.Get(ctx, userID, id)
}

func (u *UseCase) UpdateInstallment(ctx context.Context, userID string, id string, installment Installment) (Installment, error) {
	inst, err := u.installments.Update(ctx, userID, id, installment)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "UPDATE", "INSTALLMENT", &id, "Mengubah rincian cicilan: "+installment.Name)
	}
	return inst, err
}

func (u *UseCase) DeleteInstallment(ctx context.Context, userID string, id string) error {
	err := u.installments.Delete(ctx, userID, id)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "DELETE", "INSTALLMENT", &id, "Menghapus data cicilan")
	}
	return err
}

func (u *UseCase) PayInstallment(ctx context.Context, userID string, id string, paidAt time.Time, note string) (InstallmentPayment, error) {
	pay, err := u.installments.Pay(ctx, userID, id, paidAt, note)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "UPDATE", "INSTALLMENT_PAYMENT", &id, "Membayar tagihan cicilan")
	}
	return pay, err
}

func (u *UseCase) ListInstallmentPayments(ctx context.Context, userID string, id string) ([]InstallmentPaymentRecord, error) {
	return u.installments.ListPayments(ctx, userID, id)
}

func (u *UseCase) CreateSubscription(ctx context.Context, userID string, subscription Subscription) (Subscription, error) {
	sub, err := u.subscriptions.Create(ctx, userID, subscription)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "CREATE", "SUBSCRIPTION", &sub.ID, "Membuat langganan baru: "+subscription.Name)
	}
	return sub, err
}

func (u *UseCase) ListSubscriptions(ctx context.Context, userID string, pagination page.Params) (page.Result[Subscription], error) {
	return u.subscriptions.List(ctx, userID, pagination)
}

func (u *UseCase) GetSubscription(ctx context.Context, userID string, id string) (Subscription, error) {
	return u.subscriptions.Get(ctx, userID, id)
}

func (u *UseCase) UpdateSubscription(ctx context.Context, userID string, id string, subscription Subscription) (Subscription, error) {
	sub, err := u.subscriptions.Update(ctx, userID, id, subscription)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "UPDATE", "SUBSCRIPTION", &id, "Mengubah rincian langganan: "+subscription.Name)
	}
	return sub, err
}

func (u *UseCase) DeleteSubscription(ctx context.Context, userID string, id string) error {
	err := u.subscriptions.Delete(ctx, userID, id)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "DELETE", "SUBSCRIPTION", &id, "Menghapus data langganan")
	}
	return err
}

func (u *UseCase) PaySubscription(ctx context.Context, userID string, id string, paidAt time.Time, note string) (SubscriptionPayment, error) {
	pay, err := u.subscriptions.Pay(ctx, userID, id, paidAt, note)
	if err == nil && u.activityUC != nil {
		u.activityUC.LogActivity(ctx, userID, "UPDATE", "SUBSCRIPTION_PAYMENT", &id, "Membayar tagihan langganan")
	}
	return pay, err
}

func (u *UseCase) ListSubscriptionPayments(ctx context.Context, userID string, id string) ([]SubscriptionPaymentRecord, error) {
	return u.subscriptions.ListPayments(ctx, userID, id)
}
