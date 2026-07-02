package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"affluena-api/internal/config"
)

// TestEntityColorAndIconRoundTrip proves budgets, goals, installments,
// subscriptions, and recurring rules persist and return the optional UI
// metadata (color, icon) on create and update, mirroring the wallet behavior
// covered by TestWalletColorAndIconRoundTrip.
func TestEntityColorAndIconRoundTrip(t *testing.T) {
	pool := openServerIntegrationPool(t)
	router := NewRouter(config.Config{
		Env:                  "production",
		JWTSecret:            "entity-color-icon-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
	}, pool)

	userID, token := registerIntegrationAPIUser(t, router, "entity-color-icon")
	defer cleanupServerIntegrationUsers(t, pool, userID)

	walletID := createAPIResource(t, router, token, "/api/v1/wallets", `{
		"name": "Dompet metadata",
		"type": "bank",
		"currency_code": "IDR",
		"balance_minor": 10000000
	}`)
	categoryID := createAPIResource(t, router, token, "/api/v1/categories", `{
		"name": "Kategori metadata",
		"type": "expense"
	}`)

	assertColorIcon := func(path string, wantColor string, wantIcon string, context string) {
		t.Helper()
		body := performAPIRequest(t, router, token, http.MethodGet, path, "", http.StatusOK)
		var got struct {
			Color string `json:"color"`
			Icon  string `json:"icon"`
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode %s: %v", context, err)
		}
		if got.Color != wantColor || got.Icon != wantIcon {
			t.Fatalf("color/icon did not round-trip on %s: got color=%q icon=%q, want color=%q icon=%q", context, got.Color, got.Icon, wantColor, wantIcon)
		}
	}

	// Budget.
	budgetID := createAPIResource(t, router, token, "/api/v1/category-budgets", `{
		"category_id": "`+categoryID+`",
		"month": "2026-07",
		"limit_minor": 2000000,
		"color": "#E67E22",
		"icon": "food"
	}`)
	assertColorIcon("/api/v1/category-budgets/"+budgetID, "#E67E22", "food", "budget create")
	performAPIRequest(t, router, token, http.MethodPut, "/api/v1/category-budgets/"+budgetID, `{
		"category_id": "`+categoryID+`",
		"month": "2026-07",
		"limit_minor": 2500000,
		"color": "#C0392B",
		"icon": "groceries"
	}`, http.StatusOK)
	assertColorIcon("/api/v1/category-budgets/"+budgetID, "#C0392B", "groceries", "budget update")

	// Goal.
	goalID := createAPIResource(t, router, token, "/api/v1/goals", `{
		"name": "Dana darurat",
		"target_amount_minor": 5000000,
		"deadline": "2026-12-31T00:00:00Z",
		"color": "#F1C40F",
		"icon": "emergency"
	}`)
	assertColorIcon("/api/v1/goals/"+goalID, "#F1C40F", "emergency", "goal create")
	performAPIRequest(t, router, token, http.MethodPut, "/api/v1/goals/"+goalID, `{
		"name": "Dana darurat",
		"target_amount_minor": 5000000,
		"deadline": "2026-12-31T00:00:00Z",
		"color": "#F39C12",
		"icon": "vacation"
	}`, http.StatusOK)
	assertColorIcon("/api/v1/goals/"+goalID, "#F39C12", "vacation", "goal update")

	// Installment.
	installmentID := createAPIResource(t, router, token, "/api/v1/installments", `{
		"name": "Cicilan metadata",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"total_amount_minor": 900000,
		"monthly_amount_minor": 300000,
		"tenor_months": 3,
		"start_date": "2026-06-01",
		"due_day": 5,
		"color": "#3E72B8",
		"icon": "gym"
	}`)
	assertColorIcon("/api/v1/installments/"+installmentID, "#3E72B8", "gym", "installment create")
	performAPIRequest(t, router, token, http.MethodPut, "/api/v1/installments/"+installmentID, `{
		"name": "Cicilan metadata",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"total_amount_minor": 900000,
		"monthly_amount_minor": 300000,
		"tenor_months": 3,
		"remaining_months": 3,
		"start_date": "2026-06-01",
		"due_day": 5,
		"status": "active",
		"color": "#2E8B57",
		"icon": "phone"
	}`, http.StatusOK)
	assertColorIcon("/api/v1/installments/"+installmentID, "#2E8B57", "phone", "installment update")

	// Subscription.
	subscriptionID := createAPIResource(t, router, token, "/api/v1/subscriptions", `{
		"name": "Langganan metadata",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 250000,
		"billing_cycle": "monthly",
		"next_due_date": "2026-07-12",
		"color": "#8E44AD",
		"icon": "streaming"
	}`)
	assertColorIcon("/api/v1/subscriptions/"+subscriptionID, "#8E44AD", "streaming", "subscription create")
	performAPIRequest(t, router, token, http.MethodPut, "/api/v1/subscriptions/"+subscriptionID, `{
		"name": "Langganan metadata",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 250000,
		"billing_cycle": "monthly",
		"next_due_date": "2026-07-12",
		"status": "active",
		"color": "#9B59B6",
		"icon": "cloud"
	}`, http.StatusOK)
	assertColorIcon("/api/v1/subscriptions/"+subscriptionID, "#9B59B6", "cloud", "subscription update")

	// Recurring rule.
	recurringID := createAPIResource(t, router, token, "/api/v1/recurring-transactions", `{
		"name": "Rutin metadata",
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 350000,
		"frequency": "monthly",
		"interval_count": 1,
		"next_run_at": "2026-08-01T00:00:00Z",
		"color": "#16A085",
		"icon": "internet"
	}`)
	assertColorIcon("/api/v1/recurring-transactions/"+recurringID, "#16A085", "internet", "recurring create")
	performAPIRequest(t, router, token, http.MethodPut, "/api/v1/recurring-transactions/"+recurringID, `{
		"name": "Rutin metadata",
		"type": "expense",
		"wallet_id": "`+walletID+`",
		"category_id": "`+categoryID+`",
		"amount_minor": 350000,
		"frequency": "monthly",
		"interval_count": 1,
		"next_run_at": "2026-08-01T00:00:00Z",
		"status": "active",
		"color": "#1ABC9C",
		"icon": "electricity"
	}`, http.StatusOK)
	assertColorIcon("/api/v1/recurring-transactions/"+recurringID, "#1ABC9C", "electricity", "recurring update")
}
