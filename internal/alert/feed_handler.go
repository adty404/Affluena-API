package alert

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"affluena-api/internal/httpx"

	"github.com/gin-gonic/gin"
)

type FeedHandler struct {
	feedUC FeedUseCase
}

func NewFeedHandler(feedUC FeedUseCase) *FeedHandler {
	return &FeedHandler{feedUC: feedUC}
}

func (h *FeedHandler) List(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	monthValue := c.Query("month")
	if monthValue == "" {
		monthValue = time.Now().UTC().Format("2006-01")
	}

	alerts, err := h.feedUC.List(c.Request.Context(), userID, monthValue)
	if err != nil {
		fmt.Println("ERROR:", err)
		httpx.WriteError(c, err)
		return
	}

	if alerts == nil {
		alerts = []Alert{}
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

func (h *FeedHandler) Get(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	id := c.Param("id")
	if id == "" {
		httpx.Error(c, http.StatusBadRequest, "id is required")
		return
	}

	alert, err := h.feedUC.Get(c.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrAlertNotFound) {
			httpx.Error(c, http.StatusNotFound, "alert not found")
			return
		}
		fmt.Println("ERROR:", err)
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, alert)
}
