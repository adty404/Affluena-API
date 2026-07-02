package recurring

import (
	"context"
	"errors"
	"net/http"
	"time"

	"affluena-api/internal/httpx"
	"affluena-api/internal/page"
	"affluena-api/internal/transaction"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase recurringUseCase
}

type recurringUseCase interface {
	Create(ctx context.Context, userID string, input RuleInput) (Rule, error)
	List(ctx context.Context, userID string, pagination page.Params) (page.Result[Rule], error)
	Get(ctx context.Context, userID string, id string) (Rule, error)
	Update(ctx context.Context, userID string, id string, input RuleInput) (Rule, error)
	Delete(ctx context.Context, userID string, id string) error
	RunManual(ctx context.Context, userID string, id string, now time.Time) (Run, error)
}

func NewHandler(usecase recurringUseCase) *Handler {
	return &Handler{usecase: usecase}
}

type ruleRequest struct {
	Name          string                      `json:"name" binding:"required"`
	Type          transaction.TransactionType `json:"type" binding:"required"`
	WalletID      string                      `json:"wallet_id" binding:"required"`
	ToWalletID    string                      `json:"to_wallet_id"`
	CategoryID    string                      `json:"category_id"`
	AmountMinor   int64                       `json:"amount_minor" binding:"required"`
	Frequency     Frequency                   `json:"frequency" binding:"required"`
	IntervalCount int                         `json:"interval_count"`
	NextRunAt     string                      `json:"next_run_at" binding:"required"`
	EndAt         string                      `json:"end_at"`
	Status        Status                      `json:"status"`
	Note          string                      `json:"note"`
	Color         string                      `json:"color"`
	Icon          string                      `json:"icon"`
}

type runRequest struct {
	Now string `json:"now"`
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	input, ok := bindRule(c)
	if !ok {
		return
	}

	rule, err := h.usecase.Create(c.Request.Context(), userID, input)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusCreated, rule)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	pagination, ok := httpx.ParsePage(c, "next_run_at_asc", recurringSorts)
	if !ok {
		return
	}
	result, err := h.usecase.List(c.Request.Context(), userID, pagination)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list recurring transactions failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"recurring_transactions": result.Items, "pagination": result.Pagination})
}

var recurringSorts = map[string]struct{}{
	"next_run_at_asc":  {},
	"next_run_at_desc": {},
	"created_at_desc":  {},
	"created_at_asc":   {},
	"name_asc":         {},
	"name_desc":        {},
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
	rule, err := h.usecase.Get(c.Request.Context(), userID, id)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, rule)
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
	input, ok := bindRule(c)
	if !ok {
		return
	}
	rule, err := h.usecase.Update(c.Request.Context(), userID, id, input)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, rule)
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
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) RunManual(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}
	now, ok := bindRunNow(c)
	if !ok {
		return
	}

	run, err := h.usecase.RunManual(c.Request.Context(), userID, id, now)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusCreated, run)
}

func bindRule(c *gin.Context) (RuleInput, bool) {
	var req ruleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return RuleInput{}, false
	}

	nextRunAt, err := time.Parse(time.RFC3339, req.NextRunAt)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "next_run_at must be RFC3339")
		return RuleInput{}, false
	}

	var endAt *time.Time
	if req.EndAt != "" {
		parsedEndAt, err := time.Parse(time.RFC3339, req.EndAt)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "end_at must be RFC3339")
			return RuleInput{}, false
		}
		parsedEndAt = parsedEndAt.UTC()
		endAt = &parsedEndAt
	}

	input := RuleInput{
		Name:          req.Name,
		Type:          req.Type,
		WalletID:      req.WalletID,
		ToWalletID:    req.ToWalletID,
		CategoryID:    req.CategoryID,
		AmountMinor:   req.AmountMinor,
		Frequency:     req.Frequency,
		IntervalCount: req.IntervalCount,
		NextRunAt:     nextRunAt.UTC(),
		EndAt:         endAt,
		Status:        req.Status,
		Note:          req.Note,
		Color:         req.Color,
		Icon:          req.Icon,
	}
	if input.IntervalCount == 0 {
		input.IntervalCount = 1
	}
	if input.Status == "" {
		input.Status = StatusActive
	}
	if err := validateInput(input); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return RuleInput{}, false
	}
	return input, true
}

func bindRunNow(c *gin.Context) (time.Time, bool) {
	now := time.Now().UTC()

	var req runRequest
	if !httpx.BindOptionalJSON(c, &req, "invalid request body") {
		return time.Time{}, false
	}
	if req.Now == "" {
		return now, true
	}
	parsed, err := time.Parse(time.RFC3339, req.Now)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "now must be RFC3339")
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func writeError(c *gin.Context, err error) {
	switch {
	case NotFound(err):
		httpx.Error(c, http.StatusNotFound, "recurring transaction not found")
	case errors.Is(err, ErrAlreadyExecuted):
		httpx.Error(c, http.StatusConflict, "conflict in resource state")
	case errors.Is(err, ErrRuleInactive):
		httpx.Error(c, http.StatusBadRequest, "invalid request")
	default:
		httpx.Error(c, http.StatusBadRequest, "invalid request")
	}
}
