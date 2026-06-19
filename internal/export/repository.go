package export

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
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
		WHERE (
			t.user_id = $1 OR
			EXISTS (SELECT 1 FROM wallets aw WHERE aw.id = t.wallet_id AND aw.user_id = $1) OR
			EXISTS (SELECT 1 FROM wallets atw WHERE atw.id = t.to_wallet_id AND atw.user_id = $1) OR
			t.wallet_id IN (SELECT wallet_id FROM wallet_shares WHERE user_id = $1 AND status = 'joined') OR
			t.to_wallet_id IN (SELECT wallet_id FROM wallet_shares WHERE user_id = $1 AND status = 'joined')
		)
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

func (r *Repository) CreateJob(ctx context.Context, userID string, format string, fromAt *time.Time, toAt *time.Time, rowCount int, status string) (ExportJob, error) {
	query := `
		INSERT INTO export_jobs (user_id, format, from_at, to_at, row_count, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, user_id::text, format, from_at, to_at, row_count, status, created_at
	`
	var job ExportJob
	err := r.pool.QueryRow(ctx, query, userID, format, fromAt, toAt, rowCount, status).Scan(
		&job.ID,
		&job.UserID,
		&job.Format,
		&job.FromAt,
		&job.ToAt,
		&job.RowCount,
		&job.Status,
		&job.CreatedAt,
	)
	return job, err
}

func (r *Repository) ListJobs(ctx context.Context, userID string, limit, offset int) ([]ExportJob, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, "SELECT count(*) FROM export_jobs WHERE user_id = $1", userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id::text, user_id::text, format, from_at, to_at, row_count, status, created_at
		FROM export_jobs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []ExportJob
	for rows.Next() {
		var job ExportJob
		if err := rows.Scan(
			&job.ID,
			&job.UserID,
			&job.Format,
			&job.FromAt,
			&job.ToAt,
			&job.RowCount,
			&job.Status,
			&job.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if jobs == nil {
		jobs = []ExportJob{}
	}

	return jobs, total, nil
}

func (r *Repository) GetJob(ctx context.Context, userID, id string) (ExportJob, error) {
	query := `
		SELECT id::text, user_id::text, format, from_at, to_at, row_count, status, created_at
		FROM export_jobs
		WHERE user_id = $1 AND id = $2
	`
	var job ExportJob
	err := r.pool.QueryRow(ctx, query, userID, id).Scan(
		&job.ID,
		&job.UserID,
		&job.Format,
		&job.FromAt,
		&job.ToAt,
		&job.RowCount,
		&job.Status,
		&job.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ExportJob{}, ErrJobNotFound
		}
		return ExportJob{}, err
	}
	return job, nil
}
