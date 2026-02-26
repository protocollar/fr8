package port

import (
	"testing"
)

func TestAllocateEmpty(t *testing.T) {
	// Use a high port unlikely to be in use
	p, err := Allocate(nil, 61000, 10)
	if err != nil {
		t.Fatal(err)
	}
	if p != 61000 {
		t.Errorf("Allocate = %d, want 61000", p)
	}
}

func TestAllocateSkipsAllocated(t *testing.T) {
	allocated := []int{61000}
	p, err := Allocate(allocated, 61000, 10)
	if err != nil {
		t.Fatal(err)
	}
	if p != 61010 {
		t.Errorf("Allocate = %d, want 61010", p)
	}
}

func TestAllocateSkipsMultiple(t *testing.T) {
	allocated := []int{61000, 61010, 61020}
	p, err := Allocate(allocated, 61000, 10)
	if err != nil {
		t.Fatal(err)
	}
	if p != 61030 {
		t.Errorf("Allocate = %d, want 61030", p)
	}
}

func TestAllocateGap(t *testing.T) {
	// 61000 and 61020 are allocated, but 61010 is free
	allocated := []int{61000, 61020}
	p, err := Allocate(allocated, 61000, 10)
	if err != nil {
		t.Fatal(err)
	}
	if p != 61010 {
		t.Errorf("Allocate = %d, want 61010 (should fill gap)", p)
	}
}

func TestAllocateCustomPortRange(t *testing.T) {
	allocated := []int{61000}
	p, err := Allocate(allocated, 61000, 5)
	if err != nil {
		t.Fatal(err)
	}
	if p != 61005 {
		t.Errorf("Allocate = %d, want 61005", p)
	}
}

func TestAllocateRespectsMaxAttempts(t *testing.T) {
	// Allocate all blocks so nothing is left
	var allocated []int
	for i := 0; i < maxAttempts; i++ {
		allocated = append(allocated, 61000+(i*10))
	}
	_, err := Allocate(allocated, 61000, 10)
	if err == nil {
		t.Fatal("expected error when all blocks are allocated")
	}
}

func TestAllocateStopsBeforePortOverflow(t *testing.T) {
	// Base port near the top of the range — should not allocate above 65535
	_, err := Allocate(nil, 65530, 10)
	if err == nil {
		t.Fatal("expected error when port block would exceed 65535")
	}
}

func TestIsFreeUnusedPort(t *testing.T) {
	// Port 59999 is very unlikely to be in use
	if !IsFree(59999) {
		t.Skip("port 59999 unexpectedly in use")
	}
}

func TestBlockFreeUnusedPorts(t *testing.T) {
	// High ports very unlikely to be in use
	if !BlockFree(63000, 10) {
		t.Skip("ports 63000-63009 unexpectedly in use")
	}
}
