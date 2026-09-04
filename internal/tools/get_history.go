package tools

import (
	"context"
	"fmt"

	"github.com/edouard-claude/redmine-mcp/internal/redmine"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetHistory(s *server.MCPServer, client *redmine.Client) {
	tool := mcp.NewTool("get_history",
		mcp.WithDescription("Get the change history of a Redmine issue: every journal entry with the fields that changed (status, assignee, version, priority, dates, attachments, relations…) plus the comment attached to it. Numeric IDs are resolved to names. Use this to see who changed what and when; use get_comments for notes only."),
		mcp.WithNumber("issue_id",
			mcp.Description("Redmine issue number (e.g. 5871)"),
			mcp.Required(),
		),
		mcp.WithNumber("limit",
			mcp.Description("Keep only the N most recent entries (default: 0 = full history)"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		issueID := req.GetInt("issue_id", 0)
		if issueID == 0 {
			return mcp.NewToolResultError("issue_id is required"), nil
		}
		limit := req.GetInt("limit", 0)

		issue, err := client.GetIssue(issueID, "journals")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get history: %v", err)), nil
		}

		return mcp.NewToolResultText(FormatHistory(client, issue, limit)), nil
	})
}
