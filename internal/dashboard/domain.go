package dashboard

import "time"

type Summary struct {
	Month                 string                 `json:"month"`
	NetWorthMinor         int64                  `json:"net_worth_minor"`
	MonthlyIncomeMinor    int64                  `json:"monthly_income_minor"`
	MonthlyExpenseMinor   int64                  `json:"monthly_expense_minor"`
	MonthlyCashflowMinor  int64                  `json:"monthly_cashflow_minor"`
	Budget                BudgetSummary          `json:"budget"`
	UpcomingSubscriptions []UpcomingSubscription `json:"upcoming_subscriptions"`
	UpcomingInstallments  []UpcomingInstallment  `json:"upcoming_installments"`
	UpcomingDebts         []UpcomingDebt         `json:"upcoming_debts"`
}

type BudgetSummary struct {
	LimitMinor     int64   `json:"limit_minor"`
	SpentMinor     int64   `json:"spent_minor"`
	RemainingMinor int64   `json:"remaining_minor"`
	UsagePercent   float64 `json:"usage_percent"`
}

type UpcomingSubscription struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	AccountDetail string    `json:"account_detail"`
	AmountMinor   int64     `json:"amount_minor"`
	NextDueDate   time.Time `json:"next_due_date"`
}

type UpcomingInstallment struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	MonthlyAmountMinor int64     `json:"monthly_amount_minor"`
	RemainingMonths    int       `json:"remaining_months"`
	DueDate            time.Time `json:"due_date"`
}

type UpcomingDebt struct {
	ID                   string    `json:"id"`
	Type                 string    `json:"type"`
	CounterpartyName     string    `json:"counterparty_name"`
	RemainingAmountMinor int64     `json:"remaining_amount_minor"`
	DueDate              time.Time `json:"due_date"`
}
