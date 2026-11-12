package sdd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// headSHA returns the current HEAD of a test repository.
func headSHA(t *testing.T, repo string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func shortSHA(t *testing.T, repo, rev string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", repo, "rev-parse", "--short=7", rev).Output()
	if err != nil {
		t.Fatalf("rev-parse %s: %v", rev, err)
	}
	return strings.TrimSpace(string(out))
}

// commitFile writes a file, stages everything and commits.
func commitFile(t *testing.T, repo, name, body, message string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-q", "-m", message}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func TestCompleteCreatesTheLedgerWithItsHeader(t *testing.T) {
	repo := tempRepo(t)
	plan := writePlan(t, repo, "docs/plans/demo.md", "# demo\n")
	dir, err := Workspace(repo, plan)
	if err != nil {
		t.Fatal(err)
	}
	base := headSHA(t, repo)

	line, err := Complete(repo, dir, "docs/plans/demo.md", 2, base, "go test ./...", "ok example")
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	head := shortSHA(t, repo, "HEAD")
	want := "Task 2: complete (commits " + shortSHA(t, repo, base) + ".." + head + ", tests: go test ./... → ok example)"
	if line != want {
		t.Fatalf("line = %q, want %q", line, want)
	}
	ledger, err := os.ReadFile(filepath.Join(dir, "progress.md"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(ledger), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("ledger = %q, want a header and one line", ledger)
	}
	if lines[0] != "# SDD ledger — plan: docs/plans/demo.md" {
		t.Fatalf("header = %q", lines[0])
	}
	if lines[1] != want {
		t.Fatalf("ledger line = %q, want %q", lines[1], want)
	}
}

func TestCompleteKeepsEarlierLines(t *testing.T) {
	repo := tempRepo(t)
	plan := writePlan(t, repo, "docs/plans/demo.md", "# demo\n")
	dir, err := Workspace(repo, plan)
	if err != nil {
		t.Fatal(err)
	}
	base := headSHA(t, repo)
	ledger := filepath.Join(dir, "progress.md")
	if err := os.WriteFile(ledger, []byte("# SDD ledger — plan: docs/plans/demo.md\nTask 1: complete (earlier)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Complete(repo, dir, "docs/plans/demo.md", 2, base, "go test ./...", "ok"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(ledger)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Count(text, "# SDD ledger") != 1 {
		t.Fatalf("header duplicated:\n%s", text)
	}
	first, second := strings.Index(text, "Task 1:"), strings.Index(text, "Task 2:")
	if first < 0 || second < 0 || second < first {
		t.Fatalf("ledger order is wrong:\n%s", text)
	}
}

func TestReviewPackageWritesTheThreeSections(t *testing.T) {
	repo := tempRepo(t)
	plan := writePlan(t, repo, "docs/plans/demo.md", "# demo\n")
	dir, err := Workspace(repo, plan)
	if err != nil {
		t.Fatal(err)
	}
	base := headSHA(t, repo)
	commitFile(t, repo, "one.txt", "one\n", "feat: one")
	commitFile(t, repo, "two.txt", "two\n", "feat: two")

	path, commits, size, err := ReviewPackage(repo, dir, base, "HEAD")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	want := "review-" + shortSHA(t, repo, base) + ".." + shortSHA(t, repo, "HEAD") + ".diff"
	if filepath.Base(path) != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	if commits != 2 {
		t.Fatalf("commits = %d, want 2", commits)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(body)) != size {
		t.Fatalf("size = %d, want %d", size, len(body))
	}
	for _, part := range []string{"# Review package: ", "## Commits", "## Files changed", "## Diff", "feat: one", "feat: two"} {
		if !strings.Contains(string(body), part) {
			t.Fatalf("package lacks %q:\n%s", part, body)
		}
	}
}

func TestReviewRefusesABogusRange(t *testing.T) {
	repo := tempRepo(t)
	plan := writePlan(t, repo, "docs/plans/demo.md", "# demo\n")
	dir, err := Workspace(repo, plan)
	if err != nil {
		t.Fatal(err)
	}
	head := headSHA(t, repo)

	// Empty range: base == head.
	if _, _, _, err := ReviewPackage(repo, dir, head, head); err == nil {
		t.Fatal("an empty range was accepted")
	}
	// HEAD is not a descendant of the other branch's tip.
	if out, err := exec.Command("git", "-C", repo, "checkout", "-q", "-b", "other").CombinedOutput(); err != nil {
		t.Fatalf("checkout: %v\n%s", err, out)
	}
	commitFile(t, repo, "other.txt", "other\n", "feat: other")
	other := headSHA(t, repo)
	if _, _, _, err := ReviewPackage(repo, dir, other, head); err == nil {
		t.Fatal("a range not rooted at base was accepted")
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "review-*.diff"))
	if len(matches) != 0 {
		t.Fatalf("a refused range wrote %v", matches)
	}
}
