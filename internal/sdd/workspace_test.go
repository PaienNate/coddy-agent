package sdd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// tempRepo creates a git repository with one commit and returns its root.
func tempRepo(t *testing.T) string {
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
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("repo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "init")
	return dir
}

func writePlan(t *testing.T, repo, rel, body string) string {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestWorkspaceCreatesThePlanDirectory(t *testing.T) {
	repo := tempRepo(t)
	plan := writePlan(t, repo, "docs/plans/demo.md", "# demo\n")

	dir, err := Workspace(repo, plan)
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	want := filepath.Join(repo, ".superpowers", "sdd", "demo")
	if dir != want {
		t.Fatalf("workspace = %q, want %q", dir, want)
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		t.Fatalf("workspace directory missing: %v", err)
	}
	marker, err := os.ReadFile(filepath.Join(dir, "plan-path"))
	if err != nil {
		t.Fatalf("plan-path: %v", err)
	}
	if got := strings.TrimSpace(string(marker)); got != "docs/plans/demo.md" {
		t.Fatalf("plan-path = %q, want the repo-relative plan", got)
	}
	ignore, err := os.ReadFile(filepath.Join(repo, ".superpowers", "sdd", ".gitignore"))
	if err != nil {
		t.Fatalf(".gitignore: %v", err)
	}
	if got := strings.TrimSpace(string(ignore)); got != "*" {
		t.Fatalf(".gitignore = %q, want *", got)
	}
}

func TestWorkspaceDisambiguatesSameBasename(t *testing.T) {
	repo := tempRepo(t)
	alpha := writePlan(t, repo, "docs/alpha/demo.md", "# alpha\n")
	beta := writePlan(t, repo, "docs/beta/demo.md", "# beta\n")

	first, err := Workspace(repo, alpha)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Workspace(repo, beta)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("two plans share the workspace %q", first)
	}
	if filepath.Base(second) != "demo-beta" {
		t.Fatalf("second workspace = %q, want the parent folder appended", filepath.Base(second))
	}
	marker, err := os.ReadFile(filepath.Join(second, "plan-path"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(marker)); got != "docs/beta/demo.md" {
		t.Fatalf("second plan-path = %q", got)
	}
}

func TestWorkspaceReusesItsOwnDirectory(t *testing.T) {
	repo := tempRepo(t)
	plan := writePlan(t, repo, "docs/plans/demo.md", "# demo\n")

	first, err := Workspace(repo, plan)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Workspace(repo, plan)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("workspace moved from %q to %q", first, second)
	}
}
