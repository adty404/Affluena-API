package report

import (
	"context"
	"net/http"

	"affluena-api/internal/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	uc reportUseCase
}

type reportUseCase interface {
	IncomeReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error)
	ExpenseReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error)
	CashflowReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error)
	DebtReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error)
	GoalReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error)
	OverviewReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error)
}

func NewHandler(uc reportUseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Income(c *gin.Context) {
	userID := c.GetString("user_id")
	month := c.Query("month")

	resp, err := h.uc.IncomeReport(c.Request.Context(), userID, month)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Expense(c *gin.Context) {
	userID := c.GetString("user_id")
	month := c.Query("month")

	resp, err := h.uc.ExpenseReport(c.Request.Context(), userID, month)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Cashflow(c *gin.Context) {
	userID := c.GetString("user_id")
	month := c.Query("month")

	resp, err := h.uc.CashflowReport(c.Request.Context(), userID, month)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Debt(c *gin.Context) {
	userID := c.GetString("user_id")
	month := c.Query("month")

	resp, err := h.uc.DebtReport(c.Request.Context(), userID, month)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Goal(c *gin.Context) {
	userID := c.GetString("user_id")
	month := c.Query("month")

	resp, err := h.uc.GoalReport(c.Request.Context(), userID, month)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Overview(c *gin.Context) {
	userID := c.GetString("user_id")
	month := c.Query("month")

	resp, err := h.uc.OverviewReport(c.Request.Context(), userID, month)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
