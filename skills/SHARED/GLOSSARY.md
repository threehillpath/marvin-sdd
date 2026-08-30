# Plan Workflow Glossary

Canonical names, paths, and state transitions used by every skill in this plugin. Skills should reference this file rather than re-deriving names inline.

## Plan number

The plan number is the GitHub issue number being addressed, zero-padded to 5 digits. No scanning, no separate counter.

- Source issue `#42` → plan number `PLAN-00042`
- Source issue `#111` → plan number `PLAN-00111`

One arch plan, one impl plan, and any number of phase issues all share the same `XXXXX`.

### Edge case: multiple impl plans from one arch plan

Rare — used only when the work cleanly splits into independent tracks (e.g. backend and frontend developed in parallel). Suffix the impl plan with a letter:

- Arch plan: `[PLAN-00042-ARCH] Feature`
- Impl plan A: `[PLAN-00042-A] Feature — Backend`, trunk branch `feature/PLAN-00042/main-a`
- Impl plan B: `[PLAN-00042-B] Feature — Frontend`, trunk branch `feature/PLAN-00042/main-b`
- Phases under A: `[PLAN-00042-A-1]`, `[PLAN-00042-A-2]`, … → phase branches `feature/PLAN-00042/phase-a-1`, `feature/PLAN-00042/phase-a-2`
- Phases under B: `[PLAN-00042-B-1]`, `[PLAN-00042-B-2]`, … → phase branches `feature/PLAN-00042/phase-b-1`, `feature/PLAN-00042/phase-b-2`

Prefer phases within one impl plan over multiple impl plans whenever possible.

## Names

| Item | Format | Example |
|---|---|---|
| Plan number | `PLAN-XXXXX` | `PLAN-00042` |
| Arch plan issue title | `[PLAN-XXXXX-ARCH] <title>` | `[PLAN-00042-ARCH] Member Invitations` |
| Impl plan issue title | `[PLAN-XXXXX] <title>` | `[PLAN-00042] Member Invitations` |
| Phase issue title | `[PLAN-XXXXX-N] <title>` | `[PLAN-00042-3] Domain Model` |
| Trunk branch | `<type>/PLAN-XXXXX/main` | `feature/PLAN-00042/main` |
| Phase branch | `<type>/PLAN-XXXXX/phase-N` | `feature/PLAN-00042/phase-3` |
| Phase worktree path (relative to repo root) | `<worktree_base>/phase-XXXXX-N` | `.worktrees/phase-00042-3` |

`worktree_base` is the configured worktree root from `.claude/plan-workflow-config.yml` (default: `.worktrees`). `marvin names derive <phase-issue>` returns this path **relative to the repo root** (`worktree_path:` in its output), not absolute — use `marvin worktree resolve <worktree_path>` to turn it into an absolute path, anchored against the main repo root regardless of the calling process's CWD (safe to call from inside a linked worktree). Worktree paths stay flat (`.worktrees/phase-XXXXX-N`), independent of the branch hierarchy above.

`<type>` is `feature` or `bug`, derived from the source issue's `bug`/`enhancement` label (default `feature` when neither is present), resolved via `marvin names derive --type`.

The `PLAN-XXXXX` token is uppercase; the `<type>` segment, `main`, `phase-N`, and any addendum slug are lowercase with hyphens. The trunk branch is created from `main` by `/start-impl`. Phase branches are created from the trunk branch by `/implement-phase`. Phase PRs target the trunk branch; the trunk (implementation) PR targets `main`.

## Status state machine

Every plan-related issue moves through these states on the project board. The board is authoritative — every skill that changes an issue's state also updates the board.

```
Backlog → Ready → In Progress → In Review → Done
```

| Transition | Triggered by | Notes |
|---|---|---|
| (created) → Ready | `arch-plan`, `impl-plan`, `phase-split` | Newly created plan issues skip Backlog and go straight to Ready. Backlog is reserved for issues queued but not yet planned. |
| Ready → In Progress | `start-impl` (impl plan), `implement-phase` (phase) | Also reachable via `move-issue <n> in-progress` for manual moves. |
| In Progress → In Review | `finish-phase` (phase), `finish-impl` (impl plan) | The skill opens the PR and moves the issue in one step. |
| In Review → Done | `wrap-phase` (phase) | Triggered after the phase PR is merged. The impl plan moves to Done manually via `move-issue` after the impl PR is merged to main. |

`move-issue` can override any transition for manual recovery. Domain and status labels (e.g. `status:upcoming`) are independent of board state — the labels reflect issue intent at creation; the board reflects current workflow position.

## Concepts referenced by multiple skills

These terms are defined in the skill that owns them; this list points to the source of truth.

| Term | Defined in |
|---|---|
| TDD entry point | `impl-plan/SUPPLEMENTS/TDD.md` |
| Rendered-controls exemption | `impl-plan/SUPPLEMENTS/TDD.md` |
| Self-review checklist | `implement-phase/SUPPLEMENTS/LOOP.md` §4 |
| Notes section / corrections record | `implement-phase/SUPPLEMENTS/LOOP.md` §6 and `SHARED/PR_TEMPLATE.md` |
| Phase wrap-up classification | `wrap-phase/SUPPLEMENTS/CLASSIFY.md` |
| Phasing principles | `phase-split/SUPPLEMENTS/PHASING.md` |
| Code-review rubric | `SHARED/REVIEW_RUBRIC.md` |
| Review findings format | `SHARED/REVIEW_FINDING_FORMAT.md` |
| Plan red-team rubric | `SHARED/PLAN_RED_TEAM_RUBRIC.md` |
| Plan red-team findings format | `SHARED/PLAN_RED_TEAM_FORMAT.md` |
| Plan drift rubric | `SHARED/PLAN_DRIFT_RUBRIC.md` |
| Plan drift findings format | `SHARED/PLAN_DRIFT_FORMAT.md` |
| Board command patterns | `SHARED/CONFIG.md` (template) and `.claude/plan-workflow-config.yml` (per project) |
