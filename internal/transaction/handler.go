package transaction

import (
	"context"
	"net/http"
	"time"

	"affluena/internal/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase transactionUseCase
}

type transactionUseCase interface {
	Create(ctx context.Context, userID string, input TransactionInput) (Transaction, error)
	List(ctx context.Context, userID string) ([]Transaction, error)
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

	transactions, err := h.usecase.List(c.Request.Context(), userID)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list transactions failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"transactions": transactions})
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

	input := TransactionInput{
		Type:           req.Type,
		WalletID:       req.WalletID,
		ToWalletID:     req.ToWalletID,
		CategoryID:     req.CategoryID,
		AmountMinor:    req.AmountMinor,
		TransactionUTC: transactionAt,
		Note:           req.Note,
	}
	return input, true
}

func writeError(c *gin.Context, err error) {
	if NotFound(err) {
		httpx.Error(c, http.StatusNotFound, "resource not found")
		return
	}
	httpx.Error(c, http.StatusBadRequest, err.Error())
}
