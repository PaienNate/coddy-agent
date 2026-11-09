---
name: rpa-feat
metadata:
  version: 2.0.0
description: >
  Run when the user invokes /rpa-feat together with a clear feature description (for example issue text).
  Coddy feature workflow: establish intent and non-goals, plan in small verifiable tasks, RED-GREEN-REFACTOR,
  use subagents only for independent tasks, run the full suite, update docs and examples, then prove completion.
  Will not work correctly without the user supplying what to build.
---

# RPA feature work (Coddy)

## When this skill applies

The user must include **`/rpa-feat`** and a **concrete feature description** in the same message or immediately after: scope, acceptance criteria, edge cases, and issue text when available. If there is no actionable description, ask once for the minimum needed, then stop.

## Non-negotiable rules

1. **No implementation before a written acceptance test.** A throwaway spike is allowed only when the user asks for one; label it as a spike and do not ship it.
2. **One task at a time.** A task is small enough to review on its own: one behavior, the files it touches, its test, and its verification command.
3. **Subagents are for independent work.** Dispatch only tasks that touch separate files or have no shared state. Keep integration and final verification in the main session.
4. **Evidence before "done".** Report commands and their actual results. A skipped, unavailable, or failing check is not a pass.
5. **Preserve unrelated behavior.** Do not mix refactors, renames, formatting sweeps, or dependency upgrades into the feature unless acceptance requires them.

## Workflow

### 1. Establish intent

Restate the feature in one short block:

- the user-visible outcome;
- acceptance criteria;
- explicit non-goals;
- constraints and compatibility requirements;
- one blocking question, only if the work cannot start without it.

If the request is genuinely ambiguous about the outcome, use the brainstorming step: compare two or three concrete approaches, state the trade-off, and ask the user to choose. Do not ask about details the repository already answers.

### 2. Reconnaissance

Read the relevant code, docs, tests, and CI. Find the existing seam that owns the behavior; prefer extending it over adding a parallel path. Note the project's test style and its real verification commands.

### 3. Plan in small tasks

Write a task list (use the session's plan/todo tool when available) with one row per task:

- behavior;
- files or ownership boundary;
- test to add or change;
- focused verification command;
- dependencies on other tasks.

No placeholders such as "add tests" or "update docs" without naming what changes. If a task cannot be verified, split it until it can.

### 4. RED: specify the behavior

Write the smallest tests that express the acceptance criteria. Match the project's test framework and naming. Run them and confirm they fail for the intended reason, not because of a typo, missing import, or broken fixture.

### 5. GREEN: implement the narrowest complete change

Implement until the new tests pass. Keep the change inside the ownership boundary found in reconnaissance. Refactor duplicated behavior when patching it would weaken ownership or hide the cause; do not refactor merely for taste.

### 6. Parallelize only when independent

When the plan has genuinely independent tasks, dispatch them with `spawn_agent` (or the host's equivalent subagent tool) in one wave. Give each child the exact task, files, tests, and verification command. Do not let two children edit the same file or depend on each other's uncommitted work. Review and integrate each result yourself.

### 7. Full regression

Run the repository's full test command, not only the new tests. Fix regressions before moving on. If the full suite is unavailable or too expensive, run the documented focused gate and say exactly what was not run and why.

### 8. Documentation and examples

Update the page, reference table, screenshot, example, changelog, or schema that describes the changed behavior. If the repository requires a documentation stamp or generated artifact, regenerate it with the documented command rather than editing generated blocks by hand.

### 9. Quality gate

Run the linter, formatter check, type check, and pre-commit gate the repository documents. Fix what they report. Do not disable a check to make the change pass unless the user explicitly accepts that trade-off.

### 10. Verify and report

Before claiming completion, re-run the smallest sufficient proof set and confirm the output. Then report:

- what changed;
- tests added or changed;
- full-suite and focused results;
- docs, examples, and generated artifacts touched;
- lint, format, and type-check results;
- anything not run, skipped, unavailable, or still risky.

## Red flags

- Writing implementation code before a failing acceptance test.
- Reporting "tests pass" without the command and output.
- Dispatching subagents that edit the same files.
- Expanding scope because a nearby cleanup looked easy.
- Treating a spike as production code.
- Saying "done" while the full suite or a documented gate is red.
