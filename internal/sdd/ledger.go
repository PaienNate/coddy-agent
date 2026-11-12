package sdd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/EvilFreelancer/coddy-agent/internal/gitws"
)

// ErrBadRange is returned for a review range that is empty or not rooted at the
// base commit. A wrong-branch package is worse than no package.
var ErrBadRange = errors.New("review range is empty or not rooted at base")

// Complete appends one task's completion line to the ledger in dir and returns
// it. The line names the range BASE..HEAD and the test command that passed;
// callers record it only after the command exited 0. A ledger that does not
// exist yet gets its header first.
func Complete(repoRoot, dir, planArg string, n int, base, command, lastLine string) (string, error) {
	if strings.TrimSpace(base) == "" {
		return "", errors.New("sdd: no base commit")
	}
	if _, err := gitws.Run(repoRoot, "rev-parse", "--verify", "--quiet", base); err != nil {
		return "", fmt.Errorf("sdd: bad base %q: %w", base, err)
	}
	shortBase, err := gitws.Run(repoRoot, "rev-parse", "--short=7", base)
	if err != nil {
		return "", fmt.Errorf("sdd: base: %w", err)
	}
	shortHead, err := gitws.Run(repoRoot, "rev-parse", "--short=7", "HEAD")
	if err != nil {
		return "", fmt.Errorf("sdd: head: %w", err)
	}
	line := fmt.Sprintf("Task %d: complete (commits %s..%s, tests: %s → %s)",
		n, shortBase, shortHead, command, strings.TrimSpace(lastLine))

	ledger := filepath.Join(dir, "progress.md")
	if _, err := os.Stat(ledger); errors.Is(err, os.ErrNotExist) {
		header := fmt.Sprintf("# SDD ledger — plan: %s\n", planArg)
		if err := os.WriteFile(ledger, []byte(header), 0o644); err != nil {
			return "", fmt.Errorf("sdd: ledger: %w", err)
		}
	}
	f, err := os.OpenFile(ledger, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("sdd: ledger: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(line + "\n"); err != nil {
		return "", fmt.Errorf("sdd: ledger: %w", err)
	}
	return line, nil
}

// ReviewPackage writes the review package for base..head into dir and returns
// its path, the number of commits in the range and the file's size. The file
// opens with the range, then the commit list, the stat summary and the net diff
// with ten lines of context.
func ReviewPackage(repoRoot, dir, base, head string) (path string, commits int, size int64, err error) {
	if _, err := gitws.Run(repoRoot, "rev-parse", "--verify", "--quiet", base); err != nil {
		return "", 0, 0, fmt.Errorf("sdd: bad base %q: %w", base, err)
	}
	if _, err := gitws.Run(repoRoot, "rev-parse", "--verify", "--quiet", head); err != nil {
		return "", 0, 0, fmt.Errorf("sdd: bad head %q: %w", head, err)
	}
	if _, err := gitws.Run(repoRoot, "merge-base", "--is-ancestor", base, head); err != nil {
		return "", 0, 0, fmt.Errorf("%w: %s..%s", ErrBadRange, base, head)
	}
	count, err := gitws.Run(repoRoot, "rev-list", "--count", base+".."+head)
	if err != nil {
		return "", 0, 0, fmt.Errorf("sdd: range: %w", err)
	}
	if strings.TrimSpace(count) == "0" {
		return "", 0, 0, fmt.Errorf("%w: %s..%s", ErrBadRange, base, head)
	}
	log, err := gitws.Run(repoRoot, "log", "--oneline", base+".."+head)
	if err != nil {
		return "", 0, 0, fmt.Errorf("sdd: log: %w", err)
	}
	stat, err := gitws.Run(repoRoot, "diff", "--stat", base+".."+head)
	if err != nil {
		return "", 0, 0, fmt.Errorf("sdd: stat: %w", err)
	}
	diff, err := gitws.Run(repoRoot, "diff", "-U10", base+".."+head)
	if err != nil {
		return "", 0, 0, fmt.Errorf("sdd: diff: %w", err)
	}
	shortBase, err := gitws.Run(repoRoot, "rev-parse", "--short", base)
	if err != nil {
		return "", 0, 0, fmt.Errorf("sdd: base: %w", err)
	}
	shortHead, err := gitws.Run(repoRoot, "rev-parse", "--short", head)
	if err != nil {
		return "", 0, 0, fmt.Errorf("sdd: head: %w", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Review package: %s..%s\n\n", base, head)
	b.WriteString("## Commits\n")
	b.WriteString(log)
	b.WriteString("\n\n## Files changed\n")
	b.WriteString(stat)
	b.WriteString("\n\n## Diff\n")
	b.WriteString(diff)
	b.WriteString("\n")

	path = filepath.Join(dir, fmt.Sprintf("review-%s..%s.diff", shortBase, shortHead))
	body := b.String()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", 0, 0, fmt.Errorf("sdd: review package: %w", err)
	}
	return path, len(strings.Split(strings.TrimSpace(log), "\n")), int64(len(body)), nil
}
