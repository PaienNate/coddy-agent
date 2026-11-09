package skills_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EvilFreelancer/coddy-agent/internal/skills"
)

func TestWithToolCompatMapsForeignToolNames(t *testing.T) {
	s := &skills.Skill{Content: "Dispatch a `Task` and keep a `TodoWrite` list."}
	got := skills.WithToolCompat(s, s.Content)
	for _, want := range []string{
		"## Coddy tool mapping",
		"`Skill` -> `load_skill`",
		"`Task` / `Agent` -> `spawn_agent`",
		"`TodoWrite` -> `coddy_todo_plan_replace`",
		"`Read` -> `read`",
		"`Bash` -> `run_command`",
		"Never invent a tool call",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("mapping missing %q:\n%s", want, got)
		}
	}
	if !strings.HasSuffix(got, s.Content) {
		t.Error("mapping must be prepended, not replace the skill body")
	}
}

func TestWithToolCompatLeavesNativeBodyAlone(t *testing.T) {
	s := &skills.Skill{Content: "Read the file and run the tests."}
	if got := skills.WithToolCompat(s, s.Content); got != s.Content {
		t.Fatalf("native body changed:\n%s", got)
	}
}

func TestWithToolCompatHonoursExplicitMarker(t *testing.T) {
	body := "Use the `Task` tool."
	foreign := &skills.Skill{Content: body, Compat: "claude-code"}
	if got := skills.WithToolCompat(foreign, body); !strings.Contains(got, "## Coddy tool mapping") {
		t.Fatal("explicit foreign compatibility must inject the mapping")
	}
	native := &skills.Skill{Content: body, Compat: "coddy"}
	if got := skills.WithToolCompat(native, body); got != body {
		t.Fatalf("explicit native compatibility must not inject, got:\n%s", got)
	}
}

func TestWithToolCompatIsIdempotent(t *testing.T) {
	s := &skills.Skill{Content: "`Task`"}
	once := skills.WithToolCompat(s, s.Content)
	twice := skills.WithToolCompat(s, once)
	if once != twice {
		t.Fatal("mapping was injected twice")
	}
}

func TestLoadSkillCompatFromMetadata(t *testing.T) {
	cases := []struct {
		name  string
		front string
		want  string
	}{
		{"top level compatibility", "compatibility: claude-code\n", "claude-code"},
		{"metadata compat", "metadata:\n  compat: claude-code\n", "claude-code"},
		{"metadata compatibility", "metadata:\n  compatibility: opencode\n", "opencode"},
		{"metadata other", "metadata:\n  author: me\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			body := "---\nname: demo\ndescription: Does a thing.\n" + tc.front + "---\nBody.\n"
			if err := os.MkdirAll(filepath.Join(dir, "demo"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "demo", "SKILL.md"), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			loaded, err := skills.NewLoader([]string{dir}).LoadAll(dir, "")
			if err != nil {
				t.Fatal(err)
			}
			var got *skills.Skill
			for _, s := range loaded {
				if s.Name == "demo" {
					got = s
				}
			}
			if got == nil {
				t.Fatal("demo skill not loaded")
			}
			if got.Compat != tc.want {
				t.Errorf("Compat = %q, want %q", got.Compat, tc.want)
			}
		})
	}
}

func TestBuildSystemPromptSectionAddsMapping(t *testing.T) {
	section := skills.BuildSystemPromptSection([]*skills.Skill{
		{Name: "foreign", Description: "foreign workflow", Content: "`Task` and `Skill`"},
	})
	if !strings.Contains(section, "## Coddy tool mapping") {
		t.Fatalf("foreign skill section has no tool mapping:\n%s", section)
	}
	if !strings.Contains(section, "`Task` and `Skill`") {
		t.Fatal("skill body was lost")
	}
}
