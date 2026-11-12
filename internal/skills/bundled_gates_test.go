package skills

import "testing"

// The two flow skills carry the artifact contract: a version the delivery
// replaces, a routes line that puts them in the system prompt's routing hint,
// the paths the ledger and the plans live at, the sdd actions they call, and the
// reviewer brief each of them hands to a fresh subagent.
func TestTheFlowSkillsDeclareTheirGatesAndRoutes(t *testing.T) {
	byName := map[string]*Skill{}
	for _, sk := range Bundled() {
		byName[sk.Name] = sk
	}
	for _, name := range []string{"rpa-feat", "rpa-bugfix"} {
		sk, ok := byName[name]
		if !ok {
			t.Fatalf("%s is not in the delivery", name)
		}
		if sk.Version != "2.1.0" {
			t.Fatalf("%s version = %q, want 2.1.0", name, sk.Version)
		}
		if sk.Routes == "" {
			t.Fatalf("%s declares no routes line", name)
		}
		for _, want := range []string{
			"`.superpowers/sdd/", "docs/superpowers/plans/",
			"action `start`", "action `done`", "action `review`", "reviewer-prompt.md",
		} {
			if !contains(sk.Content, want) {
				t.Fatalf("%s does not name %q", name, want)
			}
		}
	}
	feat := byName["rpa-feat"]
	if !contains(feat.Content, "docs/superpowers/specs/") {
		t.Fatal("rpa-feat does not send the spec to docs/superpowers/specs/")
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
