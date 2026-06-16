package quickentry

import (
	"context"
	"net/http"
	"time"

	"affluena-api/internal/httpx"
	"affluena-api/internal/page"
	"affluena-api/internal/transaction"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase quickEntryUseCase
}

type quickEntryUseCase interface {
	Create(ctx context.Context, userID string, template Template) (Template, error)
	List(ctx context.Context, userID string, pagination page.Params) (page.Result[Template], error)
	Get(ctx context.Context, userID string, id string) (Template, error)
	Update(ctx context.Context, userID string, id string, template Template) (Template, error)
	Delete(ctx context.Context, userID string, id string) error
	Execute(ctx context.Context, userID string, id string, input ExecuteInput) (ExecuteResult, error)
}

func NewHandler(usecase quickEntryUseCase) *Handler {
	return &Handler{usecase: usecase}
}

type templateRequest struct {
	Name        string                      `json:"name" binding:"required"`
	Type        transaction.TransactionType `json:"type" binding:"required"`
	WalletID    string                      `json:"wallet_id" binding:"required"`
	ToWalletID  string                      `json:"to_wallet_id"`
	CategoryID  string                      `json:"category_id"`
	AmountMinor int64                       `json:"amount_minor" binding:"required"`
	Note        string                      `json:"note"`
}

type executeRequest struct {
	TransactionAt string `json:"transaction_at"`
	Note          string `json:"note"`
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	template, ok := bindTemplate(c)
	if !ok {
		return
	}
	created, err := h.usecase.Create(c.Request.Context(), userID, template)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusCreated, created)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	pagination, ok := httpx.ParsePage(c, "name_asc", templateSorts)
	if !ok {
		return
	}
	result, err := h.usecase.List(c.Request.Context(), userID, pagination)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list quick entry templates failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"templates": result.Items, "pagination": result.Pagination})
}

var templateSorts = map[string]struct{}{
	"name_asc":        {},
	"name_desc":       {},
	"created_at_desc": {},
	"created_at_asc":  {},
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

	template, err := h.usecase.Get(c.Request.Context(), userID, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, template)
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

	template, ok := bindTemplate(c)
	if !ok {
		return
	}
	updated, err := h.usecase.Update(c.Request.Context(), userID, id, template)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, updated)
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

func (h *Handler) Execute(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}
	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	var req executeRequest
	if !httpx.BindOptionalJSON(c, &req, "invalid request body") {
		return
	}

	transactionAt := time.Now().UTC()
	if req.TransactionAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.TransactionAt)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "transaction_at must be RFC3339")
			return
		}
		transactionAt = parsed.UTC()
	}

	result, err := h.usecase.Execute(c.Request.Context(), userID, id, ExecuteInput{
		TransactionAt: transactionAt,
		Note:          req.Note,
	})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	httpx.JSON(c, http.StatusCreated, gin.H{"transaction": result.Transaction})
}

func bindTemplate(c *gin.Context) (Template, bool) {
	var req templateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return Template{}, false
	}

	return Template{
		Name:        req.Name,
		Type:        string(req.Type),
		WalletID:    req.WalletID,
		ToWalletID:  req.ToWalletID,
		CategoryID:  req.CategoryID,
		AmountMinor: req.AmountMinor,
		Note:        req.Note,
	}, true
}
