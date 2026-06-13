package apilog

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	SaveLog(ctx context.Context, logEntry APILog) error
}

type repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{pool: pool}
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
