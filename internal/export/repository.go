package export

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func nullableTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}

func (r *Repository) GetCSVRows(ctx context.Context, userID string, opts ExportOptions) ([]TransactionExportRow, error) {
	query := `
		SELECT 
			t.id::text, 
			t.type,
			t.amount_minor,
			t.transaction_at,
			t.note,
			COALESCE(w.name, ''),
			COALESCE(tw.name, ''),
			COALESCE(c.name, ''),
			COALESCE((
				SELECT string_agg(tags.name, ', ')
				FROM transaction_tags tt
				JOIN tags ON tags.id = tt.tag_id
				WHERE tt.transaction_id = t.id AND tt.user_id = $1
			), ''),
			t.created_at
		FROM transactions t
		LEFT JOIN wallets w ON t.wallet_id = w.id
		LEFT JOIN wallets tw ON t.to_wallet_id = tw.id
		LEFT JOIN categories c ON t.category_id = c.id
		WHERE t.user_id = $1 
		  AND ($2::timestamptz IS NULL OR t.transaction_at >= $2)
		  AND ($3::timestamptz IS NULL OR t.transaction_at < $3)
		ORDER BY t.transaction_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID, nullableTime(opts.From), nullableTime(opts.To))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TransactionExportRow
	for rows.Next() {
		var row TransactionExportRow
		if err := rows.Scan(
			&row.ID,
			&row.Type,
			&row.AmountMinor,
			&row.TransactionAt,
			&row.Note,
			&row.WalletName,
			&row.ToWalletName,
			&row.CategoryName,
			&row.Tags,
			&row.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
