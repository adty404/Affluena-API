package recurring

import (
	"errors"
	"time"

	"affluena/internal/caldate"
)

type Frequency string

const (
	FrequencyWeekly  Frequency = "weekly"
	FrequencyMonthly Frequency = "monthly"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusPaused    Status = "paused"
	StatusCancelled Status = "cancelled"
)

type RunType string

const (
	RunTypeManual    RunType = "manual"
	RunTypeScheduled RunType = "scheduled"
)

func AdvanceNextRunAt(current time.Time, frequency Frequency, intervalCount int) (time.Time, error) {
	if intervalCount <= 0 {
		return time.Time{}, errors.New("interval_count must be positive")
	}

	switch frequency {
	case FrequencyWeekly:
		return current.AddDate(0, 0, 7*intervalCount), nil
	case FrequencyMonthly:
		return caldate.AddMonthsClamped(current, intervalCount), nil
	default:
		return time.Time{}, errors.New("invalid frequency")
	}
}

func AdvancePast(current time.Time, frequency Frequency, intervalCount int, now time.Time) (time.Time, error) {
	next := current
	for !next.After(now) {
		advanced, err := AdvanceNextRunAt(next, frequency, intervalCount)
		if err != nil {
			return time.Time{}, err
		}
		next = advanced
	}
	return next, nil
}

func NextStateAfterRun(current time.Time, frequency Frequency, intervalCount int, endAt *time.Time, currentStatus Status) (time.Time, Status, error) {
	nextRunAt, err := AdvanceNextRunAt(current, frequency, intervalCount)
	if err != nil {
		return time.Time{}, "", err
	}

	nextStatus := currentStatus
	if endAt != nil && nextRunAt.After((*endAt).UTC()) {
		nextStatus = StatusCancelled
	}
	return nextRunAt, nextStatus, nil
}

func IsValidFrequency(frequency Frequency) bool {
	switch frequency {
	case FrequencyWeekly, FrequencyMonthly:
		return true
	default:
		return false
	}
}

func IsValidStatus(status Status) bool {
	switch status {
	case StatusActive, StatusPaused, StatusCancelled:
		return true
	default:
		return false
	}
}
