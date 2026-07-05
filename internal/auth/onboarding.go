package auth

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Onboarding defaults: every freshly registered account starts with a usable
// setup — 8 default categories (Bahasa Indonesia, informal product tone) and
// one starter cash wallet — created in the SAME transaction as the user row
// (see Repository.CreateUser), so a failure anywhere leaves no half-onboarded
// account behind.
//
// icon ids and color hexes come from the shared client catalogs
// (kCategoryIconCatalog / kWalletIconCatalog + the kItemColorPalette swatches
// mirrored by cmd/seed/main.go); the server stores them as opaque strings, so
// only catalog values may be used here or the clients render fallbacks.

type defaultCategory struct {
	Name  string
	Type  string
	Icon  string
	Color string
}

// defaultCategories are inserted with position = slice index (the category
// position convention is a 0-based per-user sort order; user-created
// categories append at MAX(position)+1, so the first custom category lands
// after these defaults).
var defaultCategories = []defaultCategory{
	{Name: "Makanan & Minuman", Type: "expense", Icon: "food", Color: "#C2553F"},
	{Name: "Transportasi", Type: "expense", Icon: "transport", Color: "#E0A23B"},
	{Name: "Belanja", Type: "expense", Icon: "shopping", Color: "#C2588A"},
	{Name: "Tagihan & Utilitas", Type: "expense", Icon: "bills", Color: "#4256B8"},
	{Name: "Hiburan", Type: "expense", Icon: "entertainment", Color: "#7C5BC2"},
	{Name: "Kesehatan", Type: "expense", Icon: "health", Color: "#2BB3A3"},
	{Name: "Gaji", Type: "income", Icon: "salary", Color: "#2E8B57"},
	{Name: "Penghasilan Lain", Type: "income", Icon: "misc", Color: "#9E7B4F"},
}

// The starter wallet every new account begins with.
const (
	defaultWalletName  = "Dompet Utama"
	defaultWalletType  = "cash"
	defaultWalletColor = "#3E72B8"
	defaultWalletIcon  = "cash"
)

// seedOnboardingDefaults inserts the default categories and starter wallet for
// a brand-new user inside the caller's transaction.
func seedOnboardingDefaults(ctx context.Context, tx pgx.Tx, userID string) error {
	for position, category := range defaultCategories {
		if _, err := tx.Exec(ctx, `
			INSERT INTO categories (user_id, name, type, icon, color, position)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, userID, category.Name, category.Type, category.Icon, category.Color, position); err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO wallets (user_id, name, type, currency_code, balance_minor, color, icon)
		VALUES ($1, $2, $3, 'IDR', 0, $4, $5)
	`, userID, defaultWalletName, defaultWalletType, defaultWalletColor, defaultWalletIcon)
	return err
}
