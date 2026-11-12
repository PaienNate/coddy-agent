// Package sdd holds the artifacts of the RPA workflow: the per-plan workspace
// under .superpowers/sdd, the task brief sliced out of a plan, the progress
// ledger and review packages. It is the Go counterpart of the shell helpers the
// workflow used to ship, and the paths and file formats are the same, so a
// repository worked on by either tool keeps one tree.
package sdd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// workspaceRoot is the folder under the repository root that holds one
// directory per plan. Its .gitignore keeps the whole tree out of git status
// without touching the checkout's tracked .gitignore.
const workspaceRoot = ".superpowers/sdd"

// Workspace returns the directory of planArg's artifacts, creating it and
// claiming it for that plan. Two plans with the same file name do not share a
// directory: the plan-path marker decides, and a taken name gets the plan's
// parent folder appended, then a counter.
//
// repoRoot is the repository the workspace lives in (git rev-parse
// --show-toplevel); planArg may be relative to the process working directory.
func Workspace(repoRoot, planArg string) (string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return "", errors.New("sdd: no repository root")
	}
	if strings.TrimSpace(planArg) == "" {
		return "", errors.New("sdd: no plan file")
	}
	planAbs, err := filepath.Abs(planArg)
	if err != nil {
		return "", fmt.Errorf("sdd: plan path: %w", err)
	}
	if fi, err := os.Stat(planAbs); err != nil || fi.IsDir() {
		return "", fmt.Errorf("sdd: no such plan file: %s", planArg)
	}

	slug := strings.TrimSuffix(filepath.Base(planAbs), filepath.Ext(planAbs))
	if slug == "" || slug == "." || slug == ".." {
		return "", fmt.Errorf("sdd: cannot derive a workspace name from %s", planArg)
	}

	planID := planAbs
	if rel, err := filepath.Rel(repoRoot, planAbs); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		planID = rel
	}

	base := filepath.Join(repoRoot, filepath.FromSlash(workspaceRoot))
	claim := func(dir string) (bool, error) {
		marker := filepath.Join(dir, "plan-path")
		if b, err := os.ReadFile(marker); err == nil {
			return strings.TrimSpace(string(b)) == planID, nil
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return false, err
		}
		if err := os.WriteFile(marker, []byte(planID+"\n"), 0o644); err != nil {
			return false, err
		}
		return true, nil
	}

	dir := filepath.Join(base, slug)
	owned, err := claim(dir)
	if err != nil {
		return "", fmt.Errorf("sdd: workspace: %w", err)
	}
	if !owned {
		parent := filepath.Base(filepath.Dir(planAbs))
		dir = filepath.Join(base, slug+"-"+parent)
		if owned, err = claim(dir); err != nil {
			return "", fmt.Errorf("sdd: workspace: %w", err)
		}
		for n := 2; !owned; n++ {
			dir = filepath.Join(base, slug+"-"+parent+"-"+strconv.Itoa(n))
			if owned, err = claim(dir); err != nil {
				return "", fmt.Errorf("sdd: workspace: %w", err)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(base, ".gitignore"), []byte("*\n"), 0o644); err != nil {
		return "", fmt.Errorf("sdd: workspace .gitignore: %w", err)
	}
	return dir, nil
}
