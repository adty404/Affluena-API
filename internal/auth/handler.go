package auth

import (
	"context"
	"errors"
	"net/http"

	"affluena-api/internal/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service authUseCase
}

type authUseCase interface {
	Register(ctx context.Context, email string, password string) (User, TokenPair, error)
	Login(ctx context.Context, email string, password string) (User, TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (User, TokenPair, error)
	User(ctx context.Context, userID string) (User, error)
	UpdateAccount(ctx context.Context, userID string, name string, avatarURL string) (User, error)
	ChangePassword(ctx context.Context, userID string, currentPassword string, newPassword string) error
	ListSessions(ctx context.Context, userID string) ([]Session, error)
	RevokeSession(ctx context.Context, userID string, sessionID string) error
}

func NewHandler(service authUseCase) *Handler {
	return &Handler{service: service}
}

type authRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type authResponse struct {
	User   User      `json:"user"`
	Tokens TokenPair `json:"tokens"`
}

func (h *Handler) Register(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	user, tokens, err := h.service.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.JSON(c, http.StatusCreated, authResponse{User: user, Tokens: tokens})
}

func (h *Handler) Login(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	user, tokens, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Error(c, http.StatusUnauthorized, "invalid email or password")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "login failed")
		return
	}

	httpx.JSON(c, http.StatusOK, authResponse{User: user, Tokens: tokens})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	user, tokens, err := h.service.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httpx.Error(c, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	httpx.JSON(c, http.StatusOK, authResponse{User: user, Tokens: tokens})
}

func (h *Handler) Me(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	user, err := h.service.User(c.Request.Context(), userID)
	if err != nil {
		httpx.Error(c, http.StatusNotFound, "user not found")
		return
	}

	httpx.JSON(c, http.StatusOK, gin.H{"user": user})
}

type updateAccountRequest struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

func (h *Handler) UpdateAccount(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	var req updateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.UpdateAccount(c.Request.Context(), userID, req.Name, req.AvatarURL)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "update account failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"user": user})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

func (h *Handler) ChangePassword(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.service.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Error(c, http.StatusUnauthorized, "current password is incorrect")
			return
		}
		if errors.Is(err, ErrPasswordTooWeak) {
			httpx.Error(c, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "change password failed")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListSessions(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	sessions, err := h.service.ListSessions(c.Request.Context(), userID)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list sessions failed")
		return
	}
	if sessions == nil {
		sessions = []Session{}
	}
	httpx.JSON(c, http.StatusOK, gin.H{"sessions": sessions})
}

func (h *Handler) RevokeSession(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	sessionID, ok := httpx.GetUUIDParam(c, "session_id")
	if !ok {
		return
	}

	err := h.service.RevokeSession(c.Request.Context(), userID, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			httpx.Error(c, http.StatusNotFound, "session not found")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "revoke session failed")
		return
	}
	c.Status(http.StatusNoContent)
}
