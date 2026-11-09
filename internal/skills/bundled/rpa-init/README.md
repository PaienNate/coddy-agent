# Project onboarding (Coddy workflow)

## Purpose

Warm up context on a repository: study **code**, read **documentation** and **test code**, establish an evidence-based baseline, **set up the dev environment** the project expects, **run tests**, and write a **short report**. No extra brief from the user is required.

## When to use

- User runs **`/rpa-init`** or asks to onboard / understand a codebase.

## What the agent does

1. Optionally `git pull` if the repo is under git and the user has not asked to stay on the current commit.
2. Record branch, commit, dirty state, and the environment it actually used.
3. Read layout, docs, and tests.
4. Install or configure dev tooling (venv, deps, documented bootstrap).
5. Run the test suite and distinguish pass, fail, unavailable, and blocked.
6. Summarize purpose, behavior implied by tests, gaps, and next steps without fixing code.

## Contents

| File       | Role                          |
|------------|-------------------------------|
| `SKILL.md` | Metadata and full workflow    |


## Install

The upstream version of this skill is distributed through the **rpa-skills** catalog (a Claude Code plugin marketplace), and also installs as a plain **skill folder** (Cursor, OpenAI Codex, Kimi Code CLI, and others). This copy is maintained in the Coddy repository and adds Coddy's evidence rules.

**As a plugin (Claude Code)** — add the catalog once, then install this skill from it:

```text
/plugin marketplace add EvilFreelancer/rpa-skills
/plugin install rpa-init@rpa-skills
```

**As a plain skill folder** — copy or symlink this repository into a skill root (its `SKILL.md` lives at the repo root):

| Tool          | Path                          |
|---------------|-------------------------------|
| Claude Code   | `~/.claude/skills/rpa-init/`      |
| Cursor        | `~/.cursor/skills/rpa-init/`      |
| OpenAI Codex  | `~/.codex/skills/rpa-init/`       |
| Kimi Code CLI | `~/.kimi/skills/rpa-init/`        |

The directory name must match the `name` field in `SKILL.md`.

## How to invoke

- **Slash command** — type `/rpa-init` in agent chat.
- **`@` context** — attach the skill folder or `SKILL.md` to ground the message in these instructions.
- **Automatic** — the agent may load the skill on its own when your request matches the `description` in `SKILL.md`.

## Source & attribution

Upstream: **[rpa-skills](https://github.com/EvilFreelancer/rpa-skills)** — [Pavel Rykov](https://t.me/evilfreelancer)'s agent-skills collection (see [notes on vibe coding](https://t.me/evilfreelancer/1485)), packaged from **[cursor-vibe-prompts](https://github.com/EvilFreelancer/cursor-vibe-prompts)**.

This Coddy copy is maintained in the Coddy repository and may diverge from the upstream skill.

Licensed under the MIT License — see [LICENSE](LICENSE).
