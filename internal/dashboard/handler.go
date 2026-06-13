package dashboard

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"affluena-api/internal/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase dashboardUseCase
}

type dashboardUseCase interface {
	Summary(ctx context.Context, userID string, month time.Time) (Summary, error)
	CashflowTrend(ctx context.Context, userID string, months int) ([]CashflowTrend, error)
	ExpenseDistribution(ctx context.Context, userID string, month time.Time) ([]ExpenseDistribution, error)
	Forecast(ctx context.Context, userID string, month time.Time) (Forecast, error)
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

func (h *Handler) CashflowTrend(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	months := 6
	if m := c.Query("months"); m != "" {
		parsed, err := strconv.Atoi(m)
		if err != nil || parsed <= 0 || parsed > 12 {
			httpx.Error(c, http.StatusBadRequest, "months must be between 1 and 12")
			return
		}
		months = parsed
	}

	trend, err := h.usecase.CashflowTrend(c.Request.Context(), userID, months)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "failed to get cashflow trend")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"trend": trend})
}

func (h *Handler) ExpenseDistribution(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	month, err := ParseMonth(c.Query("month"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	dist, err := h.usecase.ExpenseDistribution(c.Request.Context(), userID, month)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "failed to get expense distribution")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"distribution": dist})
}

func (h *Handler) Forecast(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	month, err := ParseMonth(c.Query("month"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	forecast, err := h.usecase.Forecast(c.Request.Context(), userID, month)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "failed to get forecast")
		return
	}
	httpx.JSON(c, http.StatusOK, forecast)
}
