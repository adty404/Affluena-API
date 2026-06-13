package auth

import (
	"context"
	"errors"
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
	err         error
	storeErr    error
	revokeErr   error
	storedHash  string
	revokedHash string
}

func (f *fakeAuthRepository) CreateUser(ctx context.Context, email string, passwordHash string) (User, error) {
	if f.err != nil {
		return User{}, f.err
	}
	f.createdUser = User{ID: "user-1", Email: email, PasswordHash: passwordHash}
	return f.createdUser, nil
}

func (f *fakeAuthRepository) UserByEmail(ctx context.Context, email string) (User, error) {
	if f.err != nil {
		return User{}, f.err
	}
	return f.createdUser, nil
}

func (f *fakeAuthRepository) UserByID(ctx context.Context, userID string) (User, error) {
	if f.err != nil {
		return User{}, f.err
	}
	return f.createdUser, nil
}

func (f *fakeAuthRepository) StoreRefreshToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error {
	f.storedHash = tokenHash
	return f.storeErr
}

func (f *fakeAuthRepository) RefreshTokenUser(ctx context.Context, tokenHash string, now time.Time) (User, error) {
	if f.err != nil {
		return User{}, f.err
	}
	return f.createdUser, nil
}

func (f *fakeAuthRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	f.revokedHash = tokenHash
	return f.revokeErr
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

func TestRegisterRejectsInvalidCredentialsAndPropagatesStorageError(t *testing.T) {
	service := NewService(&fakeAuthRepository{}, NewTokenManager("secret", time.Minute, time.Hour))

	if _, _, err := service.Register(context.Background(), "", "password123"); err == nil {
		t.Fatal("expected blank email to fail")
	}
	if _, _, err := service.Register(context.Background(), "user@example.com", "short"); err == nil {
		t.Fatal("expected short password to fail")
	}

	storeErr := errors.New("store failed")
	service = NewService(&fakeAuthRepository{storeErr: storeErr}, NewTokenManager("secret", time.Minute, time.Hour))
	if _, _, err := service.Register(context.Background(), "user@example.com", "password123"); !errors.Is(err, storeErr) {
		t.Fatalf("expected store error, got %v", err)
	}
}

func TestLoginRejectsMissingUserAndWrongPassword(t *testing.T) {
	service := NewService(&fakeAuthRepository{err: errors.New("not found")}, NewTokenManager("secret", time.Minute, time.Hour))
	if _, _, err := service.Login(context.Background(), "user@example.com", "password123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials for missing user, got %v", err)
	}

	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	service = NewService(&fakeAuthRepository{createdUser: User{ID: "user-1", Email: "user@example.com", PasswordHash: hash}}, NewTokenManager("secret", time.Minute, time.Hour))
	if _, _, err := service.Login(context.Background(), "user@example.com", "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials for wrong password, got %v", err)
	}
}

func TestRefreshRevokesOldTokenBeforeIssuingNewPair(t *testing.T) {
	repo := &fakeAuthRepository{createdUser: User{ID: "user-1", Email: "user@example.com"}}
	service := NewService(repo, NewTokenManager("secret", time.Minute, time.Hour))

	_, tokens, err := service.Refresh(context.Background(), "refresh-token")
	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("expected refreshed token pair")
	}
	if repo.revokedHash != HashRefreshToken("refresh-token") {
		t.Fatalf("expected old refresh token hash to be revoked, got %q", repo.revokedHash)
	}
	if repo.storedHash == "" || repo.storedHash == repo.revokedHash {
		t.Fatalf("expected new refresh token hash to be stored, got stored=%q revoked=%q", repo.storedHash, repo.revokedHash)
	}

	service = NewService(&fakeAuthRepository{err: errors.New("invalid")}, NewTokenManager("secret", time.Minute, time.Hour))
	if _, _, err := service.Refresh(context.Background(), "bad-token"); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected invalid refresh token, got %v", err)
	}
}
