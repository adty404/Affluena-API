package activity

import (
	"errors"
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

func (h *Handler) ListActivities(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	p, ok := httpx.ParsePage(c, "created_at_desc", activitySorts)
	if !ok {
		return
	}
	limit, offset := p.Limit, p.Offset

	activities, total, err := h.uc.ListActivities(c.Request.Context(), userID.(string), limit, offset, p.Sort)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list activities"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": activities,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	})
}

var activitySorts = map[string]struct{}{
	"created_at_desc": {},
	"created_at_asc":  {},
}

func (h *Handler) GetActivity(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	activity, err := h.uc.GetActivity(c.Request.Context(), userID.(string), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, "activity not found")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "failed to get activity")
		return
	}

	c.JSON(http.StatusOK, activity)
}
