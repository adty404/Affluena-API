package transaction

import (
	"context"
	"net/http"
	"time"

	"affluena-api/internal/httpx"
	"affluena-api/internal/page"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase transactionUseCase
}

type transactionUseCase interface {
	Create(ctx context.Context, userID string, input TransactionInput) (Transaction, error)
	List(ctx context.Context, userID string, filter TransactionFilter, pagination page.Params) (page.Result[Transaction], error)
	Get(ctx context.Context, userID string, id string) (Transaction, error)
	Update(ctx context.Context, userID string, id string, input TransactionInput) (Transaction, error)
	Delete(ctx context.Context, userID string, id string) error
}

func NewHandler(usecase transactionUseCase) *Handler {
	return &Handler{usecase: usecase}
}

type transactionRequest struct {
	Type          TransactionType `json:"type" binding:"required"`
	WalletID      string          `json:"wallet_id" binding:"required"`
	ToWalletID    string          `json:"to_wallet_id"`
	CategoryID    string          `json:"category_id"`
	AmountMinor   int64           `json:"amount_minor" binding:"required"`
	TagIDs        []string        `json:"tag_ids"`
	TransactionAt string          `json:"transaction_at"`
	Note          string          `json:"note"`
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	input, ok := bindInput(c)
	if !ok {
		return
	}
	transaction, err := h.usecase.Create(c.Request.Context(), userID, input)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusCreated, transaction)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	filter, ok := bindFilter(c)
	if !ok {
		return
	}

	pagination, ok := httpx.ParsePage(c, "transaction_at_desc", transactionSorts)
	if !ok {
		return
	}

	result, err := h.usecase.List(c.Request.Context(), userID, filter, pagination)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list transactions failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"transactions": result.Items, "pagination": result.Pagination})
}

var transactionSorts = map[string]struct{}{
	"transaction_at_desc": {},
	"transaction_at_asc":  {},
	"created_at_desc":     {},
	"created_at_asc":      {},
	"amount_desc":         {},
	"amount_asc":          {},
}

func (h *Handler) Get(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	transaction, err := h.usecase.Get(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, transaction)
}

func (h *Handler) Update(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	input, ok := bindInput(c)
	if !ok {
		return
	}
	transaction, err := h.usecase.Update(c.Request.Context(), userID, c.Param("id"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, transaction)
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

func bindFilter(c *gin.Context) (TransactionFilter, bool) {
	filter := TransactionFilter{
		Type:       TransactionType(c.Query("type")),
		WalletID:   c.Query("wallet_id"),
		CategoryID: c.Query("category_id"),
		TagID:      c.Query("tag_id"),
	}
	if filter.Type != "" && !IsValidType(filter.Type) {
		httpx.Error(c, http.StatusBadRequest, "invalid transaction type")
		return TransactionFilter{}, false
	}
	for _, candidate := range []struct {
		name  string
		value string
	}{
		{name: "wallet_id", value: filter.WalletID},
		{name: "category_id", value: filter.CategoryID},
		{name: "tag_id", value: filter.TagID},
	} {
		if candidate.value != "" && !isUUID(candidate.value) {
			httpx.Error(c, http.StatusBadRequest, candidate.name+" must be a UUID")
			return TransactionFilter{}, false
		}
	}

	from, ok := parseFilterDate(c, "from")
	if !ok {
		return TransactionFilter{}, false
	}
	to, ok := parseFilterDate(c, "to")
	if !ok {
		return TransactionFilter{}, false
	}
	if !from.IsZero() && !to.IsZero() && from.After(to) {
		httpx.Error(c, http.StatusBadRequest, "from must be before or equal to to")
		return TransactionFilter{}, false
	}
	filter.From = from
	if !to.IsZero() {
		filter.To = to.AddDate(0, 0, 1)
	}
	return filter, true
}

func parseFilterDate(c *gin.Context, key string) (time.Time, bool) {
	value := c.Query(key)
	if value == "" {
		return time.Time{}, true
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, key+" must use YYYY-MM-DD")
		return time.Time{}, false
	}
	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC), true
}

func bindInput(c *gin.Context) (TransactionInput, bool) {
	var req transactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return TransactionInput{}, false
	}

	transactionAt := time.Now().UTC()
	if req.TransactionAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.TransactionAt)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "transaction_at must be RFC3339")
			return TransactionInput{}, false
		}
		transactionAt = parsed.UTC()
	}
	for _, tagID := range req.TagIDs {
		if !isUUID(tagID) {
			httpx.Error(c, http.StatusBadRequest, "tag_ids must contain UUIDs")
			return TransactionInput{}, false
		}
	}

	input := TransactionInput{
		Type:           req.Type,
		WalletID:       req.WalletID,
		ToWalletID:     req.ToWalletID,
		CategoryID:     req.CategoryID,
		AmountMinor:    req.AmountMinor,
		TagIDs:         req.TagIDs,
		TransactionUTC: transactionAt,
		Note:           req.Note,
	}
	return input, true
}

func isUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, char := range value {
		switch i {
		case 8, 13, 18, 23:
			if char != '-' {
				return false
			}
		default:
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				return false
			}
		}
	}
	return true
}

func writeError(c *gin.Context, err error) {
	if NotFound(err) {
		httpx.Error(c, http.StatusNotFound, "resource not found")
		return
	}
	httpx.Error(c, http.StatusBadRequest, err.Error())
}
