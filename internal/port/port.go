package port

import (
	"fmt"
	"net"
	"time"
)

// maxAttempts is the number of consecutive port blocks to try before giving up.
const maxAttempts = 100

// Allocate finds the next available port block starting from basePort.
// Each block is portRange consecutive ports. It skips blocks that overlap
// with any port in allocatedPorts, and validates that every port in the
// candidate block is free (not just the first).
func Allocate(allocatedPorts []int, basePort, portRange int) (int, error) {
	allocated := make(map[int]bool, len(allocatedPorts))
	for _, p := range allocatedPorts {
		allocated[p] = true
	}

	for i := 0; i < maxAttempts; i++ {
		candidate := basePort + (i * portRange)
		if candidate+portRange-1 > 65535 {
			break
		}
		if allocated[candidate] {
			continue
		}
		if BlockFree(candidate, portRange) {
			return candidate, nil
		}
	}
	maxPort := basePort + maxAttempts*portRange
	if maxPort > 65535 {
		maxPort = 65535
	}
	return 0, fmt.Errorf("no free port block found (tried %d-%d); try archiving unused workspaces with: fr8 ws archive", basePort, maxPort)
}

// BlockFree returns true if every port in [startPort, startPort+count) is free.
func BlockFree(startPort, count int) bool {
	for p := startPort; p < startPort+count; p++ {
		if !IsFree(p) {
			return false
		}
	}
	return true
}

// IsFree returns true if the port is not in use.
func IsFree(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 200*time.Millisecond)
	if err != nil {
		return true // connection refused = port is free
	}
	_ = conn.Close()
	return false
}
