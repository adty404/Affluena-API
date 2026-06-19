package report

import "errors"

var (
	ErrNotFound = errors.New("report data not found")
)

type ReportMetric struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	ValueMinor int64  `json:"value_minor"`
	Helper     string `json:"helper"`
	Tone       string `json:"tone"`
}

type ReportRow struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	Category            string  `json:"category"`
	AmountMinor         int64   `json:"amount_minor"`
	PreviousAmountMinor int64   `json:"previous_amount_minor"`
	ChangePercent       float64 `json:"change_percent"`
	Wallet              string  `json:"wallet"`
	Status              string  `json:"status"`
}

type ReportResponse struct {
	Metrics []ReportMetric `json:"metrics"`
	Rows    []ReportRow    `json:"rows"`
}
