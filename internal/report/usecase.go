package report

import (
	"context"
	"fmt"
	"time"
)

type UseCase struct {
	repo *Repository
}

func NewUseCase(repo *Repository) *UseCase {
	return &UseCase{repo: repo}
}

func parseMonth(monthStr string) (time.Time, time.Time, time.Time, time.Time, error) {
	var currentMonth time.Time
	var err error
	if monthStr == "" {
		now := time.Now()
		currentMonth = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		currentMonth, err = time.Parse("2006-01", monthStr)
		if err != nil {
			return time.Time{}, time.Time{}, time.Time{}, time.Time{}, fmt.Errorf("invalid month format, expected YYYY-MM")
		}
	}

	currentMonthStart := currentMonth
	currentMonthEnd := currentMonthStart.AddDate(0, 1, 0)

	prevMonthStart := currentMonthStart.AddDate(0, -1, 0)
	prevMonthEnd := currentMonthStart

	return currentMonthStart, currentMonthEnd, prevMonthStart, prevMonthEnd, nil
}

func calculateChangePercent(current, previous int64) float64 {
	if previous == 0 {
		return 0
	}
	return float64(current-previous) / float64(previous) * 100
}

func (uc *UseCase) IncomeReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error) {
	currStart, currEnd, prevStart, prevEnd, err := parseMonth(monthStr)
	if err != nil {
		return ReportResponse{}, err
	}

	currAgg, err := uc.repo.GetIncomeExpenseAggregation(ctx, userID, "income", currStart, currEnd)
	if err != nil {
		return ReportResponse{}, err
	}

	prevAgg, err := uc.repo.GetIncomeExpenseAggregation(ctx, userID, "income", prevStart, prevEnd)
	if err != nil {
		return ReportResponse{}, err
	}

	prevMap := make(map[string]int64)
	for _, p := range prevAgg {
		prevMap[p.CategoryName] = p.AmountMinor
	}

	var totalIncome int64
	var topSource string
	var maxAmount int64

	rows := make([]ReportRow, 0)
	for i, c := range currAgg {
		totalIncome += c.AmountMinor
		if c.AmountMinor > maxAmount {
			maxAmount = c.AmountMinor
			topSource = c.CategoryName
		}

		prevAmount := prevMap[c.CategoryName]
		change := calculateChangePercent(c.AmountMinor, prevAmount)

		status := "healthy"
		if change > 5 {
			status = "growth"
		} else if change < -5 {
			status = "watch"
		}

		rows = append(rows, ReportRow{
			ID:                  fmt.Sprintf("inc-%d", i),
			Name:                c.CategoryName,
			Category:            "Income category",
			AmountMinor:         c.AmountMinor,
			PreviousAmountMinor: prevAmount,
			ChangePercent:       change,
			Wallet:              c.WalletName,
			Status:              status,
		})
	}

	sourceCount := len(currAgg)
	var avgPerSource int64
	if sourceCount > 0 {
		avgPerSource = totalIncome / int64(sourceCount)
	}

	metrics := []ReportMetric{
		{ID: "total_income", Label: "Total Income", ValueMinor: totalIncome, Helper: "For the month", Tone: "positive"},
		{ID: "source_count", Label: "Income Sources", ValueMinor: int64(sourceCount), Helper: "Distinct categories", Tone: "neutral"},
		{ID: "avg_per_source", Label: "Avg per Source", ValueMinor: avgPerSource, Helper: "Average income per category", Tone: "neutral"},
		{ID: "top_source", Label: "Top Source", ValueMinor: 0, Helper: topSource, Tone: "positive"},
	}

	return ReportResponse{Metrics: metrics, Rows: rows}, nil
}

func (uc *UseCase) ExpenseReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error) {
	currStart, currEnd, prevStart, prevEnd, err := parseMonth(monthStr)
	if err != nil {
		return ReportResponse{}, err
	}

	currAgg, err := uc.repo.GetIncomeExpenseAggregation(ctx, userID, "expense", currStart, currEnd)
	if err != nil {
		return ReportResponse{}, err
	}

	prevAgg, err := uc.repo.GetIncomeExpenseAggregation(ctx, userID, "expense", prevStart, prevEnd)
	if err != nil {
		return ReportResponse{}, err
	}

	prevMap := make(map[string]int64)
	for _, p := range prevAgg {
		prevMap[p.CategoryName] = p.AmountMinor
	}

	var totalExpense int64
	var topCategory string
	var maxAmount int64

	rows := make([]ReportRow, 0)
	for i, c := range currAgg {
		totalExpense += c.AmountMinor
		if c.AmountMinor > maxAmount {
			maxAmount = c.AmountMinor
			topCategory = c.CategoryName
		}

		prevAmount := prevMap[c.CategoryName]
		change := calculateChangePercent(c.AmountMinor, prevAmount)

		status := "healthy"
		if change > 5 {
			status = "watch" // Expense increasing is watch
		} else if change < -5 {
			status = "growth" // Expense decreasing is growth
		}

		rows = append(rows, ReportRow{
			ID:                  fmt.Sprintf("exp-%d", i),
			Name:                c.CategoryName,
			Category:            "Expense category",
			AmountMinor:         c.AmountMinor,
			PreviousAmountMinor: prevAmount,
			ChangePercent:       change,
			Wallet:              c.WalletName,
			Status:              status,
		})
	}

	catCount := len(currAgg)
	var avgPerCat int64
	if catCount > 0 {
		avgPerCat = totalExpense / int64(catCount)
	}

	metrics := []ReportMetric{
		{ID: "total_expense", Label: "Total Expense", ValueMinor: totalExpense, Helper: "For the month", Tone: "negative"},
		{ID: "cat_count", Label: "Expense Categories", ValueMinor: int64(catCount), Helper: "Distinct categories", Tone: "neutral"},
		{ID: "avg_per_cat", Label: "Avg per Category", ValueMinor: avgPerCat, Helper: "Average expense per category", Tone: "neutral"},
		{ID: "top_category", Label: "Top Category", ValueMinor: 0, Helper: topCategory, Tone: "negative"},
	}

	return ReportResponse{Metrics: metrics, Rows: rows}, nil
}

func (uc *UseCase) CashflowReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error) {
	currStart, currEnd, prevStart, prevEnd, err := parseMonth(monthStr)
	if err != nil {
		return ReportResponse{}, err
	}

	currAgg, err := uc.repo.GetWeeklyCashflow(ctx, userID, currStart, currEnd)
	if err != nil {
		return ReportResponse{}, err
	}

	prevAgg, err := uc.repo.GetWeeklyCashflow(ctx, userID, prevStart, prevEnd)
	if err != nil {
		return ReportResponse{}, err
	}

	// Map previous weeks by relative week number (1-4)
	prevMap := make(map[int]int64)
	for _, p := range prevAgg {
		// Approximate relative week by subtracting the minimum week number
		// This is a simplification, a better way is to group by week of month
		// Let's just use the week number modulo 4 or 5
		relWeek := p.Week % 5
		prevMap[relWeek] = p.IncomeMinor - p.ExpenseMinor
	}

	var totalIncome, totalExpense int64
	rows := make([]ReportRow, 0)

	// Group current by relative week
	currMap := make(map[int]int64)
	for _, c := range currAgg {
		totalIncome += c.IncomeMinor
		totalExpense += c.ExpenseMinor
		relWeek := c.Week % 5
		currMap[relWeek] += (c.IncomeMinor - c.ExpenseMinor)
	}

	// Create 4 rows for weeks
	for i := 1; i <= 4; i++ {
		net := currMap[i]
		prevNet := prevMap[i]
		change := calculateChangePercent(net, prevNet)

		status := "healthy"
		if change > 5 {
			status = "growth"
		} else if change < -5 {
			status = "watch"
		}

		rows = append(rows, ReportRow{
			ID:                  fmt.Sprintf("week-%d", i),
			Name:                fmt.Sprintf("Week %d Cashflow", i),
			Category:            "Weekly summary",
			AmountMinor:         net,
			PreviousAmountMinor: prevNet,
			ChangePercent:       change,
			Wallet:              "All wallets",
			Status:              status,
		})
	}

	netCashflow := totalIncome - totalExpense
	var savingRate int64
	if totalIncome > 0 {
		savingRate = (netCashflow * 100) / totalIncome
	}

	metrics := []ReportMetric{
		{ID: "net_cashflow", Label: "Net Cashflow", ValueMinor: netCashflow, Helper: "Income - Expense", Tone: "positive"},
		{ID: "total_income", Label: "Income Total", ValueMinor: totalIncome, Helper: "For the month", Tone: "positive"},
		{ID: "total_expense", Label: "Expense Total", ValueMinor: totalExpense, Helper: "For the month", Tone: "negative"},
		{ID: "saving_rate", Label: "Saving Rate", ValueMinor: savingRate, Helper: "% of income saved", Tone: "positive"},
	}

	return ReportResponse{Metrics: metrics, Rows: rows}, nil
}

func (uc *UseCase) DebtReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error) {
	debts, err := uc.repo.GetDebts(ctx, userID)
	if err != nil {
		return ReportResponse{}, err
	}

	var totalPayable, totalReceivable int64
	var openCount, overdueCount int

	rows := make([]ReportRow, 0)
	for _, d := range debts {
		remaining := d.PrincipalAmountMinor - d.PaidAmountMinor

		if d.Type == "payable" {
			totalPayable += remaining
		} else {
			totalReceivable += remaining
		}

		if d.Status != "closed" {
			openCount++
		}
		if d.Status == "overdue" {
			overdueCount++
		}

		change := calculateChangePercent(remaining, d.PrincipalAmountMinor)

		status := "healthy"
		if d.Status == "overdue" {
			status = "critical"
		} else if d.Status == "open" || d.Status == "partial" {
			status = "watch"
		}

		cat := "Payable"
		if d.Type == "receivable" {
			cat = "Receivable"
		}

		rows = append(rows, ReportRow{
			ID:                  d.ID,
			Name:                d.CounterpartyName,
			Category:            cat,
			AmountMinor:         remaining,
			PreviousAmountMinor: d.PrincipalAmountMinor,
			ChangePercent:       change,
			Wallet:              d.WalletName,
			Status:              status,
		})
	}

	metrics := []ReportMetric{
		{ID: "total_payable", Label: "Total Payable", ValueMinor: totalPayable, Helper: "Remaining to pay", Tone: "negative"},
		{ID: "total_receivable", Label: "Total Receivable", ValueMinor: totalReceivable, Helper: "Remaining to receive", Tone: "positive"},
		{ID: "open_count", Label: "Open Debts", ValueMinor: int64(openCount), Helper: "Active debts", Tone: "neutral"},
		{ID: "overdue_count", Label: "Overdue", ValueMinor: int64(overdueCount), Helper: "Past due date", Tone: "negative"},
	}

	return ReportResponse{Metrics: metrics, Rows: rows}, nil
}

func (uc *UseCase) GoalReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error) {
	goals, err := uc.repo.GetGoals(ctx, userID)
	if err != nil {
		return ReportResponse{}, err
	}

	var totalSaved, totalTarget int64
	var activeCount int

	rows := make([]ReportRow, 0)
	for _, g := range goals {
		totalSaved += g.BalanceMinor
		totalTarget += g.TargetAmountMinor

		if g.Status == "active" {
			activeCount++
		}

		// We don't have historical goal balances easily queryable without a complex snapshot table
		// So we'll just use 0 for previous and 0 for change
		change := 0.0

		status := "healthy"
		if g.BalanceMinor > 0 {
			status = "growth"
		}

		rows = append(rows, ReportRow{
			ID:                  g.ID,
			Name:                g.Name,
			Category:            "Goal",
			AmountMinor:         g.BalanceMinor,
			PreviousAmountMinor: 0,
			ChangePercent:       change,
			Wallet:              g.WalletName,
			Status:              status,
		})
	}

	var progress int64
	if totalTarget > 0 {
		progress = (totalSaved * 100) / totalTarget
	}

	metrics := []ReportMetric{
		{ID: "total_saved", Label: "Total Saved", ValueMinor: totalSaved, Helper: "Across all goals", Tone: "positive"},
		{ID: "total_target", Label: "Total Target", ValueMinor: totalTarget, Helper: "Sum of all targets", Tone: "neutral"},
		{ID: "overall_progress", Label: "Overall Progress %", ValueMinor: progress, Helper: "Percentage reached", Tone: "positive"},
		{ID: "active_count", Label: "Active Goals", ValueMinor: int64(activeCount), Helper: "Currently active", Tone: "neutral"},
	}

	return ReportResponse{Metrics: metrics, Rows: rows}, nil
}

func (uc *UseCase) OverviewReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error) {
	// Reuse other reports to build overview
	cashflow, err := uc.CashflowReport(ctx, userID, monthStr)
	if err != nil {
		return ReportResponse{}, err
	}

	income, err := uc.IncomeReport(ctx, userID, monthStr)
	if err != nil {
		return ReportResponse{}, err
	}

	expense, err := uc.ExpenseReport(ctx, userID, monthStr)
	if err != nil {
		return ReportResponse{}, err
	}

	metrics := cashflow.Metrics // Net Cashflow, Total Income, Total Expense, Saving Rate

	rows := make([]ReportRow, 0)

	// Top Income
	if len(income.Rows) > 0 {
		rows = append(rows, income.Rows[0])
	}

	// Top Expense
	if len(expense.Rows) > 0 {
		rows = append(rows, expense.Rows[0])
	}

	return ReportResponse{Metrics: metrics, Rows: rows}, nil
}
