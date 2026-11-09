# Bugfix with regression tests (Coddy workflow)

## Purpose

Fix a bug by **evidence, root cause, reproduction, narrow fix, and proof**:

1. establish expected vs actual behavior and a reproduction;
2. investigate before editing and name one credible cause;
3. write a failing reproduction test;
4. apply the narrowest fix, then run the affected tests and the full suite;
5. run lint/format/type gates;
6. report evidence, not just a success claim.

**You must describe the bug** (expected vs actual, how to reproduce, issue text). **`/rpa-bugfix` alone is not enough.**

## When to use

- User runs **`/rpa-bugfix`** and provides **bug details**.

## Workflow (summary)

1. Intake: expected, actual, reproduction, environment.
2. Investigate: trace the failure, rank hypotheses, name one cause.
3. Add a failing test that reproduces the bug.
4. Apply the smallest responsible fix.
5. Run the new test, nearby tests, and the full suite.
6. Run linter or pre-commit when configured.
7. Short factual report with evidence.

## Contents

| File       | Role                    |
|------------|-------------------------|
| `SKILL.md` | Metadata and full steps |

## Install

The upstream version of this skill is distributed through the **rpa-skills** catalog (a Claude Code plugin marketplace), and also installs as a plain **skill folder** (Cursor, OpenAI Codex, Kimi Code CLI, and others). This copy is maintained in the Coddy repository and adds Coddy's workflow gates.

**As a plugin (Claude Code)** — add the catalog once, then install this skill from it:

```text
/plugin marketplace add EvilFreelancer/rpa-skills
/plugin install rpa-bugfix@rpa-skills
```

**As a plain skill folder** — copy or symlink this repository into a skill root (its `SKILL.md` lives at the repo root):

| Tool          | Path                          |
|---------------|-------------------------------|
| Claude Code   | `~/.claude/skills/rpa-bugfix/`      |
| Cursor        | `~/.cursor/skills/rpa-bugfix/`      |
| OpenAI Codex  | `~/.codex/skills/rpa-bugfix/`       |
| Kimi Code CLI | `~/.kimi/skills/rpa-bugfix/`        |

The directory name must match the `name` field in `SKILL.md`.

## How to invoke

- **Slash command** — type `/rpa-bugfix` in agent chat.
- **`@` context** — attach the skill folder or `SKILL.md` to ground the message in these instructions.
- **Automatic** — the agent may load the skill on its own when your request matches the `description` in `SKILL.md`.

## Source & attribution

Upstream: **[rpa-skills](https://github.com/EvilFreelancer/rpa-skills)** — [Pavel Rykov](https://t.me/evilfreelancer)'s agent-skills collection (see [notes on vibe coding](https://t.me/evilfreelancer/1485)), packaged from **[cursor-vibe-prompts](https://github.com/EvilFreelancer/cursor-vibe-prompts)**.

This Coddy copy is maintained in the Coddy repository and may diverge from the upstream skill.

Licensed under the MIT License — see [LICENSE](LICENSE).
