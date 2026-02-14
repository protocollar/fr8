package flock

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestLockUnlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := Lock(f.Fd()); err != nil {
		t.Fatalf("Lock: %v", err)
	}

	if err := Unlock(f.Fd()); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
}

func TestLockIsExclusive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	f1, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f1.Close()

	if err := Lock(f1.Fd()); err != nil {
		t.Fatalf("Lock f1: %v", err)
	}
	defer Unlock(f1.Fd())

	// Open a second fd and try a non-blocking lock — it should fail
	// with EWOULDBLOCK since f1 holds the exclusive lock.
	f2, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	err = syscall.Flock(int(f2.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		t.Error("expected non-blocking lock to fail while f1 holds lock")
		_ = Unlock(f2.Fd())
	}
}

func TestRelockAfterUnlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lock")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Lock, unlock, then lock again — should succeed
	if err := Lock(f.Fd()); err != nil {
		t.Fatalf("first Lock: %v", err)
	}
	if err := Unlock(f.Fd()); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if err := Lock(f.Fd()); err != nil {
		t.Fatalf("second Lock: %v", err)
	}
	if err := Unlock(f.Fd()); err != nil {
		t.Fatalf("second Unlock: %v", err)
	}
}
