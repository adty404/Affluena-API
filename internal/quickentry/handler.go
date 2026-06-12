package quickentry

import (
	"net/http"
	"time"

	"affluena/internal/httpx"
	"affluena/internal/transaction"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo            *Repository
	transactionRepo *transaction.Repository
}

func NewHandler(repo *Repository, transactionRepo *transaction.Repository) *Handler {
	return &Handler{repo: repo, transactionRepo: transactionRepo}
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
	created, err := h.repo.Create(c.Request.Context(), userID, template)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(c, http.StatusCreated, created)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	templates, err := h.repo.List(c.Request.Context(), userID)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list quick entry templates failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"templates": templates})
}

func (h *Handler) Get(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	template, err := h.repo.Get(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, template)
}

func (h *Handler) Update(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	template, ok := bindTemplate(c)
	if !ok {
		return
	}
	updated, err := h.repo.Update(c.Request.Context(), userID, c.Param("id"), template)
	if err != nil {
		writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, updated)
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

func (h *Handler) Execute(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	template, err := h.repo.Get(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}

	var req executeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
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

	note := template.Note
	if req.Note != "" {
		note = req.Note
	}

	created, err := h.transactionRepo.Create(c.Request.Context(), userID, transaction.TransactionInput{
		Type:           transaction.TransactionType(template.Type),
		WalletID:       template.WalletID,
		ToWalletID:     template.ToWalletID,
		CategoryID:     template.CategoryID,
		AmountMinor:    template.AmountMinor,
		TransactionUTC: transactionAt,
		Note:           note,
	})
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	httpx.JSON(c, http.StatusCreated, gin.H{"transaction": created})
}

func bindTemplate(c *gin.Context) (Template, bool) {
	var req templateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return Template{}, false
	}

	input := transaction.TransactionInput{
		Type:           req.Type,
		WalletID:       req.WalletID,
		ToWalletID:     req.ToWalletID,
		CategoryID:     req.CategoryID,
		AmountMinor:    req.AmountMinor,
		TransactionUTC: time.Now().UTC(),
		Note:           req.Note,
	}
	if _, err := transaction.BalanceDeltas(input); err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
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

func writeError(c *gin.Context, err error) {
	if NotFound(err) {
		httpx.Error(c, http.StatusNotFound, "quick entry template not found")
		return
	}
	httpx.Error(c, http.StatusBadRequest, err.Error())
}
