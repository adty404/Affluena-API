package transaction

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
	transaction, err := h.repo.Create(c.Request.Context(), userID, input)
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

	transactions, err := h.repo.List(c.Request.Context(), userID)
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

	transaction, err := h.repo.Get(c.Request.Context(), userID, c.Param("id"))
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
	transaction, err := h.repo.Update(c.Request.Context(), userID, c.Param("id"), input)
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

	if err := h.repo.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
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
	if _, err := BalanceDeltas(input); err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return TransactionInput{}, false
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
