package mcp

import (
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates a new fr8 MCP server.
func NewServer(version string) *server.MCPServer {
	return server.NewMCPServer(
		"fr8",
		version,
		server.WithToolCapabilities(false),
	)
}

// Serve starts the MCP server on stdio.
func Serve(s *server.MCPServer) error {
	return server.ServeStdio(s)
}
