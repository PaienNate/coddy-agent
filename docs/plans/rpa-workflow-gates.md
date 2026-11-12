# RPA workflow gates and artifacts

Design record, 2026-11-12.

The bundled `rpa-feat` and `rpa-bugfix` skills already carry the flow: intent,
reconnaissance, small tasks, RED-GREEN-REFACTOR, parallel subagents, the full
suite, documentation, quality gates and an evidence report. What they do not
carry is anything that makes a step a **gate**: nothing is written down, nothing
waits for a person, and nothing ties "done" to a test that actually passed.

The cost showed up in this repository: a session folded its context mid-run and
lost which tasks were finished; a reviewer had to be told the commit range by
hand; "tests pass" was a claim in a message rather than a line in a file. The
superpowers workflow solves this with artifacts and gates, and the bundled
skills were migrated without them.

This record adds the missing layer to the skills that exist. It does not rename
them, does not split them, and does not import another product's skill tree.

## 1. Decisions

- **R1: keep one entry per intent.** `/rpa-feat` and `/rpa-bugfix` stay the
  entry points; the gates and the artifact layer go inside them. No new skills,
  no stage commands, no upstream names.
- **The artifact paths are superpowers-compatible**, byte for byte:
  `docs/superpowers/specs/`, `docs/superpowers/plans/`, `.superpowers/sdd/`.
  A repository worked on by either tool keeps one tree, and a plan written by
  one can be executed by the other.
- **The helper is an agent tool, not a CLI.** `sdd` is registered beside
  `load_skill` and `compact_context`: structured arguments, no shell, no PATH
  lookup, available in every surface (console, HTTP, ACP, subagents). A
  `coddy sdd` subcommand is deliberately not added, so packaging (man page,
  completions, the command table) is untouched.
- **Auto-routing is data, not a hardcoded name.** A skill declares `routes:`
  in its frontmatter; the system prompt lists the skills that do, and the model
  loads the matching one before acting.

## 2. The artifact contract

Under the repository root (the `git rev-parse --show-toplevel` of the session
cwd):

```
docs/superpowers/
  specs/YYYY-MM-DD-<topic>-design.md      # written by the spec gate
  plans/YYYY-MM-DD-<feature>.md           # written by the plan gate
.superpowers/
  sdd/
    .gitignore                            # contains `*`: the tree never reaches git
    <plan-basename>/
      plan-path                           # owning plan, repo-relative
      progress.md                         # the ledger
      task-<n>-brief.md                   # one task, sliced out of the plan
      task-<n>-tests.log                  # the full output of the task's test command
      review-<base7>..<head7>.diff        # one review package per range
```

Formats that must not drift, because both the skills and the ledger's readers
depend on them:

- **Ledger header**, written when the file is created:
  `# SDD ledger — plan: <plan path as given>`
- **Completion line**, appended only after a passing test run:
  `Task <n>: complete (commits <base7>..<head7>, tests: <command> → <last non-blank line of the log>)`
- **Brief**: the plan's task section, sliced from a heading matching
  `^#+\s+(Task|任务)\s*<n>(\D|$)` outside fenced code, up to the next
  `Task`/`任务` heading.
- **Review package**: a `# Review package: <base>..<head>` heading, `## Commits`
  (`git log --oneline`), `## Files changed` (`git diff --stat`) and `## Diff`
  (`git diff -U10`).
- **Workspace names**: the plan's basename; a second plan with the same
  basename gets `-<parent folder>`, then `-<n>`. The `plan-path` marker decides
  which plan owns a directory.

## 3. The `sdd` tool

One tool, five actions, all taking the plan file as the anchor:

| action | arguments | result |
|---|---|---|
| `workspace` | `plan` | the workspace path, created and claimed for that plan |
| `brief` | `plan`, `task` | writes `task-<n>-brief.md`, returns its path and line count |
| `start` | `plan`, `task` | `brief`, then `base: <HEAD sha>` — the pair a task records |
| `done` | `plan`, `task`, `base`, `command` | runs the command, keeps the log, appends the ledger line only on exit 0 |
| `review` | `plan`, `base`, `head` | writes the review package for the range, returns its path and size |

Behavior carried over from the scripts the workflow uses today:

- `done` runs the command with `platform.CurrentShell()`, the way `run_command`
  does, so the same test command works on Linux, macOS and Windows. It prints
  the log's tail, returns the command's exit status as the tool's error on
  failure, and writes nothing to the ledger in that case.
- `review` refuses a range that is empty or not rooted at `base`
  (`base` must be an ancestor of `head`): a wrong-branch package is worse than
  no package.
- Every action fails with a message naming what is missing (no such plan,
  bad sha, task not found) rather than writing a partial file.

**Permission.** The tool is registered with `RequiresPermission`, and the
permission gate in `internal/agent/react.go` gets a branch for it: only
`action: "done"` can need approval, and it uses the same extraction and the same
session command grants as `run_command`, so an approved `go test ./...` covers
both. `workspace`, `brief`, `start` and `review` write only under
`.superpowers/` and ask nothing.

**Where the code lives.** `internal/sdd` holds the behavior (workspace
resolution, brief slicing, ledger append, review package) with unit tests and no
agent dependency; `internal/tools/sdd.go` is the tool wrapper; `gitws` gains a
thin exported `Run` so the git calls are not duplicated.

## 4. What the skills gain

`rpa-feat` (2.0.0 → 2.1.0) keeps its ten steps and gains five gates. Each gate
is a stop, not a heading:

- **G1, spec.** Before any code, write
  `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md`: outcomes, non-goals,
  constraints, the two or three approaches considered and the one chosen. Then
  stop and ask for approval. (For a change small enough that a spec would be
  ceremony, the skill says so and folds it into the plan's first section.)
- **G2, plan.** Write `docs/superpowers/plans/YYYY-MM-DD-<feature>.md` with the
  task structure the flow already asks for, plus a **Review Focus** list (the
  failure modes the tests do not cover) and a self-check pass. Then stop and ask
  for approval before executing.
- **G3, per-task evidence.** For each task: `sdd start` (brief + base), then
  RED, GREEN, then `sdd done` with that task's test command. A task whose test
  command fails is not complete; the ledger says nothing about it.
- **G4, per-task review.** After `sdd done`, `sdd review` for
  `<base>..<HEAD>`, and a fresh reviewer subagent reads the package. The
  reviewer's brief is a new file beside the skill, `reviewer-prompt.md`
  (written here, in this repository's words, not copied from another product):
  read the package, report Critical/Important/Minor with file:line, name what
  it declined to judge. Critical and Important findings are fixed in one round
  with a failing test each; Minor findings are recorded in the ledger as
  deferred.
- **G5, finish.** When every task is recorded, offer the three endings — merge
  locally, push and open a PR, or leave the branch — and clean up the worktree
  only in the first case (with the discard confirmation for the destructive
  path).

`rpa-bugfix` (2.0.0 → 2.1.0) gets the same ledger and the same G3/G4/G5, with a
one-task plan file as the anchor (a bug with several independent fixes gets
several tasks). Its existing "no fix before a credible cause" and "no fix
without a failing reproduction" stay as written; G1 is skipped, the bug note
(a symptom, a reproduction, ranked hypotheses) is the plan's first section.

Both skills also say, in one place, what the ledger is for: a compaction or a
new session resumes from `progress.md`, not from recollection.

## 5. Auto-routing

`Skill` gains one frontmatter field, `routes` — a one-line "use this when".
The skills that carry it are listed in the system prompt after the slash
catalog (`internal/agent/system_prompt.go`), replacing today's generic
`loadSkillHint` when at least one skill declares a route:

> Before you create, change, debug or review code, load the flow skill that
> matches and follow it: `rpa-feat` — <its routes line>; `rpa-bugfix` — <its>; …

With none declared, the current hint stays. The line is rendered per rules
generation with the rest of the standing prompt, so it does not move the cached
prefix between turns.

`rpa-init` and `rpa-gen-rules` carry no `routes` line: they are explicit
requests, not part of the build flow.

## 6. Compatibility and migration

- The two skills keep their names and their slash entries; an operator who only
  types `/rpa-feat` sees the same command do a stricter job.
- The version bump makes the delivery replace the copies already on disk
  (`internal/skills/seed.go` replaces a lower `metadata.version`). A local edit
  to a delivered skill is replaced by that rule, as it is for every delivered
  skill today.
- A repository that already has `.superpowers/sdd/` from superpowers keeps it:
  same paths, same ledger format, same brief and review names.
- `.superpowers/sdd/.gitignore` with `*` keeps the tree out of `git status`
  without touching the checkout's tracked `.gitignore`.

## 7. Non-goals

- No `coddy sdd` CLI subcommand, and no `/sdd` slash command (a read-only status
  command can come later on the same package).
- No splitting into stage skills (that was R2; declined for this round).
- No upstream prose or upstream skill names in the bundle.
- No change to `rpa-init` or `rpa-gen-rules`.
- No change to how skills are delivered, trusted or disabled.

## 8. Rollback

The tool is additive and unused until a skill calls it; the skill text change is
one delivery update, and `skills disable rpa-feat` (or editing the copies) turns
the new flow off. The artifact tree is inert without the skills: deleting
`.superpowers/` and the two `docs/superpowers/` folders leaves the repository as
it was.

## 9. Tests

- `internal/sdd`: workspace resolution (basename, marker, collision, self
  ignore), brief slicing (English and Chinese headings, fenced code, not-found),
  ledger append (only on pass, header creation, exact line), review package
  (three guards, file body, short names).
- `internal/tools`: the `sdd` tool's schema and its five actions against a temp
  git repository; a failing command leaves the ledger untouched.
- `internal/agent`: the permission branch — `done` asks unless the command is
  granted, every other action never asks; bypass mode never asks.
- `internal/agent`: the routing line appears when a skill declares `routes`,
  and the old hint appears when none does.
- `internal/skills`: the two bundled skills parse, declare their new versions,
  and name every `sdd` action they use.

## 10. Documentation

- `docs/features/skills.md`: the two rows in the delivery table gain the gates
  and the artifact tree; a new section, "Artifacts and gates", shows the tree and
  the ledger line. Russian and Chinese translations in the same pull request,
  restamped with `make docs-stamp`.
- No CLI reference changes (no new command), no screenshot: the surface is the
  agent's transcript, not a new UI.
