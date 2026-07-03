package apilog

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	SaveLog(ctx context.Context, logEntry APILog) error
	ListLogs(ctx context.Context, userID string, limit int) ([]APILog, error)
	GetLogByID(ctx context.Context, userID string, id string) (*APILog, error)
	// DeleteOlderThan prunes api_logs rows created before the cutoff and returns
	// the number of rows removed. Used by the retention job to bound table growth.
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
}

func (r *repository) ListLogs(ctx context.Context, userID string, limit int) ([]APILog, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT id, method, path, status_code, latency_ms, client_ip, user_agent, user_id, request_payload, response_payload, created_at
		FROM api_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []APILog
	for rows.Next() {
		var log APILog
		if err := rows.Scan(&log.ID, &log.Method, &log.Path, &log.StatusCode, &log.LatencyMs, &log.ClientIP, &log.UserAgent, &log.UserID, &log.RequestPayload, &log.ResponsePayload, &log.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (r *repository) SaveLog(ctx context.Context, logEntry APILog) error {
	query := `
		INSERT INTO api_logs (method, path, status_code, latency_ms, client_ip, user_agent, user_id, request_payload, response_payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		logEntry.Method,
		logEntry.Path,
		logEntry.StatusCode,
		logEntry.LatencyMs,
		logEntry.ClientIP,
		logEntry.UserAgent,
		logEntry.UserID,
		logEntry.RequestPayload,
		logEntry.ResponsePayload,
	)
	if err != nil {
		slog.Error("failed to save api log to database", "error", err)
		return err
	}
	return nil
}

// DeleteOlderThan removes api_logs rows older than cutoff. The query filters on
// created_at, which is backed by idx_api_logs_created_at (migration 000013).
func (r *repository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM api_logs WHERE created_at < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *repository) GetLogByID(ctx context.Context, userID string, id string) (*APILog, error) {
	query := `
		SELECT id, method, path, status_code, latency_ms, client_ip, user_agent, user_id, request_payload, response_payload, created_at
		FROM api_logs
		WHERE id = $1 AND user_id = $2
	`
	var log APILog
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&log.ID, &log.Method, &log.Path, &log.StatusCode, &log.LatencyMs,
		&log.ClientIP, &log.UserAgent, &log.UserID, &log.RequestPayload,
		&log.ResponsePayload, &log.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &log, nil
}
