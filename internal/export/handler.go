package export

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"affluena-api/internal/httpx"

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
		httpx.Error(c, http.StatusBadRequest, "from must be before or equal to to")
		return
	}
	opts.From = from
	opts.To = to

	csvData, err := h.useCase.GenerateCSVData(c.Request.Context(), userID, opts)
	if err != nil {
		_, _ = h.useCase.RecordJob(c.Request.Context(), userID, "CSV", opts, 0, "failed")
		httpx.WriteError(c, err)
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=\"transactions_export.csv\"")

	writer := csv.NewWriter(c.Writer)
	if err := writer.WriteAll(csvData); err != nil {
		_, _ = h.useCase.RecordJob(c.Request.Context(), userID, "CSV", opts, 0, "failed")
		// Cannot write JSON response if headers were already sent, so just log or ignore
		return
	}

	// Record success
	// rowCount is len(csvData) - 1 (excluding header)
	rowCount := len(csvData) - 1
	if rowCount < 0 {
		rowCount = 0
	}
	_, _ = h.useCase.RecordJob(c.Request.Context(), userID, "CSV", opts, rowCount, "completed")
}

func (h *Handler) ListJobs(c *gin.Context) {
	userID := c.GetString("user_id")

	limitStr := c.DefaultQuery("limit", "100")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 100
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	jobs, total, err := h.useCase.ListJobs(c.Request.Context(), userID, limit, offset)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs": jobs,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	})
}

func (h *Handler) GetJob(c *gin.Context) {
	userID := c.GetString("user_id")
	id, ok := httpx.GetUUIDParam(c, "id")
	if !ok {
		return
	}

	job, err := h.useCase.GetJob(c.Request.Context(), userID, id)
	if err != nil {
		if err == ErrJobNotFound {
			httpx.Error(c, http.StatusNotFound, "job not found")
			return
		}
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, job)
}

func parseExportTime(c *gin.Context, key string) (time.Time, bool) {
	value := c.Query(key)
	if value == "" {
		return time.Time{}, true
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, key+" must be RFC3339")
		return time.Time{}, false
	}
	return parsed.UTC(), true
}
