---
name: rpa-bugfix
metadata:
  version: 2.1.0
routes: the user reports a bug, a regression, or unexpected behavior
description: >
  Run when the user invokes /rpa-bugfix together with a clear bug description (expected vs actual, reproduction),
  or when the routing line sent you here for a defect report.
  Coddy bug workflow: gather evidence before editing, establish one credible root cause, write a failing reproduction
  test, apply the narrowest fix, record it in the ledger, get it reviewed, run the full suite, and prove completion.
  Will not work correctly without the bug details.
---

# RPA bugfix (Coddy)

## When this skill applies

The user must include **`/rpa-bugfix`** and a **bug description**: expected behavior, actual behavior, steps to reproduce, and environment when relevant. Paste issue text when possible. If reproduction is impossible without one missing fact, ask that single blocking question, then stop.

## Non-negotiable rules

1. **No fix before a credible cause.** A guess that happens to make the symptom disappear is not a diagnosis.
2. **No fix without a failing reproduction test**, unless the user explicitly accepts a documented reason why no test is possible.
3. **Change the narrowest layer that owns the incorrect behavior.** Do not mix cleanup, renaming, or unrelated refactoring into the fix.
4. **One bug at a time.** If investigation reveals a second independent defect, report it and ask before expanding scope.
5. **Evidence before "fixed".** Report the exact commands and results. A skipped or unavailable check is not a pass.
6. **A fixed bug is a ledger line.** Only the `sdd` tool writes it, and only after the reproduction test and the
   focused suite exited 0. A fix you believe works but the ledger does not name is not finished.
7. **One plan file is the anchor.** Even a single-fix bug gets a short plan under
   `docs/superpowers/plans/<date>-<bug>.md`: the bug note (symptom, reproduction, ranked hypotheses) as its
   first section, then one task per independent defect. The `sdd` tool takes that file as the plan argument.

## Workflow

### 1. Intake

Write down:

- expected behavior;
- actual behavior;
- reproduction steps;
- affected version, environment, and frequency;
- what the user already tried, if anything.

If one missing fact blocks reproduction, ask for exactly that fact. Do not ask a list of questions when one will do.

### 2. Investigate first

Gather evidence before changing product code:

- read the failing path, its callers, and the tests around it;
- trace inputs, state transitions, ownership boundaries, and error output;
- identify the smallest reproducible input;
- rank hypotheses by the evidence and by how cheaply they can be falsified.

Do not edit until one credible mechanism explains every observed symptom. If the evidence points to multiple causes, separate them and state which one this turn will address.

### 3. Write the bug plan and start the task

Write `docs/superpowers/plans/<date>-<bug>.md` with the bug note as its first section and one task per defect,
each naming its reproduction test and its verification command. Then call `sdd` with action `start` and the
task number: it slices the task's brief and returns the base commit the review will be cut from.

The ledger lives in `.superpowers/sdd/<plan-name>/progress.md`. Read it before deciding what is left after a
context compaction or a new session; do not work from recollection.

### 4. Reproduce with a test

Write a test that fails on the current code and encodes the expected behavior. Run it and confirm:

- it fails for the reason the report describes;
- it passes once the cause is corrected;
- it is not dependent on timing, network access, or a developer machine when the project's test style can avoid that.

If no test is possible, explain why the available proof is weaker and do not call the fix verified.

### 5. State the cause

In one short paragraph, name:

- the mechanism that produces the symptom;
- the evidence that proves it;
- why the proposed change addresses that mechanism rather than only the symptom.

If the cause is still uncertain, return to investigation instead of guessing.

### 6. Apply the surgical fix

Change the narrowest responsible layer. Preserve surrounding behavior, ordering, error handling, and public interfaces unless they are part of the bug. Keep the diff reviewable; do not rename or reformat unrelated code.

### 7. Verify and record

Run, in order:

1. the new reproduction test;
2. the nearest affected tests;
3. the repository's full test command;
4. the documented lint, format, type, and pre-commit gates.

Then call `sdd` with action `done`, the task number, the base commit and the command that proves the fix. The
tool keeps the log and writes the ledger line only when the command exits 0; a failing command records nothing.

If a check cannot run, say exactly what was not run and what risk remains.

### 8. Review gate

Call `sdd` with action `review` for `base..HEAD` and hand the package to a fresh reviewer subagent with the
instructions in `reviewer-prompt.md` (beside this file) plus the plan path. Critical and Important findings are
fixed now, one round, each with a failing test first. Minor ones are appended to the ledger as deferred.

### 9. Finish gate

Do not merge, push, or delete anything on your own initiative. Offer the endings and let the user pick: merge
into the base branch locally, push the branch and open a pull request, or leave the branch as it is. Only the
first cleans up the worktree, and only after the merged result passes the tests.

### 10. Report

Report:

- root cause and the evidence for it;
- the fix and the files touched, with the ledger line the `sdd` tool wrote;
- the reproduction test added or changed;
- focused and full-suite results;
- lint, format, and type-check results;
- anything not run, skipped, or still uncertain.

## Red flags

- "Quick fix" before the cause is understood.
- Adding a `try/catch`, retry, or null guard that hides the symptom without explaining it.
- Changing several independent behaviors in one fix.
- Reporting a pass from a test that was not run.
- Treating an intermittent failure as fixed because it did not reproduce once.
- Calling the bug fixed with no ledger line for the task.
- Skipping the review gate because the fix "is one line".
