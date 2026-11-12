package agent

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/EvilFreelancer/coddy-agent/internal/skills"
)

func flowSkill(name, routes string) *skills.Skill {
	return &skills.Skill{
		Name:        name,
		FilePath:    filepath.Join("skills", name, "SKILL.md"),
		Description: "flow " + name,
		Routes:      routes,
		Content:     "body",
	}
}

func TestSkillsPromptNamesTheDeclaredRoutes(t *testing.T) {
	loaded := []*skills.Skill{
		flowSkill("rpa-feat", "the user asks for a new feature"),
		flowSkill("rpa-bugfix", "the user reports a bug"),
	}
	active := skills.FilterForContext(loaded, nil)

	out := buildSkillsPromptMarkdown(loaded, active, true)
	if !strings.Contains(out, "flow skill that matches") {
		t.Fatalf("no routing line:\n%s", out)
	}
	for _, want := range []string{"`rpa-feat`", "the user asks for a new feature", "`rpa-bugfix`", "the user reports a bug"} {
		if !strings.Contains(out, want) {
			t.Fatalf("routing line lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, loadSkillHint) {
		t.Fatalf("the generic hint stayed next to the routing line:\n%s", out)
	}
}

func TestSkillsPromptKeepsTheGenericHintWithoutRoutes(t *testing.T) {
	loaded := []*skills.Skill{flowSkill("code-review", "")}
	active := skills.FilterForContext(loaded, nil)

	out := buildSkillsPromptMarkdown(loaded, active, true)
	if !strings.Contains(out, loadSkillHint) {
		t.Fatalf("the generic hint is gone:\n%s", out)
	}
	if strings.Contains(out, "flow skill") {
		t.Fatalf("an empty routing line was rendered:\n%s", out)
	}
}

func TestSkillsPromptRoutesOnlyWhenAutoDiscoveryIsOn(t *testing.T) {
	loaded := []*skills.Skill{flowSkill("rpa-feat", "the user asks for a new feature")}
	active := skills.FilterForContext(loaded, nil)

	if out := buildSkillsPromptMarkdown(loaded, active, false); strings.Contains(out, "flow skill") {
		t.Fatalf("routing line rendered without auto-discovery:\n%s", out)
	}
}
