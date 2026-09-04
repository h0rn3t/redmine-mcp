package tools

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/edouard-claude/redmine-mcp/internal/redmine"
)

func FormatIssue(issue *redmine.Issue, maxDesc int) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Issue #%d — %s\n", issue.ID, issue.Subject)
	fmt.Fprintf(&b, "Project: %s | Tracker: %s | Priority: %s\n", issue.Project.Name, issue.Tracker.Name, issue.Priority.Name)

	statusLabel := "open"
	if issue.Status.IsClosed {
		statusLabel = "closed"
	}
	fmt.Fprintf(&b, "Status: %s (%s) | Done: %d%%\n", issue.Status.Name, statusLabel, issue.DoneRatio)

	fmt.Fprintf(&b, "Author: %s", issue.Author.Name)
	if issue.AssignedTo != nil {
		fmt.Fprintf(&b, " | Assignee: %s", issue.AssignedTo.Name)
	}
	b.WriteString("\n")

	if issue.FixedVersion != nil {
		fmt.Fprintf(&b, "Version: %s\n", issue.FixedVersion.Name)
	}

	fmt.Fprintf(&b, "Created: %s | Updated: %s\n", formatDateTime(issue.CreatedOn), formatDateTime(issue.UpdatedOn))

	if issue.StartDate != nil || issue.DueDate != nil {
		if issue.StartDate != nil {
			fmt.Fprintf(&b, "Start: %s", *issue.StartDate)
		}
		if issue.DueDate != nil {
			if issue.StartDate != nil {
				b.WriteString(" | ")
			}
			fmt.Fprintf(&b, "Due: %s", *issue.DueDate)
		}
		b.WriteString("\n")
	}

	if issue.Parent != nil {
		fmt.Fprintf(&b, "Parent: #%d\n", issue.Parent.ID)
	}

	if hours := formatHours(issue); hours != "" {
		b.WriteString(hours)
		b.WriteString("\n")
	}

	if n := len(issue.Journals); n > 0 {
		fmt.Fprintf(&b, "History: %d journal entry(s) — use get_history for the change log\n", n)
	}

	if issue.Description != "" {
		desc := issue.Description
		if cut, ok := truncateRunes(desc, maxDesc); ok {
			desc = cut + "\n[truncated]"
		}
		fmt.Fprintf(&b, "\n## Description\n%s\n", desc)
	}

	return b.String()
}

func FormatIssueSummaries(issues []redmine.Issue, offset int) string {
	if len(issues) == 0 {
		return "No issues found."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Found %d issue(s) (offset %d):\n\n", len(issues), offset)

	for _, issue := range issues {
		fmt.Fprintf(&b, "#%d | %s | %s | %s\n", issue.ID, issue.Status.Name, issue.Tracker.Name, issue.Subject)
		fmt.Fprintf(&b, "  Project: %s", issue.Project.Name)
		if issue.AssignedTo != nil {
			fmt.Fprintf(&b, " | Assignee: %s", issue.AssignedTo.Name)
		}
		fmt.Fprintf(&b, " | %d%% | Updated: %s", issue.DoneRatio, formatDateTime(issue.UpdatedOn))
		if issue.SpentHours != nil || issue.EstimatedHours != nil {
			b.WriteString(" |")
			if issue.EstimatedHours != nil {
				fmt.Fprintf(&b, " est %sh", trimFloat(*issue.EstimatedHours))
			}
			if issue.SpentHours != nil {
				fmt.Fprintf(&b, " spent %sh", trimFloat(*issue.SpentHours))
			}
		}
		b.WriteString("\n\n")
	}

	return b.String()
}

func FormatComments(issueID int, journals []redmine.Journal) string {
	// filter to journals with notes only
	var comments []redmine.Journal
	for _, j := range journals {
		if j.Notes != "" {
			comments = append(comments, j)
		}
	}

	if len(comments) == 0 {
		return fmt.Sprintf("No comments for issue #%d.", issueID)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Comments for #%d (%d comment(s))\n\n", issueID, len(comments))

	for i, c := range comments {
		fmt.Fprintf(&b, "### Comment #%d (journal_id: %d) by %s — %s\n%s\n\n", i+1, c.ID, c.User.Name, formatDateTime(c.CreatedOn), c.Notes)
	}

	return b.String()
}

// FormatHistory renders the full journal of an issue — field changes and notes,
// oldest first. limit > 0 keeps only the most recent entries. The client is used
// to resolve numeric IDs (status, assignee, version, …) to names.
func FormatHistory(client *redmine.Client, issue *redmine.Issue, limit int) string {
	if len(issue.Journals) == 0 {
		return fmt.Sprintf("No history for issue #%d.", issue.ID)
	}

	journals := issue.Journals
	hidden := 0
	if limit > 0 && len(journals) > limit {
		hidden = len(journals) - limit
		journals = journals[hidden:]
	}
	projectID := strconv.Itoa(issue.Project.ID)

	var b strings.Builder
	fmt.Fprintf(&b, "## History of #%d — %s\n", issue.ID, issue.Subject)
	fmt.Fprintf(&b, "%d entry(s)", len(journals))
	if hidden > 0 {
		fmt.Fprintf(&b, ", %d older entry(s) hidden", hidden)
	}
	b.WriteString("\n\n")

	fmt.Fprintf(&b, "### Created — %s by %s\n\n", formatDateTime(issue.CreatedOn), issue.Author.Name)

	for i, j := range journals {
		fmt.Fprintf(&b, "### %d. %s — %s (journal_id: %d)\n", hidden+i+1, j.User.Name, formatDateTime(j.CreatedOn), j.ID)
		for _, d := range j.Details {
			fmt.Fprintf(&b, "- %s\n", formatDetail(client, d, projectID))
		}
		if j.Notes != "" {
			if len(j.Details) > 0 {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "%s\n", j.Notes)
		}
		b.WriteString("\n")
	}

	return b.String()
}

// formatDetail renders one field change from a journal entry.
func formatDetail(client *redmine.Client, d redmine.JournalDetail, projectID string) string {
	switch d.Property {
	case "attachment":
		if d.NewValue != "" {
			return fmt.Sprintf("Attachment added: %s", d.NewValue)
		}
		return fmt.Sprintf("Attachment removed: %s", d.OldValue)
	case "relation":
		if d.NewValue != "" {
			return fmt.Sprintf("Relation added: %s #%s", d.Name, d.NewValue)
		}
		return fmt.Sprintf("Relation removed: %s #%s", d.Name, d.OldValue)
	case "cf":
		return fmt.Sprintf("Custom field #%s: %s → %s", d.Name,
			detailValue(client, "", d.OldValue, projectID),
			detailValue(client, "", d.NewValue, projectID))
	}

	return fmt.Sprintf("%s: %s → %s", attrLabel(d.Name),
		detailValue(client, d.Name, d.OldValue, projectID),
		detailValue(client, d.Name, d.NewValue, projectID))
}

// detailValue resolves a raw journal value to something readable: a reference
// name when the field is an ID, otherwise the value itself (shortened).
func detailValue(client *redmine.Client, field, raw, projectID string) string {
	if raw == "" {
		return "(none)"
	}
	if name := client.LookupName(field, raw, projectID); name != "" {
		return name
	}
	switch field {
	case "parent_id":
		return "#" + raw
	case "done_ratio":
		return raw + "%"
	}
	return shorten(raw, 200)
}

var attrLabels = map[string]string{
	"subject":          "Subject",
	"description":      "Description",
	"status_id":        "Status",
	"priority_id":      "Priority",
	"tracker_id":       "Tracker",
	"assigned_to_id":   "Assignee",
	"category_id":      "Category",
	"fixed_version_id": "Version",
	"parent_id":        "Parent",
	"project_id":       "Project",
	"start_date":       "Start date",
	"due_date":         "Due date",
	"done_ratio":       "Done",
	"estimated_hours":  "Estimated hours",
	"is_private":       "Private",
}

// attrLabel maps a Redmine attribute name to a display label, falling back to a
// prettified version of the raw name for fields we don't know about.
func attrLabel(name string) string {
	if label, ok := attrLabels[name]; ok {
		return label
	}
	label := strings.ReplaceAll(strings.TrimSuffix(name, "_id"), "_", " ")
	if label == "" {
		return name
	}
	return strings.ToUpper(label[:1]) + label[1:]
}

// shorten collapses a multi-line value (a description edit, typically) onto one
// line and truncates it, so a history entry stays one readable row.
func shorten(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if cut, ok := truncateRunes(s, max); ok {
		return cut + "…"
	}
	return s
}

// truncateRunes cuts s to max characters, reporting whether it cut anything.
// max <= 0 means no limit. Cutting on runes keeps multi-byte text (Cyrillic,
// accents, emoji) from being sliced into an invalid byte sequence.
func truncateRunes(s string, max int) (string, bool) {
	if max <= 0 {
		return s, false
	}
	r := []rune(s)
	if len(r) <= max {
		return s, false
	}
	return string(r[:max]), true
}

func FormatProjects(projects []redmine.Project) string {
	if len(projects) == 0 {
		return "No projects found."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Projects (%d)\n\n", len(projects))

	for _, p := range projects {
		fmt.Fprintf(&b, "- **%s** (`%s`)", p.Name, p.Identifier)
		if p.Description != "" {
			desc := p.Description
			if cut, ok := truncateRunes(desc, 100); ok {
				desc = cut + "..."
			}
			fmt.Fprintf(&b, " — %s", desc)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func FormatAttachments(issueID int, attachments []redmine.Attachment) string {
	if len(attachments) == 0 {
		return fmt.Sprintf("No attachments for issue #%d.", issueID)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Attachments for #%d (%d file(s))\n\n", issueID, len(attachments))

	for i, a := range attachments {
		fmt.Fprintf(&b, "%d. **%s** (%s, %s)\n", i+1, a.Filename, formatSize(a.Filesize), a.ContentType)
		fmt.Fprintf(&b, "   By %s — %s\n", a.Author.Name, formatDateTime(a.CreatedOn))
		if a.Description != "" {
			fmt.Fprintf(&b, "   Description: %s\n", a.Description)
		}
		if a.ContentURL != "" {
			fmt.Fprintf(&b, "   URL: %s\n", a.ContentURL)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func FormatChildren(issueID int, children []redmine.ChildIssue) string {
	if len(children) == 0 {
		return fmt.Sprintf("No subtasks for issue #%d.", issueID)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Subtasks of #%d (%d issue(s))\n\n", issueID, len(children))

	for _, c := range children {
		fmt.Fprintf(&b, "#%d | %s | %s\n", c.ID, c.Tracker.Name, c.Subject)
	}

	return b.String()
}

// formatHours renders estimated/spent hours, including totals across subtasks
// when they differ from the issue's own values.
func formatHours(issue *redmine.Issue) string {
	parts := []string{}
	if issue.EstimatedHours != nil {
		parts = append(parts, fmt.Sprintf("Estimated: %sh", trimFloat(*issue.EstimatedHours)))
	}
	if issue.SpentHours != nil {
		parts = append(parts, fmt.Sprintf("Spent: %sh", trimFloat(*issue.SpentHours)))
	}
	if issue.TotalEstimatedHours != nil && (issue.EstimatedHours == nil || *issue.TotalEstimatedHours != *issue.EstimatedHours) {
		parts = append(parts, fmt.Sprintf("Total estimated (incl. subtasks): %sh", trimFloat(*issue.TotalEstimatedHours)))
	}
	if issue.TotalSpentHours != nil && (issue.SpentHours == nil || *issue.TotalSpentHours != *issue.SpentHours) {
		parts = append(parts, fmt.Sprintf("Total spent (incl. subtasks): %sh", trimFloat(*issue.TotalSpentHours)))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Hours — " + strings.Join(parts, " | ")
}

func trimFloat(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

// formatDateTime trims the Redmine ISO timestamp to a readable format.
func formatDateTime(s string) string {
	if len(s) >= 16 {
		return s[:10] + " " + s[11:16]
	}
	return s
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
