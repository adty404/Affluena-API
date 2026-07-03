package notification

import (
	"fmt"
	"html"

	"affluena-api/internal/pkg/money"
)

// entityLabelID maps an internal entity type to its Indonesian noun for use in
// notification copy.
func entityLabelID(entityType string) string {
	switch entityType {
	case "subscription":
		return "Langganan"
	case "installment":
		return "Cicilan"
	case "debt":
		return "Utang"
	default:
		return "Tagihan"
	}
}

// entityActionPath returns the client route a due-reminder notification links to.
func entityActionPath(entityType, entityID string) string {
	switch entityType {
	case "subscription":
		return "/subscriptions/" + entityID
	case "installment":
		return "/installments/" + entityID
	case "debt":
		return "/debts/" + entityID
	default:
		return ""
	}
}

// dueReminderNotification renders a due-reminder in Indonesian with grouped
// rupiah. Example: "Langganan Netflix jatuh tempo dalam 3 hari — Rp 186.000."
func dueReminderNotification(item DueItem) Notification {
	label := entityLabelID(item.EntityType)
	name := html.EscapeString(item.Name)
	amount := money.GroupIDR(item.AmountMinor)
	title := fmt.Sprintf("%s %s akan jatuh tempo", label, name)
	message := fmt.Sprintf("%s \"%s\" jatuh tempo dalam %d hari (%s) — Rp %s.",
		label, name, item.DaysUntilDue, item.DueDate.Format("02 Jan 2006"), amount)

	// dedupe per entity + window + due-date so re-ticks don't re-notify, but a
	// new billing cycle (different due-date) does.
	dedupe := fmt.Sprintf("%s:%s:H-%d:%s", item.EntityType, item.EntityID, item.DaysUntilDue, item.DueDate.Format("2006-01-02"))

	return Notification{
		RuleKey:    RuleKeyDueReminder,
		DedupeKey:  dedupe,
		Subject:    fmt.Sprintf("Pengingat: %s %s jatuh tempo dalam %d hari", label, item.Name, item.DaysUntilDue),
		Title:      title,
		Message:    message,
		Severity:   "warning",
		ActionPath: entityActionPath(item.EntityType, item.EntityID),
	}
}

// weeklySummaryNotification renders the weekly cashflow summary in Indonesian.
// isoWeekKey is used for de-dupe so at most one summary is sent per ISO week.
func weeklySummaryNotification(summary CashflowSummary, isoWeekKey string) Notification {
	net := summary.NetMinor
	netLabel := "surplus"
	if net < 0 {
		netLabel = "defisit"
	}
	message := fmt.Sprintf(
		"Ringkasan minggu ini: pemasukan Rp %s, pengeluaran Rp %s, %s Rp %s.",
		money.GroupIDR(summary.IncomeMinor),
		money.GroupIDR(summary.ExpenseMinor),
		netLabel,
		money.GroupIDR(absInt64(net)),
	)
	return Notification{
		RuleKey:    RuleKeyWeeklySummary,
		DedupeKey:  "weekly-summary:" + isoWeekKey,
		Subject:    "Ringkasan Keuangan Mingguan Affluena",
		Title:      "Ringkasan keuangan mingguan",
		Message:    message,
		Severity:   "info",
		ActionPath: "/reports/cashflow",
	}
}

func absInt64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// buildHTMLBody wraps a plain-text message in a minimal HTML email body. Title
// and message are escaped to avoid injecting markup from entity names.
func buildHTMLBody(title, message string) string {
	safeTitle := html.EscapeString(title)
	safeMessage := html.EscapeString(message)
	return `<html><body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">` +
		`<div style="max-width: 600px; margin: 0 auto; padding: 20px;">` +
		`<h2 style="color: #333;">` + safeTitle + `</h2>` +
		`<p>` + safeMessage + `</p>` +
		`<hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">` +
		`<p style="font-size: 12px; color: #888;">Email otomatis dari Affluena. Mohon tidak membalas email ini.</p>` +
		`</div></body></html>`
}
