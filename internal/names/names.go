package names

import (
	"fmt"
	"math/rand/v2"
)

// Generate creates a unique adjective-city name that doesn't collide with existing names.
func Generate(existing []string) string {
	taken := make(map[string]bool, len(existing))
	for _, n := range existing {
		taken[n] = true
	}

	for range 100 {
		adj := adjectives[rand.IntN(len(adjectives))]
		city := cities[rand.IntN(len(cities))]
		name := adj + "-" + city
		if !taken[name] {
			return name
		}
	}

	// Fallback: append random suffix
	adj := adjectives[rand.IntN(len(adjectives))]
	city := cities[rand.IntN(len(cities))]
	return fmt.Sprintf("%s-%s-%d", adj, city, rand.IntN(100))
}
