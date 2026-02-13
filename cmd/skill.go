package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/thomascarr/fr8/internal/exitcode"
	"github.com/thomascarr/fr8/internal/jsonout"
)

var (
	skillName    string
	skillClaude  bool
	skillCodex   bool
	skillPath    string
	skillGlobal  bool
	skillProject bool
	skillForce   bool
)

func init() {
	skillInstallCmd.Flags().StringVar(&skillName, "name", "fr8", "skill directory name and frontmatter name")
	skillInstallCmd.Flags().BoolVar(&skillClaude, "claude", false, "install to Claude Code skills directory")
	skillInstallCmd.Flags().BoolVar(&skillCodex, "codex", false, "install to OpenAI Codex skills directory")
	skillInstallCmd.Flags().StringVar(&skillPath, "path", "", "explicit parent directory for the skill")
	skillInstallCmd.Flags().BoolVar(&skillGlobal, "global", false, "install to home directory (default)")
	skillInstallCmd.Flags().BoolVar(&skillProject, "project", false, "install to current working directory")
	skillInstallCmd.Flags().BoolVar(&skillForce, "force", false, "overwrite existing SKILL.md")

	skillInstallCmd.MarkFlagsMutuallyExclusive("claude", "codex", "path")
	skillInstallCmd.MarkFlagsMutuallyExclusive("global", "project")

	skillCmd.AddCommand(skillInstallCmd)
	rootCmd.AddCommand(skillCmd)
}

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Agent skill for AI integration",
	Long: `Agent skill for CLI-based AI integration.

Agent Skills teach AI agents how to use tools through structured markdown files.
Unlike MCP (which uses a persistent server), skills work through direct CLI
invocation — the agent reads the skill file and calls fr8 commands with --json.

Use "fr8 skill install" to generate a SKILL.md that teaches agents how to manage
workspaces via the fr8 CLI.

  CLI mode (skills):   Agent runs fr8 commands directly with --json output
  Server mode (MCP):   Agent connects to a persistent fr8 MCP server on stdio

Both modes expose the same 12 operations. Choose CLI mode when your agent
supports skills, or MCP mode when it supports the Model Context Protocol.

Learn more: https://agentskills.io`,
}

var skillInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the fr8 agent skill",
	Long: `Generate and install a SKILL.md file that teaches AI agents how to use fr8.

The skill file is installed into the appropriate skills directory based on the
target agent platform. By default, it installs to the Claude Code global skills
directory (~/.claude/skills/fr8/SKILL.md).`,
	Example: `  fr8 skill install                          # Claude Code, global (default)
  fr8 skill install --codex                   # OpenAI Codex, global
  fr8 skill install --claude --project        # Claude Code, current directory
  fr8 skill install --path ./custom           # Explicit directory
  fr8 skill install --name my-fr8 --force     # Custom name, overwrite`,
	Args: cobra.NoArgs,
	RunE: runSkillInstall,
}

var skillNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

func validateSkillName(name string) error {
	if name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("skill name must be 64 characters or fewer (got %d)", len(name))
	}
	if !skillNameRegex.MatchString(name) {
		return fmt.Errorf("skill name %q is invalid: must be lowercase alphanumeric and hyphens, no leading/trailing/consecutive hyphens", name)
	}
	for i := 0; i < len(name)-1; i++ {
		if name[i] == '-' && name[i+1] == '-' {
			return fmt.Errorf("skill name %q is invalid: consecutive hyphens are not allowed", name)
		}
	}
	return nil
}

// resolveSkillPath returns the parent directory where the skill directory should
// be created. The returned path does NOT include the skill name directory.
func resolveSkillPath(path string, claude, codex, global, project bool) (string, error) {
	if path != "" {
		if global || project {
			return "", fmt.Errorf("--path cannot be used with --global or --project")
		}
		return path, nil
	}

	// Determine agent suffix
	suffix := ".claude/skills"
	if codex {
		suffix = ".agents/skills"
	}

	// Determine base directory
	var base string
	if project {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting working directory: %w", err)
		}
		base = cwd
	} else {
		// Default: home directory (--global or no flag)
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home directory: %w", err)
		}
		base = home
	}

	return filepath.Join(base, suffix), nil
}

func runSkillInstall(cmd *cobra.Command, args []string) error {
	if err := validateSkillName(skillName); err != nil {
		return err
	}

	parent, err := resolveSkillPath(skillPath, skillClaude, skillCodex, skillGlobal, skillProject)
	if err != nil {
		return err
	}

	targetDir := filepath.Join(parent, skillName)
	targetFile := filepath.Join(targetDir, "SKILL.md")

	// Check for existing file
	if !skillForce {
		if _, err := os.Stat(targetFile); err == nil {
			return exitcode.New("already_exists", exitcode.AlreadyExists,
				fmt.Sprintf("SKILL.md already exists at %s (use --force to overwrite)", targetFile))
		}
	}

	// Render template
	tmpl, err := template.New("skill").Parse(skillMDTemplate)
	if err != nil {
		return fmt.Errorf("parsing skill template: %w", err)
	}

	var buf []byte
	w := &sliceWriter{buf: &buf}
	if err := tmpl.Execute(w, struct{ Name string }{Name: skillName}); err != nil {
		return fmt.Errorf("rendering skill template: %w", err)
	}

	// Write file
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating skill directory: %w", err)
	}
	if err := os.WriteFile(targetFile, buf, 0644); err != nil {
		return fmt.Errorf("writing SKILL.md: %w", err)
	}

	if jsonout.Enabled {
		return jsonout.Write(struct {
			Action string `json:"action"`
			Path   string `json:"path"`
			Name   string `json:"name"`
		}{
			Action: "installed",
			Path:   targetFile,
			Name:   skillName,
		})
	}

	fmt.Fprintf(jsonout.MsgOut(), "Installed %s skill to %s\n", skillName, targetFile)
	return nil
}

// sliceWriter is a minimal io.Writer that appends to a byte slice.
type sliceWriter struct {
	buf *[]byte
}

func (w *sliceWriter) Write(p []byte) (int, error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}
