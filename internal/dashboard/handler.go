package dashboard

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"affluena-api/internal/httpx"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	usecase dashboardUseCase
}

type dashboardUseCase interface {
	Summary(ctx context.Context, userID string, month time.Time) (Summary, error)
	CashflowTrend(ctx context.Context, userID string, opts CashflowTrendOptions) ([]CashflowTrend, error)
	ExpenseDistribution(ctx context.Context, userID string, month time.Time) ([]ExpenseDistribution, error)
	Forecast(ctx context.Context, userID string, month time.Time) (Forecast, error)
}

func NewHandler(usecase dashboardUseCase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Summary(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	month, err := ParseMonth(c.Query("month"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}

	summary, err := h.usecase.Summary(c.Request.Context(), userID, month)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "dashboard summary failed")
		return
	}
	httpx.JSON(c, http.StatusOK, summary)
}

func (h *Handler) CashflowTrend(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	opts, errMsg := parseCashflowOptions(c)
	if errMsg != "" {
		httpx.Error(c, http.StatusBadRequest, errMsg)
		return
	}

	trend, err := h.usecase.CashflowTrend(c.Request.Context(), userID, opts)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "failed to get cashflow trend")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"trend": trend})
}

// parseCashflowOptions reads the cashflow-trend query params:
//   - granularity = month (default) | week
//   - months (1-12) for month granularity without an explicit range (default 6)
//   - weeks (1-26) for week granularity without an explicit range (default 8)
//   - from / to (YYYY-MM-DD or RFC3339) define an explicit inclusive range
//
// It returns a non-empty message on validation error.
func parseCashflowOptions(c *gin.Context) (CashflowTrendOptions, string) {
	opts := CashflowTrendOptions{Granularity: GranularityMonth, Months: 6, Weeks: 8}

	switch c.Query("granularity") {
	case "", "month":
		opts.Granularity = GranularityMonth
	case "week":
		opts.Granularity = GranularityWeek
	default:
		return CashflowTrendOptions{}, "granularity must be month or week"
	}

	if m := c.Query("months"); m != "" {
		parsed, err := strconv.Atoi(m)
		if err != nil || parsed <= 0 || parsed > 12 {
			return CashflowTrendOptions{}, "months must be between 1 and 12"
		}
		opts.Months = parsed
	}
	if w := c.Query("weeks"); w != "" {
		parsed, err := strconv.Atoi(w)
		if err != nil || parsed <= 0 || parsed > 26 {
			return CashflowTrendOptions{}, "weeks must be between 1 and 26"
		}
		opts.Weeks = parsed
	}

	from, ok := parseFlexibleDate(c.Query("from"))
	if !ok {
		return CashflowTrendOptions{}, "from must use YYYY-MM-DD or RFC3339"
	}
	to, ok := parseFlexibleDate(c.Query("to"))
	if !ok {
		return CashflowTrendOptions{}, "to must use YYYY-MM-DD or RFC3339"
	}
	if !from.IsZero() && !to.IsZero() && from.After(to) {
		return CashflowTrendOptions{}, "from must be before or equal to to"
	}
	opts.From = from
	opts.To = to
	return opts, ""
}

// parseFlexibleDate parses an empty string (zero time, ok), a YYYY-MM-DD date,
// or an RFC3339 timestamp, normalizing to a UTC calendar date.
func parseFlexibleDate(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, true
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC), true
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		p := parsed.UTC()
		return time.Date(p.Year(), p.Month(), p.Day(), 0, 0, 0, 0, time.UTC), true
	}
	return time.Time{}, false
}

func (h *Handler) ExpenseDistribution(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	month, err := ParseMonth(c.Query("month"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}

	dist, err := h.usecase.ExpenseDistribution(c.Request.Context(), userID, month)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "failed to get expense distribution")
		return
	}
	httpx.JSON(c, http.StatusOK, gin.H{"distribution": dist})
}

func (h *Handler) Forecast(c *gin.Context) {
	userID, ok := httpx.MustUserID(c)
	if !ok {
		return
	}

	month, err := ParseMonth(c.Query("month"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}

	forecast, err := h.usecase.Forecast(c.Request.Context(), userID, month)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "failed to get forecast")
		return
	}
	httpx.JSON(c, http.StatusOK, forecast)
}
