package tools

// The sdd tool carries the artifacts of the RPA workflow: the per-plan
// workspace, the task briefs, the progress ledger and the review packages. It
// is the Go counterpart of the shell helpers that workflow used to ship, and it
// exists as a tool rather than a CLI so every surface reaches it without a
// shell or a PATH lookup. internal/sdd holds the behavior; this file is the
// argument handling and the one command it runs.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/EvilFreelancer/coddy-agent/internal/gitws"
	"github.com/EvilFreelancer/coddy-agent/internal/llm"
	"github.com/EvilFreelancer/coddy-agent/internal/platform"
	"github.com/EvilFreelancer/coddy-agent/internal/sdd"
	"github.com/EvilFreelancer/coddy-agent/internal/tooling"
)

// SDDToolName is the name the model calls.
const SDDToolName = "sdd"

// SDDActionDone records a task, so it is the only action that runs a command.
const SDDActionDone = "done"

type sddArgs struct {
	Action  string `json:"action"`
	Plan    string `json:"plan"`
	Task    int    `json:"task"`
	Base    string `json:"base"`
	Head    string `json:"head"`
	Command string `json:"command"`
}

// SDDTool returns the workflow tool: workspace, brief, start, done and review.
func SDDTool() *tooling.Tool {
	description := "Work the RPA plan artifacts. `workspace` returns the plan's directory under .superpowers/sdd. " +
		"`brief` slices one task out of the plan into task-<n>-brief.md. `start` does that and prints the base commit. " +
		"`done` runs the task's test command and appends the ledger line only when it exits 0: a task is not complete until its test passes. " +
		"`review` writes review-<base>..<head>.diff for a branch range. " +
		"The plan file is the anchor of everything, so name it the same way every time."
	return &tooling.Tool{
		Definition: llm.ToolDefinition{
			Name:        SDDToolName,
			Description: description,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type": "string",
						"enum": []interface{}{"workspace", "brief", "start", "done", "review"},
					},
					"plan": map[string]interface{}{
						"type":        "string",
						"description": "Plan file, e.g. docs/superpowers/plans/2026-11-12-feature.md (relative to the working directory).",
					},
					"task": map[string]interface{}{
						"type":        "integer",
						"description": "Task number for brief, start and done.",
					},
					"base": map[string]interface{}{
						"type":        "string",
						"description": "Base commit for done and review: the sha start printed.",
					},
					"head": map[string]interface{}{
						"type":        "string",
						"description": "Head commit for review; defaults to HEAD.",
					},
					"command": map[string]interface{}{
						"type":        "string",
						"description": "The task's test command for done, run through the same shell as run_command.",
					},
				},
				"required": []interface{}{"action", "plan"},
			},
		},
		RequiresPermission: true,
		Execute: func(ctx context.Context, argsJSON string, env *tooling.Env) (string, error) {
			return executeSDD(ctx, argsJSON, env)
		},
	}
}

func executeSDD(ctx context.Context, argsJSON string, env *tooling.Env) (string, error) {
	var args sddArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("sdd: read arguments: %w", err)
	}
	cwd := ""
	if env != nil {
		cwd = env.CWD
	}
	repoRoot, err := gitws.Run(cwd, "rev-parse", "--show-toplevel")
	if err != nil || strings.TrimSpace(repoRoot) == "" {
		return "", fmt.Errorf("sdd: the working directory is not a git repository")
	}
	// The session's working directory resolves a relative plan, not the
	// process's: the two differ under coddy serve and in tests.
	planPath := args.Plan
	if !filepath.IsAbs(planPath) {
		planPath = filepath.Join(cwd, planPath)
	}
	dir, err := sdd.Workspace(repoRoot, planPath)
	if err != nil {
		return "", err
	}

	switch args.Action {
	case "workspace":
		return dir, nil

	case "brief":
		path, lines, err := sdd.WriteBrief(dir, planPath, args.Task)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("wrote %s: %d lines", path, lines), nil

	case "start":
		path, lines, err := sdd.WriteBrief(dir, planPath, args.Task)
		if err != nil {
			return "", err
		}
		base, err := gitws.Run(repoRoot, "rev-parse", "HEAD")
		if err != nil {
			return "", fmt.Errorf("sdd: base: %w", err)
		}
		return fmt.Sprintf("brief: %s (%d lines)\nbase: %s", path, lines, base), nil

	case "done":
		return executeSDDDone(ctx, args, dir, repoRoot, cwd)

	case "review":
		head := strings.TrimSpace(args.Head)
		if head == "" {
			head = "HEAD"
		}
		path, commits, size, err := sdd.ReviewPackage(repoRoot, dir, args.Base, head)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("wrote %s: %d commit(s), %d bytes", path, commits, size), nil

	default:
		return "", fmt.Errorf("sdd: unknown action %q", args.Action)
	}
}

// executeSDDDone runs the task's test command, keeps its output in the
// workspace, and records the task only when the command exited 0.
func executeSDDDone(ctx context.Context, args sddArgs, dir, repoRoot, cwd string) (string, error) {
	command := strings.TrimSpace(args.Command)
	if command == "" {
		return "", errors.New("sdd: done needs the task's test command")
	}
	logPath := filepath.Join(dir, fmt.Sprintf("task-%d-tests.log", args.Task))

	shell := platform.CurrentShell()
	name, argv := shell.Command(command)
	cmd := exec.CommandContext(ctx, name, argv...)
	cmd.Dir = cwd
	platform.AdaptCommand(cmd)
	out, runErr := cmd.CombinedOutput()
	if err := os.WriteFile(logPath, out, 0o644); err != nil {
		return "", fmt.Errorf("sdd: write the test log: %w", err)
	}
	tail := lastLines(string(out), 5)
	if runErr != nil {
		code := 1
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			code = exitErr.ExitCode()
		}
		return tail, fmt.Errorf("sdd: the test command exited %d; Task %d NOT recorded (full output: %s)",
			code, args.Task, logPath)
	}

	last := lastNonBlankLine(string(out))
	line, err := sdd.Complete(repoRoot, dir, args.Plan, args.Task, args.Base, command, last)
	if err != nil {
		return tail, err
	}
	return strings.TrimRight(tail+"\nledger: "+line, "\n"), nil
}

// lastLines returns the last n lines of s, for the model to read without the
// whole log it can open itself.
func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// lastNonBlankLine is what the ledger line quotes from the log.
func lastNonBlankLine(s string) string {
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return strings.TrimSpace(lines[i])
		}
	}
	return ""
}
