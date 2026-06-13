package apilog

import (
	"time"
)

type APILog struct {
	ID         string    `json:"id"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	LatencyMs  int       `json:"latency_ms"`
	ClientIP   string    `json:"client_ip"`
	UserAgent  string    `json:"user_agent"`
	UserID     *string   `json:"user_id"` // Optional
	CreatedAt  time.Time `json:"created_at"`
}
