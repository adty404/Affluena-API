package budget

import (
	"context"
	"net/http"

	"affluena/internal/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase budgetUseCase
}

type budgetUseCase interface {
	Create(ctx context.Context, userID string, input CreateBudgetInput) (Budget, error)
	List(ctx context.Context, userID string, monthValue string) ([]BudgetSummary, error)
	Get(ctx context.Context, userID string, id string) (Budget, error)
	Update(ctx context.Context, userID string, id string, input UpdateBudgetInput) (Budget, error)
	Delete(ctx context.Context, userID string, id string) error
}

func NewHandler(usecase budgetUseCase) *Handler {
	return &Handler{usecase: usecase}
}

type budgetRequest struct {
	CategoryID string `json:"category_id" binding:"required"`
	Month      string `json:"month" binding:"required"`
	LimitMinor int64  `json:"limit_minor" binding:"required"`
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	req, ok := bindBudgetRequest(c)
	if !ok {
		return
	}

	budget, err := h.usecase.Create(c.Request.Context(), userID, CreateBudgetInput{
		CategoryID: req.CategoryID,
		Month:      req.Month,
		LimitMinor: req.LimitMinor,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusCreated, budget)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	budgets, err := h.usecase.List(c.Request.Context(), userID, c.Query("month"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list category budgets failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"budgets": budgets})
}

func (h *Handler) Get(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	budget, err := h.usecase.Get(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, budget)
}

func (h *Handler) Update(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	req, ok := bindBudgetRequest(c)
	if !ok {
		return
	}

	budget, err := h.usecase.Update(c.Request.Context(), userID, c.Param("id"), UpdateBudgetInput{
		CategoryID: req.CategoryID,
		Month:      req.Month,
		LimitMinor: req.LimitMinor,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, budget)
}

func (h *Handler) Delete(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	if err := h.usecase.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func bindBudgetRequest(c *gin.Context) (budgetRequest, bool) {
	var req budgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return budgetRequest{}, false
	}
	return req, true
}

func writeError(c *gin.Context, err error) {
	if NotFound(err) {
		httpx.Error(c, http.StatusNotFound, "category budget not found")
		return
	}
	httpx.Error(c, http.StatusBadRequest, err.Error())
}
