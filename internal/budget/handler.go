package budget

import (
	"context"
	"net/http"

	"affluena-api/internal/httpx"
	"affluena-api/internal/page"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase budgetUseCase
}

type budgetUseCase interface {
	Create(ctx context.Context, userID string, input CreateBudgetInput) (Budget, error)
	List(ctx context.Context, userID string, monthValue string, pagination page.Params) (page.Result[BudgetSummary], error)
	Alerts(ctx context.Context, userID string, monthValue string) ([]BudgetAlert, error)
	Report(ctx context.Context, userID string, monthValue string) ([]BudgetReportItem, BudgetReportSummary, error)
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
	Color      string `json:"color"`
	Icon       string `json:"icon"`
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
		Color:      req.Color,
		Icon:       req.Icon,
	})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusCreated, budget)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	pagination, ok := httpx.ParsePage(c, "created_at_desc", budgetSorts)
	if !ok {
		return
	}
	result, err := h.usecase.List(c.Request.Context(), userID, c.Query("month"), pagination)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"budgets": result.Items, "pagination": result.Pagination})
}

func (h *Handler) Alerts(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	alerts, err := h.usecase.Alerts(c.Request.Context(), userID, c.Query("month"))
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"alerts": alerts})
}

func (h *Handler) Report(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	items, summary, err := h.usecase.Report(c.Request.Context(), userID, c.Query("month"))
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"report": items, "summary": summary})
}

var budgetSorts = map[string]struct{}{
	"created_at_desc": {},
	"created_at_asc":  {},
	"limit_desc":      {},
	"limit_asc":       {},
	"spent_desc":      {},
	"spent_asc":       {},
}

func (h *Handler) Get(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	budget, err := h.usecase.Get(c.Request.Context(), userID, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, budget)
}

func (h *Handler) Update(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	req, ok := bindBudgetRequest(c)
	if !ok {
		return
	}

	budget, err := h.usecase.Update(c.Request.Context(), userID, id, UpdateBudgetInput{
		CategoryID: req.CategoryID,
		Month:      req.Month,
		LimitMinor: req.LimitMinor,
		Color:      req.Color,
		Icon:       req.Icon,
	})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, budget)
}

func (h *Handler) Delete(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.usecase.Delete(c.Request.Context(), userID, id); err != nil {
		httpx.WriteError(c, err)
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
