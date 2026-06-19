package apilog

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("api log not found")

type APILog struct {
	ID              string    `json:"id"`
	Method          string    `json:"method"`
	Path            string    `json:"path"`
	StatusCode      int       `json:"status_code"`
	LatencyMs       int       `json:"latency_ms"`
	ClientIP        string    `json:"client_ip"`
	UserAgent       string    `json:"user_agent"`
	UserID          *string   `json:"user_id"` // Optional
	RequestPayload  *string   `json:"request_payload"`
	ResponsePayload *string   `json:"response_payload"`
	CreatedAt       time.Time `json:"created_at"`
}
