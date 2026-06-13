package export

import (
	"context"
	"fmt"
)

type ExportRepository interface {
	GetCSVRows(ctx context.Context, userID string, opts ExportOptions) ([]TransactionExportRow, error)
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
			row.Note,
			row.WalletName,
			row.ToWalletName,
			row.CategoryName,
			row.Tags,
			row.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return csvData, nil
}
