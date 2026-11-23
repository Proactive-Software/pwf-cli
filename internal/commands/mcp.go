package commands

import (
	pwfmcp "github.com/Proactive-Software/pwf-cli/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run ProWorkflow as an MCP server",
	Long:  "Run a Model Context Protocol server over stdio for MCP-compatible clients.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return pwfmcp.ServeStdio(cmd.InOrStdin(), cmd.OutOrStdout())
	},
}
