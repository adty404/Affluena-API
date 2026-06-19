package alert

import (
	"errors"
	"time"
)

var ErrAlertNotFound = errors.New("alert not found")

type Alert struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Module      string    `json:"module"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	CreatedAt   time.Time `json:"created_at"`
	ActionPath  string    `json:"action_path"`
}
