package notification

// Indonesian (id_ID) copy for the five seeded notification rules, keyed on the
// stable rule_key. The DB seeds English title/description (EnsureDefaults in
// repository.go / the migration), so we localize server-side at read time: the
// settings screen shows Indonesian without a data migration, and the copy stays
// in one place. Unknown rule_keys fall through to the stored title/description.
type ruleCopy struct {
	Title       string
	Description string
}

var ruleCopyID = map[string]ruleCopy{
	"budget-alert": {
		Title:       "Peringatan anggaran",
		Description: "Beri tahu saat pemakaian anggaran kategori mencapai 80% dan 100%.",
	},
	"due-reminder": {
		Title:       "Pengingat jatuh tempo",
		Description: "Pengingat utang, cicilan, dan langganan pada H-3 dan H-1.",
	},
	"recurring-run": {
		Title:       "Hasil transaksi berulang",
		Description: "Beri tahu saat aturan transaksi berulang dijalankan atau gagal.",
	},
	"security-alert": {
		Title:       "Peringatan keamanan",
		Description: "Beri tahu saat login dari perangkat atau lokasi baru.",
	},
	"weekly-summary": {
		Title:       "Ringkasan keuangan mingguan",
		Description: "Kirim ringkasan arus kas mingguan.",
	},
}

// LocalizeID overwrites a rule's Title/Description with the Indonesian copy for
// its rule_key when one exists; otherwise the stored (English) values are kept.
func LocalizeID(rule NotificationRule) NotificationRule {
	if c, ok := ruleCopyID[rule.RuleKey]; ok {
		rule.Title = c.Title
		rule.Description = c.Description
	}
	return rule
}
