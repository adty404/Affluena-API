package export

import (
	"context"
	"fmt"
	"time"
)

type ExportRepository interface {
	GetCSVRows(ctx context.Context, userID string, opts ExportOptions) ([]TransactionExportRow, error)
	CreateJob(ctx context.Context, userID string, format string, fromAt *time.Time, toAt *time.Time, rowCount int, status string) (ExportJob, error)
	ListJobs(ctx context.Context, userID string, limit, offset int) ([]ExportJob, int, error)
	GetJob(ctx context.Context, userID, id string) (ExportJob, error)
}

type UseCase struct {
	repo ExportRepository
}

func NewUseCase(repo ExportRepository) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) GenerateCSVData(ctx context.Context, userID string, opts ExportOptions) ([][]string, error) {
	rows, err := u.repo.GetCSVRows(ctx, userID, opts)
	if err != nil {
		return nil, err
	}

	var csvData [][]string
	// Header
	csvData = append(csvData, []string{
		"ID",
		"Type",
		"Amount",
		"Transaction Date",
		"Note",
		"Wallet",
		"To Wallet",
		"Category",
		"Tags",
		"Created At",
	})

	for _, row := range rows {
		csvData = append(csvData, []string{
			row.ID,
			row.Type,
			fmt.Sprintf("%d", row.AmountMinor),
			row.TransactionAt.Format("2006-01-02T15:04:05Z07:00"), // RFC3339
			sanitizeCSVCell(row.Note),
			sanitizeCSVCell(row.WalletName),
			sanitizeCSVCell(row.ToWalletName),
			sanitizeCSVCell(row.CategoryName),
			sanitizeCSVCell(row.Tags),
			row.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return csvData, nil
}

// sanitizeCSVCell defuses CSV formula injection: spreadsheet apps execute a cell
// that begins with =, +, -, @, or a leading tab/CR. User-controlled text is
// prefixed with a single quote so it is treated as literal text, not a formula.
func sanitizeCSVCell(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

func (u *UseCase) RecordJob(ctx context.Context, userID string, format string, opts ExportOptions, rowCount int, status string) (ExportJob, error) {
	var fromAt, toAt *time.Time
	if !opts.From.IsZero() {
		fromAt = &opts.From
	}
	if !opts.To.IsZero() {
		toAt = &opts.To
	}
	return u.repo.CreateJob(ctx, userID, format, fromAt, toAt, rowCount, status)
}

func (u *UseCase) ListJobs(ctx context.Context, userID string, limit, offset int) ([]ExportJob, int, error) {
	return u.repo.ListJobs(ctx, userID, limit, offset)
}

func (u *UseCase) GetJob(ctx context.Context, userID, id string) (ExportJob, error) {
	return u.repo.GetJob(ctx, userID, id)
}
