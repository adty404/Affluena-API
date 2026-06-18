package apilog

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	logs, err := h.repo.ListLogs(context.Background(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list api logs failed"})
		return
	}
	if logs == nil {
		logs = []APILog{}
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}
