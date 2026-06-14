package alert

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"affluena-api/internal/budget"
	"affluena-api/internal/mailer"
	"affluena-api/internal/page"
)

type UseCase interface {
	CheckBudgetAndAlert(ctx context.Context, userID, categoryID string, transactionAt time.Time)
}

type budgetProvider interface {
	List(ctx context.Context, userID string, monthValue string, pagination page.Params) (page.Result[budget.BudgetSummary], error)
}

type useCase struct {
	repo       Repository
	budgetUC   budgetProvider
	mailSender mailer.Mailer
}

func NewUseCase(repo Repository, budgetUC budgetProvider, mailSender mailer.Mailer) UseCase {
	return &useCase{
		repo:       repo,
		budgetUC:   budgetUC,
		mailSender: mailSender,
	}
}

func (u *useCase) CheckBudgetAndAlert(ctx context.Context, userID, categoryID string, transactionAt time.Time) {
	// Execute synchronously here, but caller should invoke via goroutine
	if transactionAt.IsZero() {
		transactionAt = time.Now().UTC()
	}
	monthValue := transactionAt.UTC().Format("2006-01")

	// 1. Get budgets for the month
	budgets, err := u.budgetUC.List(ctx, userID, monthValue, page.Params{Limit: 100, Offset: 0})
	if err != nil {
		slog.Error("failed to list budgets for alert", "error", err, "user_id", userID)
		return
	}

	// 2. Find the specific category budget
	var targetBudget *budget.BudgetSummary
	for _, b := range budgets.Items {
		if b.CategoryID == categoryID {
			targetBudget = &b
			break
		}
	}

	if targetBudget == nil {
		// No budget set for this category
		return
	}

	// 3. Calculate ratio
	if targetBudget.LimitMinor <= 0 {
		return
	}

	ratio := float64(targetBudget.SpentMinor) / float64(targetBudget.LimitMinor)

	// We alert at >= 100% and >= 80% thresholds
	// For simplicity, we just send the email if it crosses these thresholds.
	// In a real production app, we would track if we ALREADY sent the 80% alert this month
	// so we don't spam. But for MVP, this is acceptable.
	var alertType string
	if ratio >= 1.0 {
		alertType = "EXCEEDED"
	} else if ratio >= 0.8 {
		alertType = "WARNING"
	} else {
		// All good
		return
	}

	// 5. Fetch User Email and Category Name
	email, err := u.repo.GetUserEmail(ctx, userID)
	if err != nil {
		slog.Error("failed to get user email for alert", "error", err, "user_id", userID)
		return
	}

	categoryName, err := u.repo.GetCategoryName(ctx, categoryID)
	if err != nil {
		categoryName = "Unknown"
	}

	// 6. Send Email
	subject := fmt.Sprintf("Budget Alert: %s (%.0f%%)", categoryName, ratio*100)
	body := buildBudgetEmailBody(categoryName, targetBudget.SpentMinor, targetBudget.LimitMinor, alertType)

	err = u.mailSender.SendEmail(ctx, []string{email}, subject, body)
	if err != nil {
		slog.Error("failed to send budget alert email", "error", err, "email", email)
	} else {
		slog.Info("budget alert email sent successfully", "user_id", userID, "category", categoryName, "alert_type", alertType)
	}
}

func buildBudgetEmailBody(categoryName string, spent, limit int64, alertType string) string {
	title := "⚠️ Peringatan Budget"
	if alertType == "EXCEEDED" {
		title = "🚨 Budget Terlampaui!"
	}

	html := `
	<html>
	<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
		<div style="max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #ddd; border-radius: 8px;">
			<h2 style="color: #e74c3c; text-align: center;">` + title + `</h2>
			<p>Halo Pengguna Affluena,</p>
			<p>Kami ingin memberitahu bahwa pengeluaran Anda untuk kategori <strong>` + categoryName + `</strong> telah mencapai batas pengawasan.</p>
			
			<div style="background-color: #f9f9f9; padding: 15px; border-radius: 5px; margin: 20px 0;">
				<p style="margin: 0;"><strong>Total Terpakai:</strong> Rp ` + formatMoney(spent) + `</p>
				<p style="margin: 0;"><strong>Batas Budget:</strong> Rp ` + formatMoney(limit) + `</p>
			</div>

			<p>Harap periksa kembali pengeluaran Anda bulan ini melalui dasbor Affluena untuk menjaga kesehatan finansial Anda.</p>
			<hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
			<p style="font-size: 12px; color: #888; text-align: center;">Ini adalah email otomatis dari sistem Affluena. Mohon tidak membalas email ini.</p>
		</div>
	</body>
	</html>
	`
	return html
}

func formatMoney(minor int64) string {
	// Simple formatter since we don't have currency code here
	return fmt.Sprintf("%d", minor)
}
