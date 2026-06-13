package wallet

import (
	"context"
	"net/http"

	"affluena/internal/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase walletUseCase
}

type walletUseCase interface {
	Create(ctx context.Context, userID string, input CreateWalletInput) (Wallet, error)
	List(ctx context.Context, userID string) ([]Wallet, error)
	Get(ctx context.Context, userID string, id string) (Wallet, error)
	Update(ctx context.Context, userID string, id string, input UpdateWalletInput) (Wallet, error)
	Delete(ctx context.Context, userID string, id string) error
}

func NewHandler(usecase walletUseCase) *Handler {
	return &Handler{usecase: usecase}
}

type createWalletRequest struct {
	Name         string `json:"name" binding:"required"`
	Type         string `json:"type" binding:"required"`
	CurrencyCode string `json:"currency_code"`
	BalanceMinor int64  `json:"balance_minor"`
}

type updateWalletRequest struct {
	Name         string `json:"name" binding:"required"`
	Type         string `json:"type" binding:"required"`
	CurrencyCode string `json:"currency_code" binding:"required"`
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
	})
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(c, http.StatusCreated, wallet)
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	wallets, err := h.usecase.List(c.Request.Context(), userID)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "list wallets failed")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"wallets": wallets})
}

func (h *Handler) Get(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	wallet, err := h.usecase.Get(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		if NotFound(err) {
			httpx.Error(c, http.StatusNotFound, "wallet not found")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "get wallet failed")
		return
	}
	httpx.JSON(c, http.StatusOK, wallet)
}

func (h *Handler) Update(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	var req updateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	wallet, err := h.usecase.Update(c.Request.Context(), userID, c.Param("id"), UpdateWalletInput{
		Name:         req.Name,
		Type:         req.Type,
		CurrencyCode: req.CurrencyCode,
	})
	if err != nil {
		if NotFound(err) {
			httpx.Error(c, http.StatusNotFound, "wallet not found")
			return
		}
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(c, http.StatusOK, wallet)
}

func (h *Handler) Delete(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	if err := h.usecase.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		if NotFound(err) {
			httpx.Error(c, http.StatusNotFound, "wallet not found")
			return
		}
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
