package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestValidateSkillName(t *testing.T) {
	valid := []string{
		"fr8",
		"my-skill",
		"a",
		"a1b2",
		"abc",
		"a-b-c",
		strings.Repeat("a", 64),
	}
	for _, name := range valid {
		if err := validateSkillName(name); err != nil {
			t.Errorf("validateSkillName(%q) = %v, want nil", name, err)
		}
	}

	invalid := []struct {
		name string
		desc string
	}{
		{"", "empty"},
		{"-start", "leading hyphen"},
		{"end-", "trailing hyphen"},
		{"double--hyphen", "consecutive hyphens"},
		{"UPPER", "uppercase"},
		{"has space", "space"},
		{"has_underscore", "underscore"},
		{strings.Repeat("a", 65), "too long"},
		{"-", "single hyphen"},
		{"a--b", "consecutive hyphens mid"},
	}
	for _, tc := range invalid {
		if err := validateSkillName(tc.name); err == nil {
			t.Errorf("validateSkillName(%q) [%s] = nil, want error", tc.name, tc.desc)
		}
	}
}

func TestResolveSkillPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		claude  bool
		codex   bool
		global  bool
		project bool
		want    string
		wantErr bool
	}{
		{
			name: "default (no flags)",
			want: filepath.Join(home, ".claude/skills"),
		},
		{
			name:   "claude flag",
			claude: true,
			want:   filepath.Join(home, ".claude/skills"),
		},
		{
			name:  "codex flag",
			codex: true,
			want:  filepath.Join(home, ".agents/skills"),
		},
		{
			name:    "claude project",
			claude:  true,
			project: true,
			want:    filepath.Join(cwd, ".claude/skills"),
		},
		{
			name:    "codex project",
			codex:   true,
			project: true,
			want:    filepath.Join(cwd, ".agents/skills"),
		},
		{
			name: "explicit path",
			path: "/custom",
			want: "/custom",
		},
		{
			name:    "path with global errors",
			path:    "/custom",
			global:  true,
			wantErr: true,
		},
		{
			name:    "path with project errors",
			path:    "/custom",
			project: true,
			wantErr: true,
		},
		{
			name:   "global flag (same as default)",
			global: true,
			want:   filepath.Join(home, ".claude/skills"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveSkillPath(tc.path, tc.claude, tc.codex, tc.global, tc.project)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolveSkillPath() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveSkillPath() error = %v", err)
			}
			if got != tc.want {
				t.Errorf("resolveSkillPath() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRunSkillInstall(t *testing.T) {
	t.Run("creates directory and SKILL.md", func(t *testing.T) {
		dir := t.TempDir()

		// Set package-level flags
		skillName = "fr8"
		skillPath = dir
		skillClaude = false
		skillCodex = false
		skillGlobal = false
		skillProject = false
		skillForce = false

		if err := runSkillInstall(nil, nil); err != nil {
			t.Fatalf("runSkillInstall() error = %v", err)
		}

		targetFile := filepath.Join(dir, "fr8", "SKILL.md")
		data, err := os.ReadFile(targetFile)
		if err != nil {
			t.Fatalf("reading SKILL.md: %v", err)
		}

		content := string(data)
		if !strings.HasPrefix(content, "---") {
			t.Error("SKILL.md should start with frontmatter delimiter")
		}
		if !strings.Contains(content, "name: fr8") {
			t.Error("SKILL.md should contain 'name: fr8'")
		}
	})

	t.Run("custom name appears in path and content", func(t *testing.T) {
		dir := t.TempDir()

		skillName = "my-tool"
		skillPath = dir
		skillForce = false

		if err := runSkillInstall(nil, nil); err != nil {
			t.Fatalf("runSkillInstall() error = %v", err)
		}

		targetFile := filepath.Join(dir, "my-tool", "SKILL.md")
		data, err := os.ReadFile(targetFile)
		if err != nil {
			t.Fatalf("reading SKILL.md: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, "name: my-tool") {
			t.Error("SKILL.md should contain 'name: my-tool'")
		}
		if !strings.Contains(content, "# my-tool") {
			t.Error("SKILL.md should contain '# my-tool' heading")
		}
	})

	t.Run("errors when SKILL.md exists without force", func(t *testing.T) {
		dir := t.TempDir()

		skillName = "fr8"
		skillPath = dir
		skillForce = false

		// Create the file first
		targetDir := filepath.Join(dir, "fr8")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte("existing"), 0644); err != nil {
			t.Fatal(err)
		}

		err := runSkillInstall(nil, nil)
		if err == nil {
			t.Fatal("runSkillInstall() should error when SKILL.md exists")
		}
		if !strings.Contains(err.Error(), "--force") {
			t.Errorf("error should mention --force, got: %v", err)
		}
	})

	t.Run("succeeds with force when SKILL.md exists", func(t *testing.T) {
		dir := t.TempDir()

		skillName = "fr8"
		skillPath = dir
		skillForce = true

		// Create the file first
		targetDir := filepath.Join(dir, "fr8")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte("old"), 0644); err != nil {
			t.Fatal(err)
		}

		if err := runSkillInstall(nil, nil); err != nil {
			t.Fatalf("runSkillInstall() with --force error = %v", err)
		}

		data, err := os.ReadFile(filepath.Join(targetDir, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) == "old" {
			t.Error("SKILL.md was not overwritten")
		}
	})
}

func TestSkillTemplateRendering(t *testing.T) {
	tmpl, err := template.New("skill").Parse(skillMDTemplate)
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, struct{ Name string }{Name: "custom-name"})
	if err != nil {
		t.Fatalf("template execute error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "name: custom-name") {
		t.Error("template should substitute name in frontmatter")
	}
	if !strings.Contains(output, "# custom-name") {
		t.Error("template should substitute name in heading")
	}
	if strings.Contains(output, "{{.Name}}") {
		t.Error("template should not contain unresolved placeholders")
	}
}
