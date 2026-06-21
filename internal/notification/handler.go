package notification

import (
	"net/http"

	"affluena-api/internal/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	uc UseCase
}

func NewHandler(uc UseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	rules, err := h.uc.List(c.Request.Context(), userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

func (h *Handler) Update(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	var update NotificationRuleUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	rule, err := h.uc.Update(c.Request.Context(), userID, id, update)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, rule)
}
