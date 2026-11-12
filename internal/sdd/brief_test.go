package sdd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const twoTasks = `# Plan

Intro text.

### Task 1: add the thing

Do the first thing.

` + "```go" + `
### Task 2: not a heading, inside a fence
` + "```" + `

### Task 2: add the other thing

Do the second thing.

### Task 3: finish

Done.
`

func TestBriefSlicesOneTask(t *testing.T) {
	repo := tempRepo(t)
	plan := writePlan(t, repo, "docs/plans/demo.md", twoTasks)

	got, err := Brief(plan, 2)
	if err != nil {
		t.Fatalf("brief: %v", err)
	}
	if !strings.Contains(got, "### Task 2: add the other thing") {
		t.Fatalf("brief does not start at task 2:\n%s", got)
	}
	if strings.Contains(got, "### Task 3") || strings.Contains(got, "### Task 1") {
		t.Fatalf("brief carries another task:\n%s", got)
	}
}

func TestBriefReadsChineseHeadings(t *testing.T) {
	repo := tempRepo(t)
	plan := writePlan(t, repo, "docs/plans/zh.md", "### 任务 1：第一个\n\n内容一\n\n### 任务 2：第二个\n\n内容二\n")

	got, err := Brief(plan, 2)
	if err != nil {
		t.Fatalf("brief: %v", err)
	}
	if !strings.Contains(got, "任务 2") || strings.Contains(got, "任务 1") {
		t.Fatalf("brief = %q", got)
	}
}

func TestBriefIgnoresHeadingsInsideFences(t *testing.T) {
	repo := tempRepo(t)
	plan := writePlan(t, repo, "docs/plans/demo.md", twoTasks)

	first, err := Brief(plan, 1)
	if err != nil {
		t.Fatalf("task 1: %v", err)
	}
	if !strings.Contains(first, "### Task 2: not a heading, inside a fence") {
		t.Fatalf("the fenced line was dropped from task 1:\n%s", first)
	}
	if _, err := Brief(plan, 9); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("task 9 = %v, want ErrTaskNotFound", err)
	}
}

func TestWriteBriefNamesTheFile(t *testing.T) {
	repo := tempRepo(t)
	plan := writePlan(t, repo, "docs/plans/demo.md", twoTasks)
	dir, err := Workspace(repo, plan)
	if err != nil {
		t.Fatal(err)
	}

	path, lines, err := WriteBrief(dir, plan, 3)
	if err != nil {
		t.Fatalf("write brief: %v", err)
	}
	if want := filepath.Join(dir, "task-3-brief.md"); path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if lines != strings.Count(string(body), "\n")+1 && lines != strings.Count(string(body), "\n") {
		t.Fatalf("lines = %d for %d newlines", lines, strings.Count(string(body), "\n"))
	}
	if !strings.Contains(string(body), "### Task 3: finish") {
		t.Fatalf("brief body = %q", body)
	}
}
