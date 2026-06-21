package report

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeReportUseCase struct {
	err error
}

func (f fakeReportUseCase) IncomeReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error) {
	return ReportResponse{}, f.err
}

func (f fakeReportUseCase) ExpenseReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error) {
	return ReportResponse{}, f.err
}

func (f fakeReportUseCase) CashflowReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error) {
	return ReportResponse{}, f.err
}

func (f fakeReportUseCase) DebtReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error) {
	return ReportResponse{}, f.err
}

func (f fakeReportUseCase) GoalReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error) {
	return ReportResponse{}, f.err
}

func (f fakeReportUseCase) OverviewReport(ctx context.Context, userID string, monthStr string) (ReportResponse, error) {
	return ReportResponse{}, f.err
}

func TestHandler_Income_sanitizes_internal_error_response(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(fakeReportUseCase{err: errors.New("database password leaked")})
	router := gin.New()
	router.GET("/reports/income", func(c *gin.Context) {
		c.Set("user_id", "user-1")
		handler.Income(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/reports/income?month=2026-06", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Equal(t, "internal server error", decodeErrorMessage(t, w))
	require.NotContains(t, w.Body.String(), "database password leaked")
}

func decodeErrorMessage(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()

	var body struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body.Error
}
