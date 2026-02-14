package port

import (
	"fmt"
	"net"
	"time"
)

// Allocate finds the next available port block starting from basePort.
// Each block is portRange consecutive ports. It skips blocks that overlap
// with any port in allocatedPorts.
func Allocate(allocatedPorts []int, basePort, portRange int) (int, error) {
	allocated := make(map[int]bool, len(allocatedPorts))
	for _, p := range allocatedPorts {
		allocated[p] = true
	}

	// Try up to 100 blocks before giving up.
	for i := 0; i < 100; i++ {
		candidate := basePort + (i * portRange)
		if allocated[candidate] {
			continue
		}
		if IsFree(candidate) {
			return candidate, nil
		}
	}
	return 0, fmt.Errorf("no free port block found (tried %d-%d); try archiving unused workspaces with: fr8 ws archive", basePort, basePort+100*portRange)
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
