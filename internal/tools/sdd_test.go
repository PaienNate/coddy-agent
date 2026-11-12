package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EvilFreelancer/coddy-agent/internal/config"
	"github.com/EvilFreelancer/coddy-agent/internal/tooling"
)

// sddRepo creates a git repository with one commit, a plan file and a package
// whose tests fail, so a scenario can tell a passing command from a failing one.
func sddRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	mustWrite(t, filepath.Join(dir, "docs", "plans", "demo.md"), "# demo\n\n### Task 1: the thing\n\nDo it.\n")
	mustWrite(t, filepath.Join(dir, "README.md"), "repo\n")
	run("add", ".")
	run("commit", "-q", "-m", "init")
	return dir
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func sddCall(t *testing.T, tool *tooling.Tool, env *tooling.Env, args map[string]interface{}) (string, error) {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	return tool.Execute(context.Background(), string(raw), env)
}

func TestSDDToolSchemaHasTheFiveActions(t *testing.T) {
	def := SDDTool().Definition
	if def.Name != SDDToolName {
		t.Fatalf("name = %q, want %q", def.Name, SDDToolName)
	}
	schema, _ := def.InputSchema.(map[string]interface{})
	props, _ := schema["properties"].(map[string]interface{})
	action, _ := props["action"].(map[string]interface{})
	values, _ := action["enum"].([]interface{})
	got := make([]string, 0, len(values))
	for _, v := range values {
		got = append(got, v.(string))
	}
	want := []string{"workspace", "brief", "start", "done", "review"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("actions = %v, want %v", got, want)
	}
}

func TestSDDWorkspaceAndStartRunAgainstATempRepo(t *testing.T) {
	repo := sddRepo(t)
	env := &tooling.Env{CWD: repo}
	tool := SDDTool()

	out, err := sddCall(t, tool, env, map[string]interface{}{"action": "workspace", "plan": "docs/plans/demo.md"})
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	if !strings.Contains(out, filepath.Join(".superpowers", "sdd", "demo")) {
		t.Fatalf("workspace output = %q", out)
	}

	out, err = sddCall(t, tool, env, map[string]interface{}{"action": "start", "plan": "docs/plans/demo.md", "task": 1})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if !strings.Contains(out, "brief: ") || !strings.Contains(out, "base: ") {
		t.Fatalf("start output = %q", out)
	}
}

func TestSDDDoneRecordsNothingWhenTheCommandFails(t *testing.T) {
	repo := sddRepo(t)
	env := &tooling.Env{CWD: repo}
	tool := SDDTool()
	plan := "docs/plans/demo.md"

	start, err := sddCall(t, tool, env, map[string]interface{}{"action": "start", "plan": plan, "task": 1})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	base := strings.TrimSpace(start[strings.Index(start, "base: ")+len("base: "):])

	_, err = sddCall(t, tool, env, map[string]interface{}{
		"action": "done", "plan": plan, "task": 1, "base": base, "command": "go test ./does-not-exist",
	})
	if err == nil {
		t.Fatal("a failing command was recorded")
	}
	ledger := filepath.Join(repo, ".superpowers", "sdd", "demo", "progress.md")
	if _, statErr := os.Stat(ledger); statErr == nil {
		body, _ := os.ReadFile(ledger)
		t.Fatalf("the ledger was written for a failing command:\n%s", body)
	}

	out, err := sddCall(t, tool, env, map[string]interface{}{
		"action": "done", "plan": plan, "task": 1, "base": base, "command": "go version",
	})
	if err != nil {
		t.Fatalf("done: %v", err)
	}
	if !strings.Contains(out, "ledger: Task 1: complete") {
		t.Fatalf("done output = %q", out)
	}
	body, err := os.ReadFile(ledger)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	if !strings.Contains(string(body), "Task 1: complete (commits ") || !strings.Contains(string(body), "go version") {
		t.Fatalf("ledger body = %q", body)
	}
}

// A user who turns auto-discovery off still has /rpa-feat: an explicit slash
// command injects the skill body without that switch, and the body calls sdd.
func TestRegistryOffersSDDWhateverAutoDiscoverySays(t *testing.T) {
	off := false
	r := NewRegistryFor(&config.Config{Skills: config.Skills{AutoDiscovery: &off}})
	if _, ok := r.Get(SDDToolName); !ok {
		t.Fatal("sdd is missing with auto-discovery off: an explicit /rpa-feat would call a tool that is not there")
	}
	if _, ok := r.Get("load_skill"); ok {
		t.Fatal("load_skill must stay behind auto-discovery")
	}
}
