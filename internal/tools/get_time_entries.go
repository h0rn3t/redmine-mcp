package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/h0rn3t/redmine-mcp/internal/redmine"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetTimeEntries(s *server.MCPServer, client *redmine.Client) {
	tool := mcp.NewTool("get_time_entries",
		mcp.WithDescription("Get logged time (spent time) from Redmine: how many hours each person logged, on which issue, under which activity, and the comment they left. Filter by issue, project, user and date range. Returns totals per person, per activity and per issue plus every individual entry. Use this when get_issue only shows an aggregate 'Spent' figure and you need the breakdown."),
		mcp.WithNumber("issue_id",
			mcp.Description("Only time logged on this issue (e.g. 7415)"),
		),
		mcp.WithString("project",
			mcp.Description("Only time logged in this project (identifier, e.g. 'apnl')"),
		),
		mcp.WithString("user",
			mcp.Description("Only time logged by this person: name, login, numeric ID, or 'me'"),
		),
		mcp.WithString("from",
			mcp.Description("Range start, inclusive (YYYY-MM-DD)"),
		),
		mcp.WithString("to",
			mcp.Description("Range end, inclusive (YYYY-MM-DD)"),
		),
		mcp.WithString("spent_on",
			mcp.Description("Exact date (YYYY-MM-DD) — alternative to from/to"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max entries (default: 100, max: 100)"),
		),
		mcp.WithNumber("offset",
			mcp.Description("Offset for pagination (default: 0)"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		offset := req.GetInt("offset", 0)
		limit := req.GetInt("limit", 100)
		if limit > 100 {
			limit = 100
		}

		params, scope, err := BuildTimeEntryParams(client,
			req.GetInt("issue_id", 0),
			req.GetString("project", ""),
			req.GetString("user", ""),
			req.GetString("spent_on", ""),
			req.GetString("from", ""),
			req.GetString("to", ""),
			limit, offset)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("filter error: %v", err)), nil
		}

		entries, total, err := client.ListTimeEntries(params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get time entries: %v", err)), nil
		}

		return mcp.NewToolResultText(FormatTimeEntries(entries, total, offset, scope)), nil
	})
}

// BuildTimeEntryParams builds a TimeEntryListParams from human-readable filters,
// resolving a user name to a numeric ID. It also returns a description of the
// applied filters, used as the report heading.
func BuildTimeEntryParams(client *redmine.Client, issueID int, project, user, spentOn, from, to string, limit, offset int) (redmine.TimeEntryListParams, string, error) {
	params := redmine.TimeEntryListParams{
		ProjectID: project,
		SpentOn:   spentOn,
		From:      from,
		To:        to,
		Limit:     limit,
		Offset:    offset,
	}

	var scope []string
	if issueID > 0 {
		params.IssueID = strconv.Itoa(issueID)
		scope = append(scope, fmt.Sprintf("issue #%d", issueID))
	}
	if project != "" {
		scope = append(scope, "project "+project)
	}
	if user != "" {
		resolved, err := client.ResolveUserID(user)
		if err != nil {
			return params, "", err
		}
		params.UserID = resolved
		scope = append(scope, "user "+user)
	}
	if spentOn != "" {
		scope = append(scope, "on "+spentOn)
	}
	if from != "" {
		scope = append(scope, "from "+from)
	}
	if to != "" {
		scope = append(scope, "to "+to)
	}

	return params, strings.Join(scope, ", "), nil
}
