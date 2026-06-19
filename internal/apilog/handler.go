package apilog

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"affluena-api/internal/httpx"

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

func (h *Handler) GetLog(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	log, err := h.repo.GetLogByID(c.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, "api log not found")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "failed to get api log")
		return
	}

	c.JSON(http.StatusOK, log)
}
