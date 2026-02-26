package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFr8Json(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fr8.json"), []byte(`{
		"scripts": {"setup": "make setup", "run": "make run", "archive": "make clean"},
		"port_range": 5,
		"base_port": 3000,
		"worktree_path": "/tmp/ws"
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Scripts.Setup != "make setup" {
		t.Errorf("Setup = %q, want %q", cfg.Scripts.Setup, "make setup")
	}
	if cfg.PortRange != 5 {
		t.Errorf("PortRange = %d, want 5", cfg.PortRange)
	}
	if cfg.BasePort != 3000 {
		t.Errorf("BasePort = %d, want 3000", cfg.BasePort)
	}
	if cfg.WorktreePath != "/tmp/ws" {
		t.Errorf("WorktreePath = %q, want /tmp/ws", cfg.WorktreePath)
	}
}

func TestLoadLegacyCamelCaseKeys(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fr8.json"), []byte(`{
		"portRange": 5,
		"basePort": 3000,
		"worktreePath": "/tmp/ws"
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PortRange != 5 {
		t.Errorf("PortRange = %d, want 5", cfg.PortRange)
	}
	if cfg.BasePort != 3000 {
		t.Errorf("BasePort = %d, want 3000", cfg.BasePort)
	}
	if cfg.WorktreePath != "/tmp/ws" {
		t.Errorf("WorktreePath = %q, want /tmp/ws", cfg.WorktreePath)
	}
}

func TestHasLegacyKeys(t *testing.T) {
	dir := t.TempDir()

	// File with legacy keys
	legacyPath := filepath.Join(dir, "legacy.json")
	if err := os.WriteFile(legacyPath, []byte(`{"portRange": 5, "basePort": 3000}`), 0644); err != nil {
		t.Fatal(err)
	}
	found := HasLegacyKeys(legacyPath)
	if len(found) != 2 {
		t.Errorf("HasLegacyKeys = %v, want 2 keys", found)
	}

	// File with snake_case keys
	modernPath := filepath.Join(dir, "modern.json")
	if err := os.WriteFile(modernPath, []byte(`{"port_range": 5, "base_port": 3000}`), 0644); err != nil {
		t.Fatal(err)
	}
	found = HasLegacyKeys(modernPath)
	if len(found) != 0 {
		t.Errorf("HasLegacyKeys = %v, want 0 keys", found)
	}
}

func TestLegacyKeyReplacement(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "portRange", key: "portRange", want: "port_range"},
		{name: "basePort", key: "basePort", want: "base_port"},
		{name: "worktreePath", key: "worktreePath", want: "worktree_path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LegacyKeyReplacement(tt.key)
			if got != tt.want {
				t.Errorf("LegacyKeyReplacement(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestMigrateKeys(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "fr8.json")
	if err := os.WriteFile(p, []byte(`{
  "scripts": {"setup": "make setup"},
  "portRange": 5,
  "basePort": 3000,
  "worktreePath": "/tmp/ws"
}`), 0644); err != nil {
		t.Fatal(err)
	}

	migrated, err := MigrateKeys(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(migrated) != 3 {
		t.Errorf("MigrateKeys returned %d keys, want 3", len(migrated))
	}

	// Verify the file was rewritten with snake_case keys
	legacy := HasLegacyKeys(p)
	if len(legacy) != 0 {
		t.Errorf("HasLegacyKeys after migration = %v, want none", legacy)
	}

	// Verify the config still loads correctly
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PortRange != 5 {
		t.Errorf("PortRange = %d, want 5", cfg.PortRange)
	}
	if cfg.BasePort != 3000 {
		t.Errorf("BasePort = %d, want 3000", cfg.BasePort)
	}
	if cfg.WorktreePath != "/tmp/ws" {
		t.Errorf("WorktreePath = %q, want /tmp/ws", cfg.WorktreePath)
	}
}

func TestMigrateKeysNoOp(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "fr8.json")
	if err := os.WriteFile(p, []byte(`{"port_range": 5, "base_port": 3000}`), 0644); err != nil {
		t.Fatal(err)
	}

	migrated, err := MigrateKeys(p)
	if err != nil {
		t.Fatal(err)
	}
	if migrated != nil {
		t.Errorf("MigrateKeys = %v, want nil (nothing to migrate)", migrated)
	}
}

func TestSnakeCaseTakesPrecedenceOverCamelCase(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fr8.json"), []byte(`{
		"port_range": 20,
		"portRange": 5
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PortRange != 20 {
		t.Errorf("PortRange = %d, want 20 (snake_case should take precedence)", cfg.PortRange)
	}
}

func TestLoadFallbackToConductorJson(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "conductor.json"), []byte(`{
		"scripts": {"setup": "bin/conductor setup"}
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Scripts.Setup != "bin/conductor setup" {
		t.Errorf("Setup = %q, want %q", cfg.Scripts.Setup, "bin/conductor setup")
	}
}

func TestLoadFr8JsonTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fr8.json"), []byte(`{"scripts": {"setup": "fr8-setup"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "conductor.json"), []byte(`{"scripts": {"setup": "conductor-setup"}}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Scripts.Setup != "fr8-setup" {
		t.Errorf("Setup = %q, want fr8-setup (fr8.json should take precedence)", cfg.Scripts.Setup)
	}
}

func TestLoadNoConfigFile(t *testing.T) {
	dir := t.TempDir()

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Should get defaults
	if cfg.PortRange != 10 {
		t.Errorf("PortRange = %d, want 10 (default)", cfg.PortRange)
	}
	if cfg.BasePort != 60000 {
		t.Errorf("BasePort = %d, want 60000 (default)", cfg.BasePort)
	}
	home, _ := os.UserHomeDir()
	wantPath := filepath.Join(home, "fr8")
	if cfg.WorktreePath != wantPath {
		t.Errorf("WorktreePath = %q, want %q (default)", cfg.WorktreePath, wantPath)
	}
}

func TestLoadInvalidJson(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fr8.json"), []byte(`{invalid`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)

	if cfg.PortRange != 10 {
		t.Errorf("PortRange = %d, want 10", cfg.PortRange)
	}
	if cfg.BasePort != 60000 {
		t.Errorf("BasePort = %d, want 60000", cfg.BasePort)
	}
	home, _ := os.UserHomeDir()
	wantPath := filepath.Join(home, "fr8")
	if cfg.WorktreePath != wantPath {
		t.Errorf("WorktreePath = %q, want %q", cfg.WorktreePath, wantPath)
	}
}

func TestApplyDefaultsPreservesValues(t *testing.T) {
	cfg := &Config{PortRange: 5, BasePort: 3000, WorktreePath: "/custom"}
	applyDefaults(cfg)

	if cfg.PortRange != 5 {
		t.Errorf("PortRange = %d, want 5 (should preserve)", cfg.PortRange)
	}
	if cfg.BasePort != 3000 {
		t.Errorf("BasePort = %d, want 3000 (should preserve)", cfg.BasePort)
	}
	if cfg.WorktreePath != "/custom" {
		t.Errorf("WorktreePath = %q, want /custom (should preserve)", cfg.WorktreePath)
	}
}

func TestResolveWorktreePathRelative(t *testing.T) {
	cfg := &Config{WorktreePath: "../fr8"}
	got := ResolveWorktreePath(cfg, "/Users/me/Code/myapp")
	want := "/Users/me/Code/fr8/myapp"
	if got != want {
		t.Errorf("ResolveWorktreePath = %q, want %q", got, want)
	}
}

func TestResolveWorktreePathAbsolute(t *testing.T) {
	cfg := &Config{WorktreePath: "/tmp/workspaces"}
	got := ResolveWorktreePath(cfg, "/Users/me/Code/myapp")
	want := "/tmp/workspaces/myapp"
	if got != want {
		t.Errorf("ResolveWorktreePath = %q, want %q", got, want)
	}
}

func TestResolveWorktreePathTilde(t *testing.T) {
	home, _ := os.UserHomeDir()
	cfg := &Config{WorktreePath: "~/fr8"}
	got := ResolveWorktreePath(cfg, "/Users/me/Code/myapp")
	want := filepath.Join(home, "fr8", "myapp")
	if got != want {
		t.Errorf("ResolveWorktreePath = %q, want %q", got, want)
	}
}
