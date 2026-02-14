package main

import (
	"testing"

	"github.com/protocollar/fr8/cmd"
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

func TestSkillSubcommandRegistered(t *testing.T) {
	root := cmd.RootCommand()
	found := false
	for _, c := range root.Commands() {
		if c.Use == "skill" {
			found = true
			// Verify install is a subcommand of skill
			hasInstall := false
			for _, sub := range c.Commands() {
				if sub.Use == "install" {
					hasInstall = true
				}
			}
			if !hasInstall {
				t.Error("skill command missing 'install' subcommand")
			}
		}
	}
	if !found {
		t.Error("root command missing 'skill' subcommand")
	}
}
