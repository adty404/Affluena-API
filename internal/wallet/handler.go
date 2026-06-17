package wallet

import (
	"context"
	"net/http"
	"time"

	"affluena-api/internal/httpx"
	"affluena-api/internal/page"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase walletUseCase
}

type walletUseCase interface {
	Create(ctx context.Context, userID string, input CreateWalletInput) (Wallet, error)
	List(ctx context.Context, userID string, pagination page.Params) (page.Result[Wallet], error)
	Get(ctx context.Context, userID string, id string) (Wallet, error)
	Update(ctx context.Context, userID string, id string, input UpdateWalletInput) (Wallet, error)
	Delete(ctx context.Context, userID string, id string) error
	InviteMember(ctx context.Context, userID string, id string, input InviteMemberInput) error
	RespondInvite(ctx context.Context, userID string, id string, memberID string, input RespondInviteInput) error
	GetMembers(ctx context.Context, userID string, walletID string) ([]WalletMember, error)
	GetAnalytics(ctx context.Context, userID string, walletID string, month string) (WalletAnalytics, error)
}

func NewHandler(usecase walletUseCase) *Handler {
	return &Handler{usecase: usecase}
}

type createWalletRequest struct {
	Name         string `json:"name" binding:"required"`
	Type         string `json:"type" binding:"required"`
	CurrencyCode string `json:"currency_code"`
	BalanceMinor int64  `json:"balance_minor"`
	Color        string `json:"color"`
	Description  string `json:"description"`
}

type updateWalletRequest struct {
	Name         string `json:"name" binding:"required"`
	Type         string `json:"type" binding:"required"`
	CurrencyCode string `json:"currency_code" binding:"required"`
	Color        string `json:"color"`
	Description  string `json:"description"`
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	var req createWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	wallet, err := h.usecase.Create(c.Request.Context(), userID, CreateWalletInput{
		Name:         req.Name,
		Type:         req.Type,
		CurrencyCode: req.CurrencyCode,
		BalanceMinor: req.BalanceMinor,
		Color:        req.Color,
		Description:  req.Description,
	})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusCreated, wallet)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	pagination, ok := httpx.ParsePage(c, "created_at_desc", walletSorts)
	if !ok {
		return
	}
	result, err := h.usecase.List(c.Request.Context(), userID, pagination)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list wallets failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"wallets": result.Items, "pagination": result.Pagination})
}

var walletSorts = map[string]struct{}{
	"created_at_desc": {},
	"created_at_asc":  {},
	"name_asc":        {},
	"name_desc":       {},
	"balance_desc":    {},
	"balance_asc":     {},
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

	wallet, err := h.usecase.Get(c.Request.Context(), userID, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, wallet)
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

	var req updateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	wallet, err := h.usecase.Update(c.Request.Context(), userID, id, UpdateWalletInput{
		Name:         req.Name,
		Type:         req.Type,
		CurrencyCode: req.CurrencyCode,
		Color:        req.Color,
		Description:  req.Description,
	})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, wallet)
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

func (h *Handler) InviteMember(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	var req InviteMemberInput
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.usecase.InviteMember(c.Request.Context(), userID, id, req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	c.Status(http.StatusCreated)
}

func (h *Handler) RespondInvite(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	memberID, ok := httpx.GetUUIDParam(c, "member_id")
	if !ok {
		return
	}

	var req RespondInviteInput
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.usecase.RespondInvite(c.Request.Context(), userID, id, memberID, req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) ListMembers(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	members, err := h.usecase.GetMembers(c.Request.Context(), userID, id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"members": members})
}

func (h *Handler) Analytics(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	month := c.Query("month")
	if month == "" {
		month = time.Now().UTC().Format("2006-01")
	}
	if !isValidYearMonth(month) {
		httpx.Error(c, http.StatusBadRequest, "invalid month format, expected YYYY-MM")
		return
	}

	analytics, err := h.usecase.GetAnalytics(c.Request.Context(), userID, id, month)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, analytics)
}

func isValidYearMonth(s string) bool {
	if len(s) != 7 || s[4] != '-' {
		return false
	}
	for i := 0; i < 7; i++ {
		if i == 4 {
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
