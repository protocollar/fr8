package mcp

import "testing"

func TestNewServer(t *testing.T) {
	s := NewServer("0.0.1-test")
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
}
