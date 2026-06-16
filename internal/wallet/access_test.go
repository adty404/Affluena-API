package wallet

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockAccessChecker struct {
	level AccessLevel
	err   error
}

func (m *mockAccessChecker) GetAccessLevel(ctx context.Context, userID string, walletID string) (AccessLevel, error) {
	return m.level, m.err
}

func TestCheckOwnerAccess(t *testing.T) {
	tests := []struct {
		name    string
		level   AccessLevel
		wantErr bool
	}{
		{"Owner Allowed", AccessOwner, false},
		{"Member Denied", AccessMember, true},
		{"None Denied", AccessNone, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &mockAccessChecker{level: tt.level}
			err := CheckOwnerAccess(checker, context.Background(), "user", "wallet")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, ErrAccessDenied, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckMemberAccess(t *testing.T) {
	tests := []struct {
		name    string
		level   AccessLevel
		wantErr bool
	}{
		{"Owner Allowed", AccessOwner, false},
		{"Member Allowed", AccessMember, false},
		{"None Denied", AccessNone, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &mockAccessChecker{level: tt.level}
			err := CheckMemberAccess(checker, context.Background(), "user", "wallet")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, ErrAccessDenied, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
