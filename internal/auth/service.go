package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"affluena-api/internal/activity"
)

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

type Service struct {
	repo       RepositoryPort
	tokens     *TokenManager
	activityUC activity.UseCase
}

type RepositoryPort interface {
	CreateUser(ctx context.Context, email string, passwordHash string) (User, error)
	UserByEmail(ctx context.Context, email string) (User, error)
	UserByID(ctx context.Context, userID string) (User, error)
	StoreRefreshToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error
	RefreshTokenUser(ctx context.Context, tokenHash string, now time.Time) (User, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	ConsumeRefreshToken(ctx context.Context, tokenHash string, now time.Time) (string, error)
}

func NewService(repo RepositoryPort, tokens *TokenManager, activityUC activity.UseCase) *Service {
	return &Service{repo: repo, tokens: tokens, activityUC: activityUC}
}

func (s *Service) Register(ctx context.Context, email string, password string) (User, TokenPair, error) {
	email = NormalizeEmail(email)
	if email == "" || len(password) < 8 {
		return User{}, TokenPair{}, errors.New("email and password of at least 8 characters are required")
	}

	hash, err := HashPassword(password)
	if err != nil {
		return User{}, TokenPair{}, err
	}

	user, err := s.repo.CreateUser(ctx, email, hash)
	if err != nil {
		return User{}, TokenPair{}, err
	}

	pair, err := s.issuePair(ctx, user)
	if err != nil {
		return User{}, TokenPair{}, err
	}

	if s.activityUC != nil {
		s.activityUC.LogActivity(ctx, user.ID, "REGISTER", "AUTH", nil, "Mendaftar akun baru")
	}

	return user, pair, nil
}

func (s *Service) Login(ctx context.Context, email string, password string) (User, TokenPair, error) {
	user, err := s.repo.UserByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		return User{}, TokenPair{}, ErrInvalidCredentials
	}
	if !CheckPassword(user.PasswordHash, password) {
		return User{}, TokenPair{}, ErrInvalidCredentials
	}

	pair, err := s.issuePair(ctx, user)
	if err != nil {
		return User{}, TokenPair{}, err
	}

	if s.activityUC != nil {
		s.activityUC.LogActivity(ctx, user.ID, "LOGIN", "AUTH", nil, "Berhasil masuk ke aplikasi")
	}

	return user, pair, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (User, TokenPair, error) {
	hash := HashRefreshToken(refreshToken)
	userID, err := s.repo.ConsumeRefreshToken(ctx, hash, time.Now().UTC())
	if err != nil {
		return User{}, TokenPair{}, ErrInvalidRefreshToken
	}

	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return User{}, TokenPair{}, err
	}

	pair, err := s.issuePair(ctx, user)
	if err != nil {
		return User{}, TokenPair{}, err
	}
	return user, pair, nil
}

func (s *Service) User(ctx context.Context, userID string) (User, error) {
	return s.repo.UserByID(ctx, userID)
}

func (s *Service) issuePair(ctx context.Context, user User) (TokenPair, error) {
	now := time.Now().UTC()
	accessToken, accessExpiresAt, err := s.tokens.IssueAccessToken(user, now)
	if err != nil {
		return TokenPair{}, err
	}
	refreshToken, refreshHash, refreshExpiresAt, err := s.tokens.IssueRefreshToken(now)
	if err != nil {
		return TokenPair{}, err
	}
	if err := s.repo.StoreRefreshToken(ctx, user.ID, refreshHash, refreshExpiresAt); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshTokenExpiresAt: refreshExpiresAt,
	}, nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash string, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
