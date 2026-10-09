# CLAUDE.md

## What this repo is

A Claude Code plugin defining a structured architecture-to-implementation workflow on top of GitHub issues and Projects v2 boards. Includes `marvin`, a compiled Go CLI that encapsulates the deterministic shell operations (board, issues, templates, labels, PRs, worktrees, config, findings cache) so skills call one binary rather than re-synthesizing `gh`/`jq`/`git` invocations.

## Where things live

- `skills/` — one directory per skill (`SKILL.md` + `SUPPLEMENTS/`); `skills/SHARED/` holds the files referenced by multiple skills (names, statuses, labels, PR templates, rubrics, config template). See `skills/CLAUDE.md`.
- `agents/` — plugin sub-agents with fixed model + effort. See `agents/CLAUDE.md`.
- `tool/` — Go module for `marvin`. See `tool/CLAUDE.md`.
- `hooks/hooks.json` — `SessionStart` hook that runs `tool/build.sh` into `${CLAUDE_PLUGIN_ROOT}/bin/marvin`, degrading quietly (stderr diagnostic, exit 0) if `go` or `tool/` is missing.
- `docs/stories/` — durable per-story records written by `finish-impl` and `wrap-phase`; layout in `skills/SHARED/GLOSSARY.md`.
- `.github/ISSUE_TEMPLATE/` — issue forms for human-filed source issues.

The plugin is installed only from the GitHub marketplace (`plan-workflow@plan-workflow-marketplace`). Per-project config and plan-template precedence are documented in `skills/SHARED/CONFIG.md`. Install requirements are in `README.md`.

## Workflow

```
arch-plan → impl-plan → red-team-plan → phase-split → start-impl →
    [ implement-phase → (plan-drift) → review-phase → merge → wrap-phase ]  per phase
    → finish-impl → review-impl → merge
```

`move-issue`, `finish-phase`, `plan-drift`, `unslop` are auxiliaries; `quick-task` is a standalone single-cycle pipeline for a bug or small task that bypasses the hierarchy.

## Workflow design rules

- **Plans specify *what*, not *how*** — implementation is for Claude to discover from the consuming project's code context.
- **TDD is mandatory** across the workflow; the exemption is narrow (rendered controls only). Full rule and litmus test in `skills/impl-plan/SUPPLEMENTS/TDD.md`.
- **Phases are independently mergeable** — each phase has its own branch and PR to the impl branch; the impl branch PRs to main.
- **Board state is authoritative** — every skill that changes intent also moves the issue. State machine in `skills/SHARED/GLOSSARY.md`.
- **Errors must not fail silently** — a command that cannot complete its intended effect must return a non-zero exit code or emit a stderr diagnostic. The only exception is a genuine no-op (the target is already in the desired state), and even that must be observably distinct from a caller resolving the wrong target (e.g. a stale relative path) — never silently identical to one. Exceptions to this rule must be explicit and deliberate, not a byproduct of convenient error handling.

## marvin contract

Exit codes: `0` success, `1` operational error, `2` config missing or malformed, `3` draft or body not conforming to its plan schema. `stdout` = data, `stderr` = diagnostics (the one exception: `marvin template validate`, whose findings are its data). Details in `tool/CLAUDE.md`.
