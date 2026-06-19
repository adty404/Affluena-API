package report

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	uc *UseCase
}

func NewHandler(uc *UseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Income(c *gin.Context) {
	userID := c.GetString("user_id")
	month := c.Query("month")

	resp, err := h.uc.IncomeReport(c.Request.Context(), userID, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Expense(c *gin.Context) {
	userID := c.GetString("user_id")
	month := c.Query("month")

	resp, err := h.uc.ExpenseReport(c.Request.Context(), userID, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Cashflow(c *gin.Context) {
	userID := c.GetString("user_id")
	month := c.Query("month")

	resp, err := h.uc.CashflowReport(c.Request.Context(), userID, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Debt(c *gin.Context) {
	userID := c.GetString("user_id")
	month := c.Query("month")

	resp, err := h.uc.DebtReport(c.Request.Context(), userID, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Goal(c *gin.Context) {
	userID := c.GetString("user_id")
	month := c.Query("month")

	resp, err := h.uc.GoalReport(c.Request.Context(), userID, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Overview(c *gin.Context) {
	userID := c.GetString("user_id")
	month := c.Query("month")

	resp, err := h.uc.OverviewReport(c.Request.Context(), userID, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
