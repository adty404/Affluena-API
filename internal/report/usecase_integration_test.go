package report

import (
	"context"
	"os"
	"testing"
	"time"

	"affluena-api/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// These integration tests pin the Bahasa Indonesia copy the report use case
// emits in every metric/row (labels, helpers, categories, wallet fallbacks).
// The web /reports/* pages and the mobile Laporan screen render these strings
// verbatim, so translating them is a user-facing contract — guard it.

func openReportPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("AFFLUENA_API_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AFFLUENA_API_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	t.Cleanup(pool.Close)

	migrationsDir := "../../migrations"
	if _, statErr := os.Stat("migrations"); statErr == nil {
		migrationsDir = "migrations"
	}
	if err := db.Migrate(ctx, pool, migrationsDir); err != nil {
		t.Fatalf("migrate integration database: %v", err)
	}
	return pool
}

func newReportUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	email := "report-" + time.Now().UTC().Format("20060102150405.000000000") + "@example.test"
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO users (email, password_hash) VALUES ($1, 'integration-test') RETURNING id::text
	`, email).Scan(&id); err != nil {
		t.Fatalf("create report user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

// month returns a YYYY-MM string and the [start,end) instants of the previous
// complete calendar month, so seeded rows land inside the reported window.
func prevMonth(t *testing.T) (string, time.Time) {
	t.Helper()
	now := time.Now().UTC()
	firstOfThis := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	start := firstOfThis.AddDate(0, -1, 0)
	return start.Format("2006-01"), start
}

func labelsByID(metrics []ReportMetric) map[string]ReportMetric {
	m := make(map[string]ReportMetric, len(metrics))
	for _, x := range metrics {
		m[x.ID] = x
	}
	return m
}

func TestIncomeReport_IndonesianCopy(t *testing.T) {
	pool := openReportPool(t)
	uc := NewUseCase(NewRepository(pool))
	userID := newReportUser(t, pool)
	monthStr, start := prevMonth(t)

	var walletID, catID string
	require.NoError(t, pool.QueryRow(context.Background(),
		`INSERT INTO wallets (user_id, name, type, currency_code, balance_minor) VALUES ($1,'BCA','bank','IDR',0) RETURNING id::text`,
		userID).Scan(&walletID))
	require.NoError(t, pool.QueryRow(context.Background(),
		`INSERT INTO categories (user_id, name, type) VALUES ($1,'Gaji','income') RETURNING id::text`,
		userID).Scan(&catID))
	_, err := pool.Exec(context.Background(),
		`INSERT INTO transactions (user_id, wallet_id, category_id, type, amount_minor, transaction_at, note)
		 VALUES ($1,$2,$3,'income',9000000,$4,'Gaji')`,
		userID, walletID, catID, start.AddDate(0, 0, 5))
	require.NoError(t, err)

	resp, err := uc.IncomeReport(context.Background(), userID, monthStr)
	require.NoError(t, err)

	m := labelsByID(resp.Metrics)
	require.Equal(t, "Total Pemasukan", m["total_income"].Label)
	require.Equal(t, "Bulan ini", m["total_income"].Helper)
	require.Equal(t, "Sumber Pemasukan", m["source_count"].Label)
	require.Equal(t, "Kategori berbeda", m["source_count"].Helper)
	require.Equal(t, "Rata-rata per Sumber", m["avg_per_source"].Label)
	require.Equal(t, "Sumber Teratas", m["top_source"].Label)

	require.NotEmpty(t, resp.Rows)
	require.Equal(t, "Kategori pemasukan", resp.Rows[0].Category)
}

func TestExpenseReport_IndonesianCopy(t *testing.T) {
	pool := openReportPool(t)
	uc := NewUseCase(NewRepository(pool))
	userID := newReportUser(t, pool)
	monthStr, start := prevMonth(t)

	var walletID, catID string
	require.NoError(t, pool.QueryRow(context.Background(),
		`INSERT INTO wallets (user_id, name, type, currency_code, balance_minor) VALUES ($1,'Cash','cash','IDR',0) RETURNING id::text`,
		userID).Scan(&walletID))
	require.NoError(t, pool.QueryRow(context.Background(),
		`INSERT INTO categories (user_id, name, type) VALUES ($1,'Makan','expense') RETURNING id::text`,
		userID).Scan(&catID))
	_, err := pool.Exec(context.Background(),
		`INSERT INTO transactions (user_id, wallet_id, category_id, type, amount_minor, transaction_at, note)
		 VALUES ($1,$2,$3,'expense',150000,$4,'Makan siang')`,
		userID, walletID, catID, start.AddDate(0, 0, 5))
	require.NoError(t, err)

	resp, err := uc.ExpenseReport(context.Background(), userID, monthStr)
	require.NoError(t, err)

	m := labelsByID(resp.Metrics)
	require.Equal(t, "Total Pengeluaran", m["total_expense"].Label)
	require.Equal(t, "Bulan ini", m["total_expense"].Helper)
	require.Equal(t, "Kategori Pengeluaran", m["cat_count"].Label)
	require.Equal(t, "Rata-rata per Kategori", m["avg_per_cat"].Label)
	require.Equal(t, "Kategori Teratas", m["top_category"].Label)

	require.NotEmpty(t, resp.Rows)
	require.Equal(t, "Kategori pengeluaran", resp.Rows[0].Category)
}

func TestCashflowReport_IndonesianCopy(t *testing.T) {
	pool := openReportPool(t)
	uc := NewUseCase(NewRepository(pool))
	userID := newReportUser(t, pool)
	monthStr, _ := prevMonth(t)

	resp, err := uc.CashflowReport(context.Background(), userID, monthStr)
	require.NoError(t, err)

	m := labelsByID(resp.Metrics)
	require.Equal(t, "Arus Kas Bersih", m["net_cashflow"].Label)
	require.Equal(t, "Pemasukan − Pengeluaran", m["net_cashflow"].Helper)
	require.Equal(t, "Total Pemasukan", m["total_income"].Label)
	require.Equal(t, "Total Pengeluaran", m["total_expense"].Label)
	require.Equal(t, "Rasio Menabung", m["saving_rate"].Label)
	require.Equal(t, "% pemasukan yang ditabung", m["saving_rate"].Helper)

	// Cashflow always renders 4 week rows even with no data.
	require.Len(t, resp.Rows, 4)
	require.Equal(t, "Arus Kas Minggu 1", resp.Rows[0].Name)
	require.Equal(t, "Ringkasan mingguan", resp.Rows[0].Category)
	require.Equal(t, "Semua dompet", resp.Rows[0].Wallet)
}

func TestDebtReport_IndonesianCopy(t *testing.T) {
	pool := openReportPool(t)
	uc := NewUseCase(NewRepository(pool))
	userID := newReportUser(t, pool)
	monthStr, _ := prevMonth(t)

	var walletID, disbCatID, payCatID, txID string
	require.NoError(t, pool.QueryRow(context.Background(),
		`INSERT INTO wallets (user_id, name, type, currency_code, balance_minor) VALUES ($1,'BCA','bank','IDR',0) RETURNING id::text`,
		userID).Scan(&walletID))
	require.NoError(t, pool.QueryRow(context.Background(),
		`INSERT INTO categories (user_id, name, type) VALUES ($1,'Pinjaman','expense') RETURNING id::text`,
		userID).Scan(&disbCatID))
	require.NoError(t, pool.QueryRow(context.Background(),
		`INSERT INTO categories (user_id, name, type) VALUES ($1,'Bayar utang','income') RETURNING id::text`,
		userID).Scan(&payCatID))
	require.NoError(t, pool.QueryRow(context.Background(),
		`INSERT INTO transactions (user_id, wallet_id, category_id, type, amount_minor, transaction_at, note)
		 VALUES ($1,$2,$3,'expense',5000000, now(),'Pencairan') RETURNING id::text`,
		userID, walletID, disbCatID).Scan(&txID))
	_, err := pool.Exec(context.Background(),
		`INSERT INTO debts (user_id, type, counterparty_name, wallet_id, disbursement_category_id, payment_category_id, origination_transaction_id, principal_amount_minor, paid_amount_minor, opened_at, due_date, status)
		 VALUES ($1,'payable','Bank X',$2,$3,$4,$5,5000000,1000000, now(), (now() + interval '10 days')::date, 'open'),
		        ($1,'receivable','Teman',$2,$3,$4,$5,2000000,0, now(), (now() + interval '10 days')::date, 'open')`,
		userID, walletID, disbCatID, payCatID, txID)
	require.NoError(t, err)

	resp, err := uc.DebtReport(context.Background(), userID, monthStr)
	require.NoError(t, err)

	m := labelsByID(resp.Metrics)
	require.Equal(t, "Total Utang", m["total_payable"].Label)
	require.Equal(t, "Sisa yang harus dibayar", m["total_payable"].Helper)
	require.Equal(t, "Total Piutang", m["total_receivable"].Label)
	require.Equal(t, "Sisa yang akan diterima", m["total_receivable"].Helper)
	require.Equal(t, "Utang Aktif", m["open_count"].Label)
	require.Equal(t, "Terlambat", m["overdue_count"].Label)

	cats := map[string]bool{}
	for _, r := range resp.Rows {
		cats[r.Category] = true
	}
	require.True(t, cats["Utang"], "expected a Utang row")
	require.True(t, cats["Piutang"], "expected a Piutang row")
}

func TestGoalReport_IndonesianCopy(t *testing.T) {
	pool := openReportPool(t)
	uc := NewUseCase(NewRepository(pool))
	userID := newReportUser(t, pool)
	monthStr, _ := prevMonth(t)

	_, err := pool.Exec(context.Background(),
		`INSERT INTO goals (user_id, name, target_amount_minor, deadline, status)
		 VALUES ($1,'Liburan',50000000,(now() + interval '365 days')::date,'active')`,
		userID)
	require.NoError(t, err)

	resp, err := uc.GoalReport(context.Background(), userID, monthStr)
	require.NoError(t, err)

	m := labelsByID(resp.Metrics)
	require.Equal(t, "Total Ditabung", m["total_saved"].Label)
	require.Equal(t, "Seluruh target", m["total_saved"].Helper)
	require.Equal(t, "Total Target", m["total_target"].Label)
	require.Equal(t, "Progres Keseluruhan %", m["overall_progress"].Label)
	require.Equal(t, "Target Aktif", m["active_count"].Label)

	require.NotEmpty(t, resp.Rows)
	require.Equal(t, "Target", resp.Rows[0].Category)
}
