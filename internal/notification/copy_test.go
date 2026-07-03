package notification

import "testing"

func TestLocalizeIDTranslatesKnownRuleKeys(t *testing.T) {
	cases := map[string]string{
		RuleKeyBudgetAlert:   "Peringatan anggaran",
		RuleKeyDueReminder:   "Pengingat jatuh tempo",
		RuleKeyRecurringRun:  "Hasil transaksi berulang",
		RuleKeySecurityAlert: "Peringatan keamanan",
		RuleKeyWeeklySummary: "Ringkasan keuangan mingguan",
	}
	for key, wantTitle := range cases {
		got := LocalizeID(NotificationRule{RuleKey: key, Title: "English title", Description: "English desc"})
		if got.Title != wantTitle {
			t.Errorf("LocalizeID(%q).Title = %q, want %q", key, got.Title, wantTitle)
		}
		if got.Description == "English desc" {
			t.Errorf("LocalizeID(%q) did not translate description", key)
		}
	}
}

func TestLocalizeIDLeavesUnknownRuleUntouched(t *testing.T) {
	in := NotificationRule{RuleKey: "mystery", Title: "Keep me", Description: "Keep desc"}
	got := LocalizeID(in)
	if got.Title != "Keep me" || got.Description != "Keep desc" {
		t.Fatalf("unknown rule_key must be untouched, got %+v", got)
	}
}
