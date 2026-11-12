package sdd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ErrTaskNotFound is returned when the plan has no section for the task.
var ErrTaskNotFound = errors.New("task not found in the plan")

// taskHeading matches the heading a plan marks a task with, in English or
// Chinese: "### Task 3: ..." and "### 任务 3：...".
var taskHeading = regexp.MustCompile(`^#+[ \t]+(Task|任务)[ \t]*([0-9]+)([^0-9]|$)`)

// Brief returns the plan's section for task n: the text from its heading up to
// the next task heading, with headings inside fenced code left alone.
func Brief(planArg string, n int) (string, error) {
	if n < 1 {
		return "", fmt.Errorf("%w: task %d", ErrTaskNotFound, n)
	}
	data, err := os.ReadFile(planArg)
	if err != nil {
		return "", fmt.Errorf("sdd: plan: %w", err)
	}
	var b strings.Builder
	inFence, collecting := false, false
	for _, line := range strings.SplitAfter(string(data), "\n") {
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
		} else if !inFence {
			if m := taskHeading.FindStringSubmatch(line); m != nil {
				if num, err := strconv.Atoi(m[2]); err == nil && num == n {
					collecting = true
				} else {
					collecting = false
				}
			}
		}
		if collecting {
			b.WriteString(line)
		}
	}
	body := b.String()
	if strings.TrimSpace(body) == "" {
		return "", fmt.Errorf("%w: task %d", ErrTaskNotFound, n)
	}
	return body, nil
}

// WriteBrief writes task n's brief into dir as task-<n>-brief.md and returns
// the path and the number of lines, the way the workflow's task-brief printed
// them.
func WriteBrief(dir, planArg string, n int) (path string, lines int, err error) {
	body, err := Brief(planArg, n)
	if err != nil {
		return "", 0, err
	}
	path = filepath.Join(dir, fmt.Sprintf("task-%d-brief.md", n))
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", 0, fmt.Errorf("sdd: write brief: %w", err)
	}
	return path, strings.Count(body, "\n"), nil
}
