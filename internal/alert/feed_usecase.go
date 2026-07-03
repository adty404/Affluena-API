package alert

import (
	"context"
	"fmt"
	"sort"
	"time"

	"affluena-api/internal/page"
	"affluena-api/internal/pkg/money"
)

type FeedUseCase interface {
	List(ctx context.Context, userID string, monthValue string) ([]Alert, error)
	Get(ctx context.Context, userID string, id string) (Alert, error)
}

type feedUseCase struct {
	feedRepo FeedRepository
	budgetUC budgetProvider
}

func NewFeedUseCase(feedRepo FeedRepository, budgetUC budgetProvider) FeedUseCase {
	return &feedUseCase{
		feedRepo: feedRepo,
		budgetUC: budgetUC,
	}
}

func (u *feedUseCase) List(ctx context.Context, userID string, monthValue string) ([]Alert, error) {
	var alerts []Alert

	// 1. Budget alerts
	budgets, err := u.budgetUC.List(ctx, userID, monthValue, page.Params{Limit: 100, Offset: 0})
	if err != nil {
		return nil, fmt.Errorf("failed to list budgets: %w", err)
	}

	for _, b := range budgets.Items {
		if b.UsagePercent >= 80 {
			severity := "warning"
			if b.UsagePercent >= 100 {
				severity = "danger"
			}

			title := "Anggaran hampir habis"
			if b.UsagePercent >= 100 {
				title = "Anggaran terlampaui"
			}

			alerts = append(alerts, Alert{
				ID:          fmt.Sprintf("budget-%s-%s", b.ID, monthValue),
				Type:        "budget",
				Title:       title,
				Module:      "Budget",
				Description: fmt.Sprintf("Pemakaian %.0f%%. Batas Rp %s, terpakai Rp %s.", b.UsagePercent, money.GroupIDR(b.LimitMinor), money.GroupIDR(b.SpentMinor)),
				Severity:    severity,
				CreatedAt:   time.Now().UTC(),
				ActionPath:  "/budgets/alerts",
			})
		}
	}

	// 2. Debt alerts
	debts, err := u.feedRepo.ListOverdueDebts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list overdue debts: %w", err)
	}

	now := time.Now().UTC()
	for _, d := range debts {
		if d.DueDate == nil {
			continue
		}
		daysOverdue := int(now.Sub(*d.DueDate).Hours() / 24)
		if daysOverdue < 0 {
			daysOverdue = 0
		}

		severity := "warning"
		if daysOverdue > 7 {
			severity = "danger"
		}

		alerts = append(alerts, Alert{
			ID:          fmt.Sprintf("debt-%s", d.ID),
			Type:        "debt",
			Title:       fmt.Sprintf("%s jatuh tempo", d.CounterpartyName),
			Module:      "Debt",
			Description: fmt.Sprintf("Utang telah lewat jatuh tempo %d hari. Kirim pengingat atau catat pembayaran.", daysOverdue),
			Severity:    severity,
			CreatedAt:   *d.DueDate,
			ActionPath:  fmt.Sprintf("/debts/%s", d.ID),
		})
	}

	// 3. Recurring alerts
	since := now.Add(-24 * time.Hour)
	activities, err := u.feedRepo.ListRecentRecurringActivities(ctx, userID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to list recent recurring activities: %w", err)
	}

	for _, a := range activities {
		severity := "success"
		title := "Aturan berulang berhasil dijalankan"
		if a.ActionType == "RUN_FAILED" {
			severity = "warning"
			title = "Aturan berulang gagal dijalankan"
		}

		actionPath := "/recurring"
		if a.EntityID != nil {
			actionPath = fmt.Sprintf("/recurring/%s/history", *a.EntityID)
		}

		alerts = append(alerts, Alert{
			ID:          fmt.Sprintf("recurring-%s", a.ID),
			Type:        "recurring",
			Title:       title,
			Module:      "Recurring",
			Description: a.Description,
			Severity:    severity,
			CreatedAt:   a.CreatedAt,
			ActionPath:  actionPath,
		})
	}

	// Sort alerts by CreatedAt descending
	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].CreatedAt.After(alerts[j].CreatedAt)
	})

	return alerts, nil
}

func (u *feedUseCase) Get(ctx context.Context, userID string, id string) (Alert, error) {
	monthValue := time.Now().UTC().Format("2006-01")

	if len(id) > 7 && id[:7] == "budget-" {
		if len(id) >= 7+36+1+7 {
			budgetMonth := id[len(id)-7:]
			monthValue = budgetMonth
		}
	}

	alerts, err := u.List(ctx, userID, monthValue)
	if err != nil {
		return Alert{}, err
	}

	for _, a := range alerts {
		if a.ID == id {
			return a, nil
		}
	}

	return Alert{}, ErrAlertNotFound
}
