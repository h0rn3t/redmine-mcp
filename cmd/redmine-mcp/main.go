package main

import (
	"fmt"
	"os"

	"github.com/h0rn3t/redmine-mcp/internal/cli"
	"github.com/h0rn3t/redmine-mcp/internal/redmine"
	"github.com/h0rn3t/redmine-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/server"
)

var version = "2.0.0"

func main() {
	os.Exit(run(os.Args[1:]))
}

// run dispatches between MCP stdio mode (default, or explicit "mcp"/"serve")
// and CLI mode (any other subcommand). Returns the process exit code.
func run(args []string) int {
	client, err := redmine.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "redmine client: %v\n", err)
		return 1
	}

	if len(args) == 0 || args[0] == "mcp" || args[0] == "serve" {
		return runMCP(client)
	}
	return cli.Run(args, client)
}

func runMCP(client *redmine.Client) int {
	s := server.NewMCPServer(
		"redmine-mcp",
		version,
		server.WithInstructions(`Redmine access via REST API. Query and manage issues, comments, attachments, and projects.

When reading an issue with get_issue, if the issue contains a substantial description or requirements that imply implementation work, suggest entering plan mode to design the approach before coding.

Before creating a large number of issues, consider using plan mode to batch them.
`),
	)

	tools.RegisterAll(s, client)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
