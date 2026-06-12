package budget

import (
	"net/http"
	"time"

	"affluena/internal/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
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

	req, month, ok := bindBudgetRequest(c)
	if !ok {
		return
	}

	budget, err := h.repo.Create(c.Request.Context(), userID, req.CategoryID, month, req.LimitMinor)
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

	month, err := ParseBudgetMonth(c.Query("month"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	budgets, err := h.repo.List(c.Request.Context(), userID, month)
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

	budget, err := h.repo.Get(c.Request.Context(), userID, c.Param("id"))
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

	req, month, ok := bindBudgetRequest(c)
	if !ok {
		return
	}

	budget, err := h.repo.Update(c.Request.Context(), userID, c.Param("id"), req.CategoryID, month, req.LimitMinor)
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

	if err := h.repo.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func bindBudgetRequest(c *gin.Context) (budgetRequest, time.Time, bool) {
	var req budgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return budgetRequest{}, time.Time{}, false
	}
	if req.LimitMinor <= 0 {
		httpx.Error(c, http.StatusBadRequest, "limit_minor must be positive")
		return budgetRequest{}, time.Time{}, false
	}
	month, err := ParseBudgetMonth(req.Month)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return budgetRequest{}, time.Time{}, false
	}
	return req, month, true
}

func writeError(c *gin.Context, err error) {
	if NotFound(err) {
		httpx.Error(c, http.StatusNotFound, "category budget not found")
		return
	}
	httpx.Error(c, http.StatusBadRequest, err.Error())
}
