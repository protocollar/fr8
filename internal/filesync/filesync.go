package filesync

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Sync copies files matching .worktreeinclude patterns from rootPath to worktreePath.
// Files that already exist with identical content are skipped.
func Sync(rootPath, worktreePath string) error {
	// Look for .worktreeinclude in worktree first, then root
	var includeFile string
	for _, base := range []string{worktreePath, rootPath} {
		p := filepath.Join(base, ".worktreeinclude")
		if _, err := os.Stat(p); err == nil {
			includeFile = p
			break
		}
	}

	if includeFile == "" {
		return nil // no .worktreeinclude, nothing to sync
	}

	patterns, err := parseIncludeFile(includeFile)
	if err != nil {
		return fmt.Errorf("parsing .worktreeinclude: %w", err)
	}

	for _, pattern := range patterns {
		matches, err := doublestar.Glob(os.DirFS(rootPath), pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: invalid pattern %q: %v\n", pattern, err)
			continue
		}

		for _, rel := range matches {
			src := filepath.Join(rootPath, rel)
			dst := filepath.Join(worktreePath, rel)

			info, err := os.Stat(src)
			if err != nil || info.IsDir() {
				continue
			}

			if filesEqual(src, dst) {
				continue
			}

			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return fmt.Errorf("creating directory for %s: %w", rel, err)
			}

			srcData, err := os.ReadFile(src)
			if err != nil {
				return fmt.Errorf("reading %s: %w", rel, err)
			}

			if err := os.WriteFile(dst, srcData, info.Mode()); err != nil {
				return fmt.Errorf("writing %s: %w", rel, err)
			}

			fmt.Printf("  Copied %s\n", rel)
		}
	}

	return nil
}

func parseIncludeFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns, scanner.Err()
}

func filesEqual(a, b string) bool {
	dataA, errA := os.ReadFile(a)
	dataB, errB := os.ReadFile(b)
	if errA != nil || errB != nil {
		return false
	}
	return bytes.Equal(dataA, dataB)
}
