---
name: rpa-init
metadata:
  version: 1.1.0
description: >
  Run when the user invokes /rpa-init or asks to onboard or warm up context on a repository.
  The agent studies code, documentation and test code, establishes an evidence-based baseline,
  sets up the documented dev environment, runs the tests, and writes a short project report.
  No extra user brief is required.
---

# RPA project initialization (Coddy)

## Intent

Treat automated tests as the project's long-term memory. Initialization maps **documented intent**, **behavior encoded in tests**, and **implementation**. Prefer learning from tests and docs before inferring only from production code.

## Evidence rules

- Separate **observed** facts from **inferred** explanations. Cite the file, command, or output that supports each important claim.
- Record the repository state you actually inspected: branch, commit, dirty files, and any generated or ignored paths that affect the result.
- Distinguish **pass**, **fail**, **unavailable**, and **blocked**. Never turn "could not run" into "looks fine".
- Do not fix code, tests, dependencies, or configuration during initialization unless the user explicitly asks. Report the blocker instead.

## Preconditions you must verify yourself

1. Read **application source**, **docs**, and **tests** (test code is part of the specification).
2. **Dev environment** - install dependencies and prepare the environment the repo documents (for example `python -m venv .venv`, `pip install -e .`, `uv sync`, `npm ci`, or commands from `README` / CI). If the stack is not Python, follow that ecosystem's norms.
3. Locate how tests are run. Default for Python: `pytest` via `.venv` when present. Respect `pytest.ini`, `pyproject.toml`, or `tox` / `nox` if present.
4. Identify entrypoints: `README`, `docs/`, package layout, `main` modules, CLI.

## Workflow

1. **Optional git update** - if the directory is a git repository and the user has not asked to stay on the current commit, run `git pull` to refresh the local copy before analysis.
2. **Baseline** - record branch, commit, and dirty state before changing the environment.
3. **Scan** repository structure (layout, monorepo packages if any).
4. **Read** user-facing documentation and specs.
5. **Study test code** (naming, fixtures, markers, integration vs unit). This is the BDD-facing view of expected behavior.
6. **Set up dev environment** so tests can run (create venv, install deps, any documented bootstrap). Note blockers if setup cannot be completed.
7. **Run the test suite**, for example:

   ```bash
   .venv/bin/pytest -q
   ```

   Record the exact command, the result, and any failure that is unrelated to the current checkout. If tests fail, record where and why; do not fix unless the user asked.

8. **Summarize** in a concise report (see template). Use English for code-related terms if the codebase uses English; respond in the user's language for narrative.

## Report template

Use this structure (adapt if needed):

```markdown
## Project snapshot
- Purpose (one paragraph)
- Main packages and boundaries

## Baseline
- Branch, commit, dirty state
- Environment and dependency setup actually performed

## How to run tests
- Commands actually used

## Results
- Pass / fail / unavailable / blocked, with the exact evidence
- Pre-existing failures and environmental blockers

## Behavior from tests
- Scenarios covered by tests (bullets)
- Gaps (important paths with weak or missing tests)

## Risks and notes
- Flaky tests, secrets, external services

## Suggested next steps
- 1 to 3 follow-ups
```

## Constraints

- Comments in any new code: English only.
- Do not add secrets or keys.
- If the repository is not Python, still follow the same pattern: official install and test commands, then report.
- Keep the report short enough to read; put evidence next to the claim it supports.
