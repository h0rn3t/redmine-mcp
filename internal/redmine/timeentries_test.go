package redmine

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const timeEntriesJSON = `{"time_entries":[
  {"id":1,"project":{"id":7,"name":"APNL"},"issue":{"id":7415},"user":{"id":42,"name":"Іван Петренко"},
   "activity":{"id":9,"name":"Development"},"hours":2.5,"comments":"правка міграції","spent_on":"2026-09-05"},
  {"id":2,"project":{"id":7,"name":"APNL"},"user":{"id":43,"name":"Олена Ко"},
   "activity":{"id":8,"name":"Analysis"},"hours":1,"comments":"","spent_on":"2026-09-04"}
],"total_count":2,"offset":0,"limit":25}`

func TestListTimeEntries(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/time_entries.json" {
			t.Errorf("path = %q, want /time_entries.json", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		w.Write([]byte(timeEntriesJSON))
	}))
	defer srv.Close()

	t.Setenv("REDMINE_URL", srv.URL)
	t.Setenv("REDMINE_API_KEY", "test-key")
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	entries, total, err := client.ListTimeEntries(TimeEntryListParams{
		IssueID: "7415", UserID: "42", From: "2026-09-01", To: "2026-09-07", Limit: 50,
	})
	if err != nil {
		t.Fatalf("ListTimeEntries: %v", err)
	}

	want := "from=2026-09-01&issue_id=7415&limit=50&to=2026-09-07&user_id=42"
	if gotQuery != want {
		t.Errorf("query = %q, want %q", gotQuery, want)
	}
	if total != 2 || len(entries) != 2 {
		t.Fatalf("got %d entries (total %d), want 2 (total 2)", len(entries), total)
	}

	e := entries[0]
	if e.Hours != 2.5 || e.User.Name != "Іван Петренко" || e.Activity.Name != "Development" ||
		e.Comments != "правка міграції" || e.SpentOn != "2026-09-05" {
		t.Errorf("entry[0] decoded as %+v", e)
	}
	if e.Issue == nil || e.Issue.ID != 7415 {
		t.Errorf("entry[0].Issue = %+v, want #7415", e.Issue)
	}
	if entries[1].Issue != nil {
		t.Errorf("entry[1].Issue = %+v, want nil for project-level time", entries[1].Issue)
	}
}
