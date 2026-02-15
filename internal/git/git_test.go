package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestParsePorcelain(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []Worktree
	}{
		{
			name:   "empty",
			input:  "",
			expect: nil,
		},
		{
			name: "single worktree",
			input: "worktree /Users/me/project\n" +
				"HEAD abc123\n" +
				"branch refs/heads/main\n" +
				"\n",
			expect: []Worktree{
				{Path: "/Users/me/project", HEAD: "abc123", Branch: "main"},
			},
		},
		{
			name: "multiple worktrees",
			input: "worktree /Users/me/project\n" +
				"HEAD abc123\n" +
				"branch refs/heads/main\n" +
				"\n" +
				"worktree /Users/me/worktrees/feature\n" +
				"HEAD def456\n" +
				"branch refs/heads/feature/auth\n" +
				"\n",
			expect: []Worktree{
				{Path: "/Users/me/project", HEAD: "abc123", Branch: "main"},
				{Path: "/Users/me/worktrees/feature", HEAD: "def456", Branch: "feature/auth"},
			},
		},
		{
			name: "bare repo",
			input: "worktree /Users/me/project.git\n" +
				"HEAD abc123\n" +
				"bare\n" +
				"\n",
			expect: []Worktree{
				{Path: "/Users/me/project.git", HEAD: "abc123", Bare: true},
			},
		},
		{
			name: "no trailing newline",
			input: "worktree /Users/me/project\n" +
				"HEAD abc123\n" +
				"branch refs/heads/main",
			expect: []Worktree{
				{Path: "/Users/me/project", HEAD: "abc123", Branch: "main"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePorcelain(tt.input)
			if len(got) != len(tt.expect) {
				t.Fatalf("got %d worktrees, want %d", len(got), len(tt.expect))
			}
			for i := range got {
				if got[i] != tt.expect[i] {
					t.Errorf("worktree[%d] = %+v, want %+v", i, got[i], tt.expect[i])
				}
			}
		})
	}
}

// Integration tests that use a real git repo

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", args, out)
		}
	}
	return dir
}

func TestWorktreeListIntegration(t *testing.T) {
	dir := initTestRepo(t)

	wts, err := WorktreeList(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(wts))
	}
	// macOS resolves /var -> /private/var for temp dirs, so resolve both
	gotReal, _ := filepath.EvalSymlinks(wts[0].Path)
	wantReal, _ := filepath.EvalSymlinks(dir)
	if gotReal != wantReal {
		t.Errorf("path = %q, want %q", gotReal, wantReal)
	}
}

func TestWorktreeAddRemoveIntegration(t *testing.T) {
	dir := initTestRepo(t)
	wtPath := filepath.Join(t.TempDir(), "feature-ws")

	if err := WorktreeAdd(dir, wtPath, "feature", true, ""); err != nil {
		t.Fatal(err)
	}

	wts, err := WorktreeList(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(wts))
	}

	if err := WorktreeRemove(dir, wtPath); err != nil {
		t.Fatal(err)
	}

	wts, err = WorktreeList(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 1 {
		t.Fatalf("expected 1 worktree after remove, got %d", len(wts))
	}
}

func TestWorktreeMoveIntegration(t *testing.T) {
	dir := initTestRepo(t)
	oldPath := filepath.Join(t.TempDir(), "old-ws")
	newPath := filepath.Join(t.TempDir(), "new-ws")

	// Add a worktree
	if err := WorktreeAdd(dir, oldPath, "feature", true, ""); err != nil {
		t.Fatal(err)
	}

	// Move it
	if err := WorktreeMove(dir, oldPath, newPath); err != nil {
		t.Fatal(err)
	}

	// Old path should no longer exist
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("expected old path to not exist after move")
	}

	// New path should exist
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("expected new path to exist after move: %v", err)
	}

	// Git should list the worktree at the new path
	wts, err := WorktreeList(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, wt := range wts {
		real, _ := filepath.EvalSymlinks(wt.Path)
		want, _ := filepath.EvalSymlinks(newPath)
		if real == want {
			found = true
		}
	}
	if !found {
		t.Errorf("worktree list does not contain new path %q", newPath)
	}

	// Cleanup
	if err := WorktreeRemove(dir, newPath); err != nil {
		t.Fatal(err)
	}
}

func TestWorktreeMoveNonexistentIntegration(t *testing.T) {
	dir := initTestRepo(t)

	err := WorktreeMove(dir, "/nonexistent/worktree", "/tmp/new-path")
	if err == nil {
		t.Error("expected error when moving nonexistent worktree")
	}
}

func TestCommonDirIntegration(t *testing.T) {
	dir := initTestRepo(t)

	common, err := CommonDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(dir, ".git")
	if common != expected {
		t.Errorf("CommonDir = %q, want %q", common, expected)
	}
}

func TestDefaultBranchIntegration(t *testing.T) {
	dir := initTestRepo(t)

	// Default git init creates "main" (or "master" on older git).
	branch, err := DefaultBranch(dir)
	if err != nil {
		t.Fatal(err)
	}
	if branch != "main" && branch != "master" {
		t.Errorf("DefaultBranch = %q, want main or master", branch)
	}
}

func TestCurrentBranchIntegration(t *testing.T) {
	dir := initTestRepo(t)

	branch, err := CurrentBranch(dir)
	if err != nil {
		t.Fatal(err)
	}
	if branch != "main" && branch != "master" {
		t.Errorf("CurrentBranch = %q, want main or master", branch)
	}
}

func TestHasUncommittedChangesIntegration(t *testing.T) {
	dir := initTestRepo(t)

	dirty, err := HasUncommittedChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dirty {
		t.Error("expected clean repo")
	}

	// Create an untracked file
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	dirty, err = HasUncommittedChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !dirty {
		t.Error("expected dirty repo")
	}
}

func TestIsInsideWorkTreeIntegration(t *testing.T) {
	dir := initTestRepo(t)

	if !IsInsideWorkTree(dir) {
		t.Error("expected true for git repo")
	}
	if IsInsideWorkTree(t.TempDir()) {
		t.Error("expected false for non-git dir")
	}
}

func TestIsMergedIntegration(t *testing.T) {
	dir := initTestRepo(t)

	defaultBranch, err := DefaultBranch(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a feature branch from the current commit.
	runGit(t, dir, "checkout", "-b", "feature")
	runGit(t, dir, "checkout", defaultBranch)

	// At this point feature and main point at the same commit,
	// so feature is trivially an ancestor of main.
	merged, err := IsMerged(dir, "feature", defaultBranch)
	if err != nil {
		t.Fatal(err)
	}
	if !merged {
		t.Error("expected feature to be merged (same commit)")
	}

	// Add a commit on feature — now feature is ahead.
	runGit(t, dir, "checkout", "feature")
	runGit(t, dir, "commit", "--allow-empty", "-m", "feature work")
	runGit(t, dir, "checkout", defaultBranch)

	// feature is no longer an ancestor of main.
	merged, err = IsMerged(dir, "feature", defaultBranch)
	if err != nil {
		t.Fatal(err)
	}
	if merged {
		t.Error("expected feature NOT to be merged after adding a commit")
	}

	// But main IS an ancestor of feature.
	merged, err = IsMerged(dir, defaultBranch, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if !merged {
		t.Error("expected main to be ancestor of feature")
	}
}

func TestAheadBehindIntegration(t *testing.T) {
	dir := initTestRepo(t)

	defaultBranch, err := DefaultBranch(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Create feature branch.
	runGit(t, dir, "checkout", "-b", "feature")

	// Identical — 0 ahead, 0 behind.
	ahead, behind, err := AheadBehind(dir, "feature", defaultBranch)
	if err != nil {
		t.Fatal(err)
	}
	if ahead != 0 || behind != 0 {
		t.Errorf("expected 0/0, got ahead=%d behind=%d", ahead, behind)
	}

	// Add 2 commits on feature.
	runGit(t, dir, "commit", "--allow-empty", "-m", "f1")
	runGit(t, dir, "commit", "--allow-empty", "-m", "f2")

	ahead, behind, err = AheadBehind(dir, "feature", defaultBranch)
	if err != nil {
		t.Fatal(err)
	}
	if ahead != 2 || behind != 0 {
		t.Errorf("expected 2/0, got ahead=%d behind=%d", ahead, behind)
	}

	// Add 1 commit on main to create divergence.
	runGit(t, dir, "checkout", defaultBranch)
	runGit(t, dir, "commit", "--allow-empty", "-m", "m1")

	ahead, behind, err = AheadBehind(dir, "feature", defaultBranch)
	if err != nil {
		t.Fatal(err)
	}
	if ahead != 2 || behind != 1 {
		t.Errorf("expected 2/1, got ahead=%d behind=%d", ahead, behind)
	}
}

func TestTrackingBranchIntegration(t *testing.T) {
	// Create a bare repo to act as a remote.
	bare := t.TempDir()
	runGit(t, bare, "init", "--bare")

	// Create a working repo and push to the bare remote.
	dir := initTestRepo(t)
	runGit(t, dir, "remote", "add", "origin", bare)
	defaultBranch, err := DefaultBranch(dir)
	if err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "push", "-u", "origin", defaultBranch)

	tracking, err := TrackingBranch(dir, defaultBranch)
	if err != nil {
		t.Fatal(err)
	}
	want := "origin/" + defaultBranch
	if tracking != want {
		t.Errorf("TrackingBranch = %q, want %q", tracking, want)
	}

	// A branch with no upstream should return an error.
	runGit(t, dir, "checkout", "-b", "no-upstream")
	_, err = TrackingBranch(dir, "no-upstream")
	if err == nil {
		t.Error("expected error for branch with no upstream")
	}
}

func TestDirtyStatusClean(t *testing.T) {
	dir := initTestRepo(t)
	dc, err := DirtyStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dc.Dirty() {
		t.Errorf("expected clean, got %+v", dc)
	}
}

func TestDirtyStatusUntracked(t *testing.T) {
	dir := initTestRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	dc, err := DirtyStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dc.Untracked != 1 {
		t.Errorf("Untracked = %d, want 1", dc.Untracked)
	}
	if dc.Staged != 0 || dc.Modified != 0 {
		t.Errorf("expected only untracked, got %+v", dc)
	}
}

func TestDirtyStatusStaged(t *testing.T) {
	dir := initTestRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "staged.txt")
	dc, err := DirtyStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dc.Staged != 1 {
		t.Errorf("Staged = %d, want 1", dc.Staged)
	}
	if dc.Modified != 0 || dc.Untracked != 0 {
		t.Errorf("expected only staged, got %+v", dc)
	}
}

func TestDirtyStatusModified(t *testing.T) {
	dir := initTestRepo(t)
	// Create a tracked file
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "tracked.txt")
	runGit(t, dir, "commit", "-m", "add tracked")
	// Modify the tracked file
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}
	dc, err := DirtyStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dc.Modified != 1 {
		t.Errorf("Modified = %d, want 1", dc.Modified)
	}
	if dc.Staged != 0 || dc.Untracked != 0 {
		t.Errorf("expected only modified, got %+v", dc)
	}
}

func TestDirtyStatusMixed(t *testing.T) {
	dir := initTestRepo(t)
	// Staged file
	if err := os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("s"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "staged.txt")

	// Modified tracked file
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("t"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "tracked.txt")
	runGit(t, dir, "commit", "-m", "add tracked")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("mod"), 0644); err != nil {
		t.Fatal(err)
	}

	// Untracked file
	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("u"), 0644); err != nil {
		t.Fatal(err)
	}

	dc, err := DirtyStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Note: staged.txt was committed by the commit above, so it's no longer staged.
	// We need a fresh staged file.
	if err := os.WriteFile(filepath.Join(dir, "staged2.txt"), []byte("s2"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "staged2.txt")

	dc, err = DirtyStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dc.Staged != 1 {
		t.Errorf("Staged = %d, want 1", dc.Staged)
	}
	if dc.Modified != 1 {
		t.Errorf("Modified = %d, want 1", dc.Modified)
	}
	if dc.Untracked != 1 {
		t.Errorf("Untracked = %d, want 1", dc.Untracked)
	}
}

func TestLastCommitIntegration(t *testing.T) {
	dir := initTestRepo(t)
	// The init commit has subject "init" from initTestRepo
	ci, err := LastCommit(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ci.Subject != "init" {
		t.Errorf("Subject = %q, want %q", ci.Subject, "init")
	}
	// Commit time should be recent (within last 60 seconds)
	if time.Since(ci.Time) > 60*time.Second {
		t.Errorf("commit time %v is too old", ci.Time)
	}
}

// runGit is a test helper that runs a git command in dir and fails on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %s", args, out)
	}
}
