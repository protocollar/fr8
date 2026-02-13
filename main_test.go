package main

import (
	"testing"

	"github.com/thomascarr/fr8/cmd"
)

func TestRootCommandExists(t *testing.T) {
	root := cmd.RootCommand()
	if root == nil {
		t.Fatal("root command is nil")
	}
	if root.Use != "fr8" {
		t.Errorf("root command Use = %q, want %q", root.Use, "fr8")
	}
}

func TestMCPSubcommandRegistered(t *testing.T) {
	root := cmd.RootCommand()
	found := false
	for _, c := range root.Commands() {
		if c.Use == "mcp" {
			found = true
			// Verify serve is a subcommand of mcp
			hasServe := false
			for _, sub := range c.Commands() {
				if sub.Use == "serve" {
					hasServe = true
				}
			}
			if !hasServe {
				t.Error("mcp command missing 'serve' subcommand")
			}
		}
	}
	if !found {
		t.Error("root command missing 'mcp' subcommand")
	}
}
