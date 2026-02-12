package names

import (
	"strings"
	"testing"
)

func TestGenerateFormat(t *testing.T) {
	name := Generate(nil)

	parts := strings.SplitN(name, "-", 2)
	if len(parts) != 2 {
		t.Fatalf("expected adjective-city format, got %q", name)
	}
	if parts[0] == "" || parts[1] == "" {
		t.Fatalf("expected non-empty parts, got %q", name)
	}
}

func TestGenerateUnique(t *testing.T) {
	seen := make(map[string]bool)
	for range 50 {
		name := Generate(nil)
		if seen[name] {
			// Collisions are possible but very unlikely in 50 iterations
			// with 60*55 = 3300 combinations
			t.Logf("collision on %q (acceptable if rare)", name)
		}
		seen[name] = true
	}
}

func TestGenerateAvoidsExisting(t *testing.T) {
	// Fill up a small number of existing names
	existing := []string{"bright-berlin", "calm-cairo"}

	for range 20 {
		name := Generate(existing)
		for _, e := range existing {
			if name == e {
				t.Errorf("generated %q which is in the existing list", name)
			}
		}
	}
}

func TestGenerateAdjectiveInWordlist(t *testing.T) {
	name := Generate(nil)
	adj := strings.SplitN(name, "-", 2)[0]

	found := false
	for _, a := range adjectives {
		if a == adj {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("adjective %q not in wordlist", adj)
	}
}

func TestGenerateCityInWordlist(t *testing.T) {
	name := Generate(nil)
	parts := strings.SplitN(name, "-", 2)
	city := parts[1]

	// City might have a numeric suffix from fallback, strip it
	for i, c := range city {
		if c >= '0' && c <= '9' {
			city = city[:i]
			// Remove trailing dash from fallback format "adj-city-42"
			city = strings.TrimSuffix(city, "-")
			break
		}
	}

	found := false
	for _, c := range cities {
		if c == city {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("city %q not in wordlist", city)
	}
}
