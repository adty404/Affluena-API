package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"affluena-api/internal/config"
	"affluena-api/internal/export"
)

func TestExportJobsIntegration(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "export-jobs-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "export-jobs-owner")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	userID2, token2 := registerIntegrationAPIUser(t, router, "export-jobs-other")
	defer cleanupServerIntegrationUsers(t, pool, userID2)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Cash wallet",
		"type": "cash",
		"currency_code": "IDR",
		"balance_minor": 10000000
	}`)

	catID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Food",
		"type": "expense"
	}`)

	createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+catID+`",
		"amount_minor": 5000000,
		"transaction_at": "2025-01-01T12:00:00Z",
		"note": "Lunch with friends"
	}`)

	// 1. Trigger GET /export/csv to seed an export_jobs row
	req, _ := http.NewRequest("GET", "/api/v1/export/csv", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. body: %s", w.Code, w.Body.String())
	}

	// 2. GET /export/jobs
	req, _ = http.NewRequest("GET", "/api/v1/export/jobs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. body: %s", w.Code, w.Body.String())
	}

	var listResp struct {
		Jobs       []export.ExportJob `json:"jobs"`
		Pagination struct {
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
			Total  int `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if listResp.Pagination.Total != 1 {
		t.Fatalf("expected 1 job, got %d", listResp.Pagination.Total)
	}
	if len(listResp.Jobs) != 1 {
		t.Fatalf("expected 1 job in list, got %d", len(listResp.Jobs))
	}

	job := listResp.Jobs[0]
	if job.Status != "completed" {
		t.Fatalf("expected status completed, got %s", job.Status)
	}
	if job.RowCount != 1 {
		t.Fatalf("expected row_count 1, got %d", job.RowCount)
	}
	if job.Format != "CSV" {
		t.Fatalf("expected format CSV, got %s", job.Format)
	}

	// 3. GET /export/jobs/:id
	req, _ = http.NewRequest("GET", "/api/v1/export/jobs/"+job.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. body: %s", w.Code, w.Body.String())
	}

	var detailJob export.ExportJob
	if err := json.Unmarshal(w.Body.Bytes(), &detailJob); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if detailJob.ID != job.ID {
		t.Fatalf("expected job ID %s, got %s", job.ID, detailJob.ID)
	}

	// 4. GET /export/jobs/:id with other user -> 404
	req, _ = http.NewRequest("GET", "/api/v1/export/jobs/"+job.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token2)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d. body: %s", w.Code, w.Body.String())
	}

	// 5. Isolation: user 2 list is empty
	req, _ = http.NewRequest("GET", "/api/v1/export/jobs", nil)
	req.Header.Set("Authorization", "Bearer "+token2)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. body: %s", w.Code, w.Body.String())
	}

	var listResp2 struct {
		Jobs       []export.ExportJob `json:"jobs"`
		Pagination struct {
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
			Total  int `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp2); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if listResp2.Pagination.Total != 0 {
		t.Fatalf("expected 0 jobs for user 2, got %d", listResp2.Pagination.Total)
	}
	if len(listResp2.Jobs) != 0 {
		t.Fatalf("expected 0 jobs in list for user 2, got %d", len(listResp2.Jobs))
	}
}
