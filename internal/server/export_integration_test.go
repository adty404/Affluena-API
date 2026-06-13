package server

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"affluena-api/internal/config"
)

func TestExportCSV(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "export-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "export-owner")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	userID2, token2 := registerIntegrationAPIUser(t, router, "export-other")
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

	tagID := createAPIResource(t, router, token, "/api/v1/tags", `{
		"name": "Lunch"
	}`)

	txID := createAPIResource(t, router, token, "/api/v1/transactions", `{
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+catID+`",
		"amount_minor": 5000000,
		"transaction_at": "2025-01-01T12:00:00Z",
		"note": "Lunch with friends",
		"tag_ids": ["`+tagID+`"]
	}`)

	// User 2 tx
	walletID2 := createAPIResource(t, router, token2, "/api/v1/wallets", `{
		"name": "Bank",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 10000000
	}`)
	catID2 := createAPIResource(t, router, token2, "/api/v1/categories", `{
		"name": "Salary Category",
		"type": "income"
	}`)

	txID2 := createAPIResource(t, router, token2, "/api/v1/transactions", `{
		"type": "income",
		"wallet_id": "`+walletID2+`",
		"category_id": "`+catID2+`",
		"amount_minor": 10000000,
		"transaction_at": "2025-01-02T10:00:00Z",
		"note": "Salary"
	}`)

	// Fetch Export CSV for User 1
	req, _ := http.NewRequest("GET", "/api/v1/export/csv", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. body: %s", w.Code, w.Body.String())
	}

	if ct := w.Header().Get("Content-Type"); ct != "text/csv" {
		t.Fatalf("expected content type text/csv, got %s", ct)
	}

	reader := csv.NewReader(strings.NewReader(w.Body.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV: %v", err)
	}

	// 1 header row + 1 transaction row (User 1 only)
	if len(records) != 2 {
		t.Fatalf("expected 2 rows in CSV (header + 1 tx), got %d", len(records))
	}

	header := records[0]
	if header[0] != "ID" || header[5] != "Wallet" || header[7] != "Category" || header[8] != "Tags" {
		t.Fatalf("unexpected CSV header: %v", header)
	}

	row := records[1]
	if row[0] != txID {
		t.Fatalf("expected tx id %s, got %s", txID, row[0])
	}
	if row[2] != "5000000" {
		t.Fatalf("expected amount 5000000, got %s", row[2])
	}
	parsedTime, _ := time.Parse("2006-01-02T15:04:05Z07:00", row[3])
	if !parsedTime.Equal(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected transaction time 2025-01-01T12:00:00Z, got %s", row[3])
	}
	if row[4] != "Lunch with friends" || row[5] != "Cash wallet" || row[7] != "Food" || row[8] != "Lunch" {
		t.Fatalf("unexpected values in CSV row: %v", row)
	}

	// Fetch Export CSV with Date Filter out of range
	req, _ = http.NewRequest("GET", "/api/v1/export/csv?from=2025-02-01T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	reader = csv.NewReader(strings.NewReader(w.Body.String()))
	records, _ = reader.ReadAll()
	// Should only be header
	if len(records) != 1 {
		t.Fatalf("expected 1 row (header only) when filtering out dates, got %d", len(records))
	}

	// Fetch Export CSV for User 2
	req, _ = http.NewRequest("GET", "/api/v1/export/csv", nil)
	req.Header.Set("Authorization", "Bearer "+token2)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	reader = csv.NewReader(strings.NewReader(w.Body.String()))
	records, _ = reader.ReadAll()
	if len(records) != 2 {
		t.Fatalf("expected 2 rows for user 2, got %d", len(records))
	}
	if records[1][0] != txID2 || records[1][5] != "Bank" || records[1][2] != "10000000" {
		t.Fatalf("unexpected row for user 2: %v", records[1])
	}
}
