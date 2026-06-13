package dashboard

import (
	"context"
	"net/http"
	"time"

	"affluena/internal/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase dashboardUseCase
}

type dashboardUseCase interface {
	Summary(ctx context.Context, userID string, month time.Time) (Summary, error)
}

func NewHandler(usecase dashboardUseCase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Summary(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	month, err := ParseMonth(c.Query("month"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	summary, err := h.usecase.Summary(c.Request.Context(), userID, month)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "dashboard summary failed")
		return
	}
	httpx.JSON(c, http.StatusOK, summary)
}
