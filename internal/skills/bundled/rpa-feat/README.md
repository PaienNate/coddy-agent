# New feature (Coddy workflow)

## Purpose

Add a feature by **intent, plan, RED-GREEN-REFACTOR, regression, documentation, and proof**:

1. establish the outcome, acceptance criteria, and non-goals;
2. study code, docs, and tests;
3. plan small verifiable tasks;
4. add failing tests, then implementation;
5. dispatch independent tasks to subagents only when they do not share files or state;
6. run the full suite, update docs and examples, and run the quality gates;
7. report commands and actual results before claiming completion.

**You must pass what to build** (for example issue text, acceptance criteria). **`/rpa-feat` alone is not enough.**

## When to use

- User runs **`/rpa-feat`** and supplies a **feature description** in the same turn or right after.

## Workflow (summary)

1. Establish intent, acceptance criteria, and non-goals.
2. Study code, docs, and tests; find the owning seam.
3. Plan small tasks with behavior, files, test, and verification.
4. Write failing tests, then implement the narrowest complete change.
5. Parallelize only independent tasks; integrate and review them yourself.
6. Run the full suite, update docs and examples, and run lint/format/type gates.
7. Verify with evidence and report what was not run.

## Contents

| File       | Role                    |
|------------|-------------------------|
| `SKILL.md` | Metadata and full steps |


## Install

The upstream version of this skill is distributed through the **rpa-skills** catalog (a Claude Code plugin marketplace), and also installs as a plain **skill folder** (Cursor, OpenAI Codex, Kimi Code CLI, and others). This copy is maintained in the Coddy repository and adds Coddy's workflow gates.

**As a plugin (Claude Code)** — add the catalog once, then install this skill from it:

```text
/plugin marketplace add EvilFreelancer/rpa-skills
/plugin install rpa-feat@rpa-skills
```

**As a plain skill folder** — copy or symlink this repository into a skill root (its `SKILL.md` lives at the repo root):

| Tool          | Path                          |
|---------------|-------------------------------|
| Claude Code   | `~/.claude/skills/rpa-feat/`      |
| Cursor        | `~/.cursor/skills/rpa-feat/`      |
| OpenAI Codex  | `~/.codex/skills/rpa-feat/`       |
| Kimi Code CLI | `~/.kimi/skills/rpa-feat/`        |

The directory name must match the `name` field in `SKILL.md`.

## How to invoke

- **Slash command** — type `/rpa-feat` in agent chat.
- **`@` context** — attach the skill folder or `SKILL.md` to ground the message in these instructions.
- **Automatic** — the agent may load the skill on its own when your request matches the `description` in `SKILL.md`.

## Source & attribution

Upstream: **[rpa-skills](https://github.com/EvilFreelancer/rpa-skills)** — [Pavel Rykov](https://t.me/evilfreelancer)'s agent-skills collection (see [notes on vibe coding](https://t.me/evilfreelancer/1485)), packaged from **[cursor-vibe-prompts](https://github.com/EvilFreelancer/cursor-vibe-prompts)**.

This Coddy copy is maintained in the Coddy repository and may diverge from the upstream skill.

Licensed under the MIT License — see [LICENSE](LICENSE).
