package gh

import "testing"

func TestPRStatusGracefulDegradation(t *testing.T) {
	// A temp dir with no GitHub remote should return nil, nil
	dir := t.TempDir()
	pr, err := PRStatus(dir, "main")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if pr != nil {
		t.Errorf("expected nil PRInfo, got %+v", pr)
	}
}
