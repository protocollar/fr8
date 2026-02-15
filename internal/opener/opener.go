package opener

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/protocollar/fr8/internal/userconfig"
)

// Run resolves the opener's command to an executable and opens the workspace path.
// The Command field may contain arguments (e.g. "code --new-window").
// Returns an error if the executable is not found in $PATH.
func Run(o userconfig.Opener, workspacePath string) error {
	parts := strings.Fields(o.Command)
	if len(parts) == 0 {
		return fmt.Errorf("opener %q has an empty command", o.Name)
	}
	binPath, err := exec.LookPath(parts[0])
	if err != nil {
		return fmt.Errorf("%s: executable not found in $PATH (check that it is installed and on your PATH)", parts[0])
	}
	args := make([]string, len(parts)-1, len(parts))
	copy(args, parts[1:])
	args = append(args, workspacePath)
	cmd := exec.Command(binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}
