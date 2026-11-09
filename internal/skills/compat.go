package skills

import (
	"regexp"
	"strings"
)

// foreignToolName matches names from Claude Code, Codex and the other agents
// Superpowers targets. Only names that are unlikely to appear as ordinary
// English words are used for detection; a body that says "read the file" is
// not treated as a foreign skill just because it has the verb "read".
//
// A skill can skip detection and declare its vocabulary directly:
//
//	compatibility: claude-code
//
// or:
//
//	metadata:
//	  compat: claude-code
//
// "coddy" and "native" explicitly mean no mapping.
var foreignToolName = regexp.MustCompile(`\b(Task|TodoWrite|Skill|EnterPlanMode|ExitPlanMode|AskUserQuestion|WebFetch|WebSearch)\b`)

const coddyToolMapping = "## Coddy tool mapping\n\n" +
	"This skill uses another agent's tool names. In Coddy:\n\n" +
	"- `Skill` -> `load_skill`\n" +
	"- `Task` / `Agent` -> `spawn_agent` (`agent: \"explore\"` for read-only exploration)\n" +
	"- `TodoWrite` -> `coddy_todo_plan_replace`, `coddy_todo_item_update`, `coddy_todo_plan_read`\n" +
	"- `Read` -> `read`; `Write` -> `write`; `Edit` -> `edit`\n" +
	"- `Bash` -> `run_command`; `Grep` -> `grep`; `Glob` -> `glob`\n" +
	"- `WebFetch` -> `webfetch`; `WebSearch` -> `websearch`\n" +
	"- `AskUserQuestion` -> `question`\n" +
	"- `EnterPlanMode` -> Coddy plan mode (`/plan`); `ExitPlanMode` -> `plan_exit`\n\n" +
	"If a mapped tool is unavailable, use the closest native Coddy tool or do the\n" +
	"action inline. Never invent a tool call. If no subagent tool is available,\n" +
	"execute sequentially instead of pretending to dispatch one.\n"

// NeedsToolCompat reports whether a skill body was written for another agent
// and therefore needs the Coddy tool mapping. An explicit Compat value wins;
// otherwise a small set of unambiguous foreign tool names triggers it.
func NeedsToolCompat(s *Skill) bool {
	if s == nil {
		return false
	}
	switch normalizeCompat(s.Compat) {
	case "coddy", "native":
		return false
	case "":
		return foreignToolName.MatchString(s.Content)
	default:
		return true
	}
}

// WithToolCompat prepends the Coddy tool mapping to a foreign skill body. It
// is idempotent: a body that already carries the mapping is returned as is.
func WithToolCompat(s *Skill, body string) string {
	if !NeedsToolCompat(s) || strings.TrimSpace(body) == "" {
		return body
	}
	if strings.Contains(body, "## Coddy tool mapping") {
		return body
	}
	return coddyToolMapping + "\n" + body
}
