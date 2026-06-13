package export

import (
	"encoding/csv"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	useCase *UseCase
}

func NewHandler(useCase *UseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) ExportCSV(c *gin.Context) {
	userID := c.GetString("user_id")

	var opts ExportOptions
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			opts.From = t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			opts.To = t
		}
	}

	csvData, err := h.useCase.GenerateCSVData(c.Request.Context(), userID, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=\"transactions_export.csv\"")

	writer := csv.NewWriter(c.Writer)
	if err := writer.WriteAll(csvData); err != nil {
		// Cannot write JSON response if headers were already sent, so just log or ignore
		return
	}
}
