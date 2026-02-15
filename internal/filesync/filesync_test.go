package filesync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseIncludeFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, ".worktreeinclude")
	if err := os.WriteFile(f, []byte(`# comment
.env*

# another comment
config/master.key
config/credentials/*.key

.mcp.json
`), 0644); err != nil {
		t.Fatal(err)
	}

	patterns, err := parseIncludeFile(f)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{".env*", "config/master.key", "config/credentials/*.key", ".mcp.json"}
	if len(patterns) != len(expected) {
		t.Fatalf("got %d patterns, want %d: %v", len(patterns), len(expected), patterns)
	}
	for i, p := range patterns {
		if p != expected[i] {
			t.Errorf("pattern[%d] = %q, want %q", i, p, expected[i])
		}
	}
}

func TestParseIncludeFileEmpty(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, ".worktreeinclude")
	if err := os.WriteFile(f, []byte("# only comments\n\n# nothing here\n"), 0644); err != nil {
		t.Fatal(err)
	}

	patterns, err := parseIncludeFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns) != 0 {
		t.Errorf("expected 0 patterns, got %d", len(patterns))
	}
}

func TestFilesEqual(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	c := filepath.Join(dir, "c")

	if err := os.WriteFile(a, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c, []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}

	if !filesEqual(a, b) {
		t.Error("expected a and b to be equal")
	}
	if filesEqual(a, c) {
		t.Error("expected a and c to differ")
	}
	if filesEqual(a, filepath.Join(dir, "missing")) {
		t.Error("expected false when file is missing")
	}
}

func TestFilesEqualDifferentSizes(t *testing.T) {
	dir := t.TempDir()
	short := filepath.Join(dir, "short")
	long := filepath.Join(dir, "long")

	if err := os.WriteFile(short, []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(long, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	// Different sizes should return false without reading content
	if filesEqual(short, long) {
		t.Error("expected files with different sizes to be unequal")
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")

	content := []byte("streaming copy test content")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Errorf("copied content = %q, want %q", got, content)
	}
}

func TestCopyFilePreservesMode(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")

	if err := os.WriteFile(src, []byte("exec"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst, 0755); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	// Check executable bit is set (masking umask variations)
	if info.Mode()&0100 == 0 {
		t.Errorf("expected executable mode, got %v", info.Mode())
	}
}

func TestCopyFileMissingSrc(t *testing.T) {
	dir := t.TempDir()
	err := copyFile(filepath.Join(dir, "missing"), filepath.Join(dir, "dst"), 0644)
	if err == nil {
		t.Error("expected error when source file is missing")
	}
}

func TestCopyFileOverwrites(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")

	if err := os.WriteFile(src, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("old content here"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Errorf("expected dst to be overwritten, got %q", got)
	}
}

func TestSyncCopiesFiles(t *testing.T) {
	root := t.TempDir()
	worktree := t.TempDir()

	// Create source files
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET=123"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env.local"), []byte("LOCAL=yes"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "master.key"), []byte("key123"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .worktreeinclude
	if err := os.WriteFile(filepath.Join(root, ".worktreeinclude"), []byte(".env*\nconfig/master.key\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Sync(root, worktree); err != nil {
		t.Fatal(err)
	}

	// Verify files were copied
	for _, rel := range []string{".env", ".env.local", "config/master.key"} {
		dst := filepath.Join(worktree, rel)
		if _, err := os.Stat(dst); err != nil {
			t.Errorf("expected %s to be copied", rel)
		}
	}
}

func TestSyncSkipsIdentical(t *testing.T) {
	root := t.TempDir()
	worktree := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET=123"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktree, ".env"), []byte("SECRET=123"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".worktreeinclude"), []byte(".env\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Get mod time before sync
	info, _ := os.Stat(filepath.Join(worktree, ".env"))
	modBefore := info.ModTime()

	if err := Sync(root, worktree); err != nil {
		t.Fatal(err)
	}

	// File should not have been rewritten
	info, _ = os.Stat(filepath.Join(worktree, ".env"))
	if info.ModTime() != modBefore {
		t.Error("expected identical file to be skipped")
	}
}

func TestSyncNoIncludeFile(t *testing.T) {
	root := t.TempDir()
	worktree := t.TempDir()

	// No .worktreeinclude — should be a no-op
	if err := Sync(root, worktree); err != nil {
		t.Fatal(err)
	}
}

func TestSyncCreatesDirectories(t *testing.T) {
	root := t.TempDir()
	worktree := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "config", "credentials"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config", "credentials", "dev.key"), []byte("key"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".worktreeinclude"), []byte("config/credentials/*.key\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Sync(root, worktree); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(worktree, "config", "credentials", "dev.key")
	if _, err := os.Stat(dst); err != nil {
		t.Error("expected nested directory and file to be created")
	}
}
