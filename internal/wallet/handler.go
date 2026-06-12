package wallet

import (
	"net/http"

	"affluena/internal/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
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
	if !IsValidType(req.Type) {
		httpx.Error(c, http.StatusBadRequest, "invalid wallet type")
		return
	}

	wallet, err := h.repo.Create(c.Request.Context(), userID, req.Name, req.Type, req.CurrencyCode, req.BalanceMinor)
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

	wallets, err := h.repo.List(c.Request.Context(), userID)
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

	wallet, err := h.repo.Get(c.Request.Context(), userID, c.Param("id"))
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
	if !IsValidType(req.Type) {
		httpx.Error(c, http.StatusBadRequest, "invalid wallet type")
		return
	}

	wallet, err := h.repo.Update(c.Request.Context(), userID, c.Param("id"), req.Name, req.Type, req.CurrencyCode)
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

	if err := h.repo.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		if NotFound(err) {
			httpx.Error(c, http.StatusNotFound, "wallet not found")
			return
		}
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
