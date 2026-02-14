package cmd

import (
	"io"

	"github.com/spf13/cobra"
	fr8mcp "github.com/protocollar/fr8/internal/mcp"
	"github.com/protocollar/fr8/internal/jsonout"
)

func init() {
	mcpCmd.AddCommand(mcpServeCmd)
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server on stdio",
	Args:  cobra.NoArgs,
	RunE:  runMCPServe,
}

func runMCPServe(cmd *cobra.Command, args []string) error {
	// Suppress all human progress messages — stdout is used by MCP protocol
	jsonout.SetMsgOut(io.Discard)

	s := fr8mcp.NewServer(Version)
	registerMCPTools(s)
	return fr8mcp.Serve(s)
}
