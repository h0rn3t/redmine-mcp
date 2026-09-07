package tools

import (
	"strings"
	"testing"

	"github.com/h0rn3t/redmine-mcp/internal/redmine"
)

func TestFormatTimeEntries(t *testing.T) {
	entries := []redmine.TimeEntry{
		{Issue: &redmine.IDRef{ID: 7415}, User: redmine.IDName{Name: "Іван Петренко"},
			Activity: redmine.IDName{Name: "Development"}, Hours: 2.5,
			Comments: "правка міграції", SpentOn: "2026-09-05"},
		{Issue: &redmine.IDRef{ID: 7415}, User: redmine.IDName{Name: "Іван Петренко"},
			Activity: redmine.IDName{Name: "Development"}, Hours: 1.5, SpentOn: "2026-09-04"},
		{Project: redmine.IDName{Name: "APNL"}, User: redmine.IDName{Name: "Олена Ко"},
			Activity: redmine.IDName{Name: "Analysis"}, Hours: 1,
			Comments: "аналіз логів", SpentOn: "2026-09-04"},
	}

	got := FormatTimeEntries(entries, 3, 0, "issue #7415")

	for _, want := range []string{
		"## Time entries — issue #7415",
		"3 entry(s) shown | Total: 5h",
		"- Іван Петренко — 4h (2 entry(s))",
		"- Олена Ко — 1h (1 entry(s))",
		"- Development — 4h (2 entry(s))",
		"- #7415 — 4h (2 entry(s))",
		"- project APNL — 1h (1 entry(s))",
		"- 2026-09-05 | 2.5h | Іван Петренко | Development | #7415",
		"  правка міграції",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n--- got ---\n%s", want, got)
		}
	}
}

func TestFormatTimeEntriesEmpty(t *testing.T) {
	if got, want := FormatTimeEntries(nil, 0, 0, "user Олена Ко"), "No time entries for user Олена Ко."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatTimeEntriesTruncatedPage(t *testing.T) {
	entries := []redmine.TimeEntry{{Issue: &redmine.IDRef{ID: 1}, Hours: 1, SpentOn: "2026-09-05"}}
	if got := FormatTimeEntries(entries, 40, 0, "project apnl"); !strings.Contains(got, "1 entry(s) shown of 40 (offset 0)") {
		t.Errorf("missing pagination note:\n%s", got)
	}
}
