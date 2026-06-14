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
	from, ok := parseExportTime(c, "from")
	if !ok {
		return
	}
	to, ok := parseExportTime(c, "to")
	if !ok {
		return
	}
	if !from.IsZero() && !to.IsZero() && from.After(to) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from must be before or equal to to"})
		return
	}
	opts.From = from
	opts.To = to

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

func parseExportTime(c *gin.Context, key string) (time.Time, bool) {
	value := c.Query(key)
	if value == "" {
		return time.Time{}, true
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": key + " must be RFC3339"})
		return time.Time{}, false
	}
	return parsed.UTC(), true
}
