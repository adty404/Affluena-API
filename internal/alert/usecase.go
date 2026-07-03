package alert

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"time"

	"affluena-api/internal/budget"
	"affluena-api/internal/mailer"
	"affluena-api/internal/page"
	"affluena-api/internal/pkg/money"
)

type UseCase interface {
	CheckBudgetAndAlert(ctx context.Context, userID, categoryID string, transactionAt time.Time)
}

type budgetProvider interface {
	List(ctx context.Context, userID string, monthValue string, pagination page.Params) (page.Result[budget.BudgetSummary], error)
}

// notificationGate consults the user's notification_rules row for the
// budget-alert rule so the alert respects the enabled flag + channel. Satisfied
// by an adapter over *notification.Notifier (Decide). Kept as a local interface
// to avoid an import cycle (notification must not import alert).
type notificationGate interface {
	Decide(ctx context.Context, userID, ruleKey string) (ChannelDecision, error)
}

// ChannelDecision mirrors notification.Decision without importing it directly in
// the interface signature. The adapter in the composition root maps between them.
type ChannelDecision struct {
	Send  bool
	Email bool
	InApp bool
}

type useCase struct {
	repo       Repository
	budgetUC   budgetProvider
	mailSender mailer.Mailer
	gate       notificationGate
}

func NewUseCase(repo Repository, budgetUC budgetProvider, mailSender mailer.Mailer) UseCase {
	return &useCase{
		repo:       repo,
		budgetUC:   budgetUC,
		mailSender: mailSender,
	}
}

// NewUseCaseWithGate is like NewUseCase but consults the notification gate before
// sending, so the budget-alert respects the user's notification_rules
// (enabled + email/in-app channel). When gate is nil the behaviour is unchanged.
func NewUseCaseWithGate(repo Repository, budgetUC budgetProvider, mailSender mailer.Mailer, gate notificationGate) UseCase {
	return &useCase{
		repo:       repo,
		budgetUC:   budgetUC,
		mailSender: mailSender,
		gate:       gate,
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
	// Check deduplication to avoid spamming
	var alertType string
	if ratio >= 1.0 {
		alertType = "EXCEEDED"
	} else if ratio >= 0.8 {
		alertType = "WARNING"
	} else {
		// All good
		return
	}

	// Gate on the user's budget-alert notification rule (enabled + channel). A
	// disabled or missing rule means no send at all; the channel decides whether
	// email goes out. When no gate is wired we preserve the legacy always-email
	// behaviour so existing deployments keep working.
	sendEmail := true
	if u.gate != nil {
		decision, err := u.gate.Decide(ctx, userID, "budget-alert")
		if err != nil {
			slog.Error("failed to consult budget-alert notification rule", "error", err, "user_id", userID)
			return
		}
		if !decision.Send {
			// Rule disabled (or missing): respect the user's choice, send nothing.
			return
		}
		sendEmail = decision.Email
	}

	// Atomically try to mark alert as sending (prevents duplicate emails)
	shouldSend, err := u.repo.TryInsertSentAlert(ctx, userID, categoryID, monthValue, alertType)
	if err != nil {
		slog.Error("failed to check alert deduplication", "error", err, "user_id", userID)
		return
	}
	if !shouldSend {
		// Alert already sent for this month/category/type
		return
	}

	// If the channel is in-app only, there is nothing to email — the sent_alerts
	// row above already records the (deduped) in-app alert surfaced by the feed.
	if !sendEmail {
		slog.Info("budget alert recorded (in-app only, email channel disabled)", "user_id", userID, "alert_type", alertType)
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

	// 6. Send Email (escape user input to prevent XSS)
	safeCategoryName := html.EscapeString(categoryName)
	subject := fmt.Sprintf("Budget Alert: %s (%.0f%%)", safeCategoryName, ratio*100)
	body := buildBudgetEmailBody(safeCategoryName, targetBudget.SpentMinor, targetBudget.LimitMinor, alertType)

	err = u.mailSender.SendEmail(ctx, []string{email}, subject, body)
	if err != nil {
		slog.Error("failed to send budget alert email", "error", err, "email", email)
		// Email failed, delete the alert marker so it can be retried later
		_ = u.repo.DeleteSentAlert(ctx, userID, categoryID, monthValue, alertType)
	} else {
		slog.Info("budget alert email sent successfully", "user_id", userID, "category", safeCategoryName, "alert_type", alertType)
	}
}

func buildBudgetEmailBody(categoryName string, spent, limit int64, alertType string) string {
	title := "⚠️ Peringatan Budget"
	if alertType == "EXCEEDED" {
		title = "🚨 Budget Terlampaui!"
	}

	// Escape user-controlled input to prevent XSS
	safeCategoryName := html.EscapeString(categoryName)

	html := `
	<html>
	<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
		<div style="max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #ddd; border-radius: 8px;">
			<h2 style="color: #e74c3c; text-align: center;">` + title + `</h2>
			<p>Halo Pengguna Affluena,</p>
			<p>Kami ingin memberitahu bahwa pengeluaran Anda untuk kategori <strong>` + safeCategoryName + `</strong> telah mencapai batas pengawasan.</p>

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
	// IDR minor units have no decimal subunit; group thousands with "." so the
	// email shows Rp 20.000.000 instead of a raw 20000000.
	return money.GroupIDR(minor)
}
