package transaction

import (
	"testing"
	"time"
)

func mustTestTime(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("invalid test time %q: %v", value, err)
	}
	return parsed
}
