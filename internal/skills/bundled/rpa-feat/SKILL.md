---
name: rpa-feat
metadata:
  version: 2.1.0
routes: the user asks for a new feature, a behavior change, or a refactor
description: >
  Run when the user invokes /rpa-feat together with a clear feature description (for example issue text),
  or when the routing line sent you here for a feature request.
  Coddy feature workflow: intent and non-goals, a written spec the user approves, a written plan with small
  verifiable tasks, RED-GREEN-REFACTOR, a ledger line per finished task, an independent review per task,
  the full suite, documentation, and a finish menu. Will not work correctly without the user supplying
  what to build.
---

# RPA feature work (Coddy)

## When this skill applies

The user includes **`/rpa-feat`** and a **concrete feature description** in the same message or right after:
scope, acceptance criteria, edge cases, and issue text when available. A request that matches the routing
line also starts here. If there is no actionable description, ask once for the minimum needed, then stop.

## Non-negotiable rules

1. **No implementation before a written acceptance test.** A throwaway spike is allowed only when the user
   asks for one; label it as a spike and do not ship it.
2. **One task at a time.** A task is small enough to review on its own: one behavior, the files it touches,
   its test, and its verification command.
3. **A gate is a stop.** The spec and the plan each wait for the user's answer. Do not start the next step
   while the previous one is waiting.
4. **A finished task is a ledger line.** Only the `sdd` tool writes it, and only after that task's test
   command exited 0. A task you believe is done but the ledger does not name is not done.
5. **Subagents are for independent work.** Dispatch only tasks that touch separate files or have no shared
   state. Keep integration and final verification in the main session.
6. **Evidence before "done".** Report commands and their actual results. A skipped, unavailable, or failing
   check is not a pass.
7. **Preserve unrelated behavior.** Do not mix refactors, renames, formatting sweeps, or dependency upgrades
   into the feature unless acceptance requires them.

## The artifacts

Everything lives in the repository, at the paths the workflow standardizes on. Another tool that follows
the same convention reads the same files.

| what | where |
|---|---|
| spec | `docs/superpowers/specs/<date>-<topic>-design.md` |
| plan | `docs/superpowers/plans/<date>-<feature>.md` |
| workspace, ledger, briefs, review packages | `.superpowers/sdd/<plan-name>/` |

The `sdd` tool owns the second and third rows. Its `plan` argument is the plan file, and the same spelling
must be used every time:

| action | what it does |
|---|---|
| `workspace` | returns the plan's directory |
| `brief` | slices task n into `task-<n>-brief.md` |
| `start` | does that and returns the base commit for the task |
| `done` | runs the task's test command and appends the ledger line when it exits 0 |
| `review` | writes `review-<base>..<head>.diff` for a branch range |

The ledger (`progress.md`) is the record that survives a context compaction or a new session. Read it
before you decide what is left; do not work from recollection.

## Workflow

### 1. Establish intent

Restate the feature in one short block:

- the user-visible outcome;
- acceptance criteria;
- explicit non-goals;
- constraints and compatibility requirements;
- one blocking question, only if the work cannot start without it.

### 2. Reconnaissance

Read the relevant code, docs, tests, and CI. Find the existing seam that owns the behavior; prefer
extending it over adding a parallel path. Note the project's test style and its real verification commands.

### 3. Spec gate

Write the spec: the outcome, the non-goals, the constraints, two or three concrete approaches with their
trade-offs and the one you chose, and the acceptance criteria the tests will encode.

Then **stop**. Tell the user the spec path and what you need from them, and wait for an answer before
writing any implementation. A change small enough that a separate spec would be ceremony may fold the
same content into the first section of the plan, and still stops here for the answer.

### 4. Plan gate

Write `docs/superpowers/plans/<date>-<feature>.md` with one row per task:

- behavior;
- files or ownership boundary;
- test to add or change;
- focused verification command;
- dependencies on other tasks.

No placeholders such as "add tests" or "update docs" without naming what changes. Add a **Review Focus**
section: the input classes and failure modes the tests do not cover, most damaging first. Then self-check
the plan the way the writing-plans discipline asks: every spec section has a task, every step names one
thing, the types and names agree across tasks.

Then **stop** again and wait for approval of the plan.

### 5. Execute one task

For each task in order:

1. Call `sdd` with action `start` and the task number. Read the brief it prints; the base commit it returns
   is the task's review range.
2. **RED.** Write the smallest tests that express the acceptance criteria and run them. Confirm they fail
   for the intended reason, not because of a typo, missing import, or broken fixture.
3. **GREEN.** Implement the narrowest complete change. Keep it inside the ownership boundary found in
   reconnaissance.
4. Call `sdd` with action `done`, the task number, the base commit, and the task's test command. The tool
   keeps the log and writes the ledger line only when the command exits 0. If it fails, fix the cause and
   run it again; nothing was recorded.

### 6. Review gate

Call `sdd` with action `review` for the task's `base..HEAD`. Spawn a fresh reviewer subagent with the
instructions in `reviewer-prompt.md` (beside this file), the review package path, and the spec and plan
paths. The reviewer never sees this conversation.

Sort its findings by what a user of the software would get. Critical and Important ones are fixed now, one
round, each with a failing test first and the whole focused suite green afterwards. Minor ones are not
fixed in this round: append a line to the ledger naming them as deferred. Record anything you decided not
to act on, with the reason.

### 7. Full regression

Run the repository's full test command, not only the new tests. Fix regressions before moving on. If the
full suite is unavailable or too expensive, run the documented focused gate and say exactly what was not
run and why.

### 8. Documentation and examples

Update the page, reference table, screenshot, example, changelog, or schema that describes the changed
behavior. If the repository requires a documentation stamp or generated artifact, regenerate it with the
documented command rather than editing generated blocks by hand.

### 9. Quality gate

Run the linter, formatter check, type check, and pre-commit gate the repository documents. Fix what they
report. Do not disable a check to make the change pass unless the user explicitly accepts that trade-off.

### 10. Finish gate

Do not merge, push, or delete anything on your own initiative. Offer the endings and let the user pick:

1. merge into the base branch locally;
2. push the branch and open a pull request;
3. leave the branch as it is.

Only the first ending cleans up the worktree, and only after the merged result passes the tests. Discarding
the work happens only when the user asks for it in so many words, and after a confirmation.

### 11. Report

Re-run the smallest sufficient proof set, read the output, then report:

- what changed;
- the ledger lines the `sdd` tool wrote;
- tests added or changed, and the full-suite and focused results;
- docs, examples, and generated artifacts touched;
- lint, format, and type-check results;
- anything not run, skipped, unavailable, or still risky.

## Red flags

- Writing implementation code before a failing acceptance test, or before the spec was approved.
- Starting the plan while the spec is still waiting for an answer.
- Reporting "tests pass" without the command and output.
- Calling a task done with no ledger line for it.
- Skipping the review gate because the change "looks simple".
- Dispatching subagents that edit the same files.
- Expanding scope because a nearby cleanup looked easy.
- Reporting "done" while the full suite or a documented gate is red.
