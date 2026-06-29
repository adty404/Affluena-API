package partner

import (
	"context"
	"errors"
	"net/http"

	"affluena-api/internal/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase partnerUseCase
}

type partnerUseCase interface {
	Invite(ctx context.Context, userID string, input InviteInput) error
	Respond(ctx context.Context, userID, linkID string, input RespondInput) error
	List(ctx context.Context, userID string) ([]Link, error)
	Revoke(ctx context.Context, userID, linkID string) error
}

func NewHandler(usecase partnerUseCase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Invite(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	var req InviteInput
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.usecase.Invite(c.Request.Context(), userID, req); err != nil {
		switch {
		case errors.Is(err, ErrPartnerLimit):
			httpx.Error(c, http.StatusConflict, "You can share with at most 5 people. Remove someone first to add another.")
		case errors.Is(err, ErrAlreadyLinked):
			httpx.Error(c, http.StatusConflict, "That person can already see your wallets.")
		default:
			httpx.WriteError(c, err)
		}
		return
	}
	c.Status(http.StatusCreated)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	links, err := h.usecase.List(c.Request.Context(), userID)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list partners failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"partners": links})
}

func (h *Handler) Respond(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	var req RespondInput
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.usecase.Respond(c.Request.Context(), userID, id, req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) Revoke(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.usecase.Revoke(c.Request.Context(), userID, id); err != nil {
		httpx.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
