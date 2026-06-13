package auth

import (
	"context"
	"errors"
	"net/http"

	"affluena/internal/httpx"

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
		httpx.Error(c, http.StatusBadRequest, err.Error())
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
