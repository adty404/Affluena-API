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
	ErrInvalidResetToken   = errors.New("invalid or expired reset token")
	ErrSessionNotFound     = errors.New("session not found")
	ErrPasswordTooWeak     = errors.New("password must be at least 8 characters")
)

type MailerPort interface {
	Send(ctx context.Context, to string, subject string, htmlBody string) error
}

type Service struct {
	repo       RepositoryPort
	tokens     *TokenManager
	activityUC activity.UseCase
	mailer     MailerPort
	fromEmail  string
	appBaseURL string
}

type RepositoryPort interface {
	CreateUser(ctx context.Context, email string, passwordHash string) (User, error)
	UserByEmail(ctx context.Context, email string) (User, error)
	UserByID(ctx context.Context, userID string) (User, error)
	StoreRefreshToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error
	RefreshTokenUser(ctx context.Context, tokenHash string, now time.Time) (User, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	ConsumeRefreshToken(ctx context.Context, tokenHash string, now time.Time) (string, error)
	UpdateUserProfile(ctx context.Context, userID string, name string, avatarURL string) (User, error)
	ChangePassword(ctx context.Context, userID string, newPasswordHash string) error
	RevokeAllSessionsExcept(ctx context.Context, userID string, exceptTokenHash string) error
	ListSessions(ctx context.Context, userID string) ([]Session, error)
	RevokeSessionByID(ctx context.Context, userID string, sessionID string) error
	CreatePasswordResetToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error
	ConsumePasswordResetToken(ctx context.Context, tokenHash string, now time.Time) (string, error)
}

func NewService(repo RepositoryPort, tokens *TokenManager, activityUC activity.UseCase) *Service {
	return &Service{repo: repo, tokens: tokens, activityUC: activityUC}
}

func NewServiceWithMailer(repo RepositoryPort, tokens *TokenManager, activityUC activity.UseCase, mailer MailerPort, fromEmail string, appBaseURL string) *Service {
	return &Service{repo: repo, tokens: tokens, activityUC: activityUC, mailer: mailer, fromEmail: fromEmail, appBaseURL: appBaseURL}
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

func (s *Service) UpdateAccount(ctx context.Context, userID string, name string, avatarURL string) (User, error) {
	user, err := s.repo.UpdateUserProfile(ctx, userID, name, avatarURL)
	if err != nil {
		return User{}, err
	}
	if s.activityUC != nil {
		s.activityUC.LogActivity(ctx, userID, "UPDATE", "AUTH", nil, "Memperbarui profil akun")
	}
	return user, nil
}

func (s *Service) ChangePassword(ctx context.Context, userID string, currentPassword string, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrPasswordTooWeak
	}
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if !CheckPassword(user.PasswordHash, currentPassword) {
		return ErrInvalidCredentials
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.ChangePassword(ctx, userID, hash); err != nil {
		return err
	}
	if s.activityUC != nil {
		s.activityUC.LogActivity(ctx, userID, "UPDATE", "AUTH", nil, "Mengubah password akun")
	}
	return nil
}

func (s *Service) ListSessions(ctx context.Context, userID string) ([]Session, error) {
	return s.repo.ListSessions(ctx, userID)
}

func (s *Service) RevokeSession(ctx context.Context, userID string, sessionID string) error {
	err := s.repo.RevokeSessionByID(ctx, userID, sessionID)
	if err == nil && s.activityUC != nil {
		s.activityUC.LogActivity(ctx, userID, "REVOKE", "AUTH_SESSION", nil, "Mencabut sesi: "+sessionID)
	}
	return err
}

// RequestPasswordReset always returns nil error to avoid leaking whether the email exists.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	email = NormalizeEmail(email)
	user, err := s.repo.UserByEmail(ctx, email)
	if err != nil {
		return nil // swallow not-found to prevent email enumeration
	}

	token, hash, expiresAt, err := s.tokens.IssuePasswordResetToken(time.Now().UTC())
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordResetToken(ctx, user.ID, hash, expiresAt); err != nil {
		return err
	}

	if s.mailer != nil {
		resetURL := s.appBaseURL + "/reset-password?token=" + token
		html := "<p>Permintaan reset password diterima.</p><p>Klik link berikut untuk mengganti password: <a href=\"" + resetURL + "\">" + resetURL + "</a></p><p>Link berlaku 1 jam.</p>"
		_ = s.mailer.Send(ctx, user.Email, "Reset Password Affluena", html)
	}

	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token string, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrPasswordTooWeak
	}
	hash := HashRefreshToken(token)
	userID, err := s.repo.ConsumePasswordResetToken(ctx, hash, time.Now().UTC())
	if err != nil {
		return ErrInvalidResetToken
	}
	pwHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.ChangePassword(ctx, userID, pwHash); err != nil {
		return err
	}
	if s.activityUC != nil {
		s.activityUC.LogActivity(ctx, userID, "UPDATE", "AUTH", nil, "Reset password via email")
	}
	return nil
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
