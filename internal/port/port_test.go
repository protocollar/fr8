package port

import (
	"testing"
)

func TestAllocateEmpty(t *testing.T) {
	// Use a high port unlikely to be in use
	p, err := Allocate(nil, 51000, 10)
	if err != nil {
		t.Fatal(err)
	}
	if p != 51000 {
		t.Errorf("Allocate = %d, want 51000", p)
	}
}

func TestAllocateSkipsAllocated(t *testing.T) {
	allocated := []int{51000}
	p, err := Allocate(allocated, 51000, 10)
	if err != nil {
		t.Fatal(err)
	}
	if p != 51010 {
		t.Errorf("Allocate = %d, want 51010", p)
	}
}

func TestAllocateSkipsMultiple(t *testing.T) {
	allocated := []int{51000, 51010, 51020}
	p, err := Allocate(allocated, 51000, 10)
	if err != nil {
		t.Fatal(err)
	}
	if p != 51030 {
		t.Errorf("Allocate = %d, want 51030", p)
	}
}

func TestAllocateGap(t *testing.T) {
	// 51000 and 51020 are allocated, but 51010 is free
	allocated := []int{51000, 51020}
	p, err := Allocate(allocated, 51000, 10)
	if err != nil {
		t.Fatal(err)
	}
	if p != 51010 {
		t.Errorf("Allocate = %d, want 51010 (should fill gap)", p)
	}
}

func TestAllocateCustomPortRange(t *testing.T) {
	allocated := []int{51000}
	p, err := Allocate(allocated, 51000, 5)
	if err != nil {
		t.Fatal(err)
	}
	if p != 51005 {
		t.Errorf("Allocate = %d, want 51005", p)
	}
}

func TestIsFreeUnusedPort(t *testing.T) {
	// Port 59999 is very unlikely to be in use
	if !IsFree(59999) {
		t.Skip("port 59999 unexpectedly in use")
	}
}
