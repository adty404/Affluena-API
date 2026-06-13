package auth

import (
	"context"
	"testing"
	"time"
)

func TestHashPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if !CheckPassword(hash, "correct horse battery staple") {
		t.Fatal("expected password check to accept the original password")
	}

	if CheckPassword(hash, "wrong password") {
		t.Fatal("expected password check to reject a different password")
	}
}

func TestNormalizeEmail(t *testing.T) {
	got := NormalizeEmail("  USER@Example.COM ")
	if got != "user@example.com" {
		t.Fatalf("expected normalized email user@example.com, got %q", got)
	}
}

type fakeAuthRepository struct {
	createdUser User
}

func (f *fakeAuthRepository) CreateUser(ctx context.Context, email string, passwordHash string) (User, error) {
	f.createdUser = User{ID: "user-1", Email: email, PasswordHash: passwordHash}
	return f.createdUser, nil
}

func (f *fakeAuthRepository) UserByEmail(ctx context.Context, email string) (User, error) {
	return f.createdUser, nil
}

func (f *fakeAuthRepository) UserByID(ctx context.Context, userID string) (User, error) {
	return f.createdUser, nil
}

func (f *fakeAuthRepository) StoreRefreshToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error {
	return nil
}

func (f *fakeAuthRepository) RefreshTokenUser(ctx context.Context, tokenHash string, now time.Time) (User, error) {
	return f.createdUser, nil
}

func (f *fakeAuthRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	return nil
}

func TestServiceUsesRepositoryPort(t *testing.T) {
	service := NewService(&fakeAuthRepository{}, NewTokenManager("secret", time.Minute, time.Hour))

	user, tokens, err := service.Register(context.Background(), "USER@Example.COM", "password123")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if user.Email != "user@example.com" {
		t.Fatalf("expected normalized email, got %q", user.Email)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("expected issued token pair")
	}
}
