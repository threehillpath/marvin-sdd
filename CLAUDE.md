# CLAUDE.md

## What this repo is

A Claude Code plugin defining a structured architecture-to-implementation workflow on top of GitHub issues and Projects v2 boards. Includes `marvin`, a compiled Go CLI that encapsulates the deterministic shell operations (board reads and moves, issue listing, label management, PR lookup, worktree lifecycle, config access, findings cache) so skills can call a single binary rather than re-synthesizing `gh`/`jq`/`git` invocations.

## Repository structure

```
.claude-plugin/plugin.json     ← Plugin metadata (name, version, description)
hooks/hooks.json               ← SessionStart hook: builds marvin if missing/stale (marketplace installs)
.github/
  ISSUE_TEMPLATE/              ← Native GitHub issue forms for human-filed source issues
    feature.yml                ← Feature request form (applies `enhancement` label)
    bug.yml                    ← Bug report form (applies `bug` label)
    config.yml                 ← Issue chooser config (blank_issues_enabled, contact links)
tool/                          ← Go module for the marvin CLI
  build.sh                     ← Shared build step: compiles cmd/marvin to a given output path, skipping if up to date
  cmd/marvin/main.go           ← Entry point; compiled to bin/marvin at install time
  internal/
    board/                     ← GitHub Projects v2 board operations (add, move, list, status)
    issue/                     ← GitHub issue reads (list with label/prefix/state filters)
    cli/                       ← Cobra command handlers
    clierr/                    ← Exit-code constants (0 / 1 / 2)
    config/                    ← YAML config loader, legacy markdown fallback, CWD-walk discovery
    exec/                      ← Runner interface (injectable for tests)
    exectest/                  ← Fake runner for unit tests (no network / no git state)
    findings/                  ← Findings cache read/write
    gh/                        ← gh JSON client wrapper
    label/                     ← Label ensure-exists
    names/                     ← Plan name derivation (PLAN-XXXXX, branch, worktree path, prefix)
    parse/                     ← Identifier parsing (issue titles → plan numbers)
    pr/                        ← PR discovery and target resolution
    template/                  ← Plan-template render from YAML schemas
    worktree/                  ← Worktree lifecycle (create, remove, prune, repo-root path resolution)
skills/
  SHARED/                      ← Files referenced by multiple skills
    CONFIG.md                  ← Template for per-project config
    GLOSSARY.md                ← Names, paths, status state machine
    LABELS.md                  ← Label rules
    PR_TEMPLATE.md             ← PR body templates
    RENDERING.md               ← Markdown output guidance
    REVIEW_RUBRIC.md           ← Code-review rubric for review-phase / review-impl / quick-task
    REVIEW_FINDING_FORMAT.md   ← Findings JSON schema (review output contract)
    PLAN_RED_TEAM_RUBRIC.md    ← Plan-critique rubric for red-team-plan
    PLAN_RED_TEAM_FORMAT.md    ← Findings JSON schema (plan-critique output contract)
    PLAN_DRIFT_RUBRIC.md       ← Coverage + containment rubric for plan-drift
    PLAN_DRIFT_FORMAT.md       ← Findings JSON schema (drift-audit output contract)
    templates/                 ← Internal YAML schemas per bot-created plan type
      arch-plan.yml            ← Schema for arch plan issues (sections, validation)
      impl-plan.yml            ← Schema for implementation plan issues
      impl-phase.yml           ← Schema for phase issues
      quick-task.yml           ← Schema for single-cycle Task issues
  <skill-name>/
    SKILL.md                   ← Authoritative skill prompt
    SUPPLEMENTS/               ← Templates and deeper guidance
```

## The marvin tool

`marvin` is compiled from `tool/` by `tool/build.sh`, which writes the binary to a caller-supplied output path and skips the build when that binary is already newer than every file under `tool/`. Skills call it for all deterministic operations — board moves, label management, config access, name derivation, PR lookup, worktree lifecycle, findings cache — so that none of that logic needs to be re-synthesized from shell in skill prose.

Two callers invoke `tool/build.sh`, covering the two install paths:
- **`deploy.sh`** (local-directory install) — hard-fails first if `go` is absent, then builds into `${PLUGIN_DIR}/bin/marvin`.
- **`hooks/hooks.json`** (marketplace install) — a `SessionStart` hook that builds into `${CLAUDE_PLUGIN_ROOT}/bin/marvin` on every session start, degrading quietly (stderr diagnostic, exit 0) if `go` is missing or `tool/` isn't present in that install, rather than blocking the session.

Subcommand groups: `config`, `names`, `parse`, `template`, `board`, `issue`, `label`, `pr`, `findings`, `worktree`, `version`.

Exit-code contract: `0` = success, `1` = operational error, `2` = config missing or malformed. Output contract: `stdout` = data, `stderr` = diagnostics.

## Skills (in workflow order)

```
arch-plan → impl-plan → red-team-plan → phase-split → start-impl →
    [ implement-phase → (plan-drift) → review-phase → merge → wrap-phase ]  per phase
    → finish-impl → review-impl → merge
```

`move-issue`, `finish-phase`, and `plan-drift` are auxiliaries usable at any point — `plan-drift` is most valuable mid-phase or before opening a PR, but can run any time. See each skill's `SKILL.md` for behavior.

`quick-task` is a standalone single-cycle pipeline for a bug or small task filed as a source issue: it bypasses the arch-plan → impl-plan → phase-split hierarchy entirely, driving a Task issue from filed requirement to merged PR (and all of that issue's own board transitions) in one skill invocation, with the same TDD and review rigor as the phased pipeline above.

### Spec-checking subagents

Four skills spawn fresh-context sub-agents that apply a shared rubric and return structured findings JSON:

- **`red-team-plan`** — opus sub-agent critiques the impl plan against `skills/SHARED/PLAN_RED_TEAM_RUBRIC.md`, returns findings as `skills/SHARED/PLAN_RED_TEAM_FORMAT.md`. Catches hidden assumptions, missing dependencies, weak TDD entry points, and unfalsifiable success criteria *before* phase-split, where errors compound.
- **`plan-drift`** — sonnet sub-agent audits a phase branch against its spec using `skills/SHARED/PLAN_DRIFT_RUBRIC.md`, returns findings as `skills/SHARED/PLAN_DRIFT_FORMAT.md`. Tracks two things: per-criterion coverage and out-of-scope/interface-divergence containment. Complements but does not replace `review-phase`.
- **`review-phase` / `review-impl`** — opus sub-agent (extended thinking) applies `skills/SHARED/REVIEW_RUBRIC.md` and returns findings as `skills/SHARED/REVIEW_FINDING_FORMAT.md`.
- **`quick-task`** — opus sub-agent review step, reusing the same `skills/SHARED/REVIEW_RUBRIC.md` (extended with a Task-specific spec-drift clause) and `skills/SHARED/REVIEW_FINDING_FORMAT.md` as `review-phase`/`review-impl`.

Each findings JSON is the stable contract a future auto-fix loop will consume — these skills produce it, future skills will act on it.

## Per-project config

Skills read `.claude/plan-workflow-config.yml` (preferred) or `.claude/plan-workflow-config.md` (legacy) in the **consuming project**, not this repo. The file is generated by `/configure-plan-plugin` and holds GitHub repo, project IDs, status option IDs, and test commands. Template at `skills/SHARED/CONFIG.md`.

## Skill authoring conventions

- Each `SKILL.md` declares its model (opus for planning, sonnet for implementation, haiku for board ops) and allowed tools in frontmatter.
- Skills reference SHARED files rather than re-defining names, statuses, PR templates, or label rules.
- Heavy code-reading is delegated to subagents (Explore for code digestion, general-purpose for autonomous work) so the orchestrating skill's context stays small.

## Workflow design rules

- **Plans specify *what*, not *how*** — implementation is for Claude to discover from the consuming project's code context.
- **TDD is mandatory** across the workflow; the exemption is narrow (rendered controls only). Full rule and litmus test in `skills/impl-plan/SUPPLEMENTS/TDD.md`.
- **Phases are independently mergeable** — each phase has its own branch and PR to the impl branch; the impl branch PRs to main.
- **Board state is authoritative** — every skill that changes intent also moves the issue. State machine in `skills/SHARED/GLOSSARY.md`.
- **Errors must not fail silently** — a command that cannot complete its intended effect must return a non-zero exit code or emit a stderr diagnostic. The only exception is a genuine no-op (the target is already in the desired state), and even that must be observably distinct from a caller resolving the wrong target (e.g. a stale relative path) — never silently identical to one. Exceptions to this rule must be explicit and deliberate, not a byproduct of convenient error handling.

## Requirements (for consuming projects)

**Install-time** (needed when running `deploy.sh`):
- [Go SDK](https://go.dev/dl/) (`go` on PATH) — `deploy.sh` compiles `marvin` during install and hard-fails with a clear message if `go` is absent

**Runtime** (needed when using skills):
- `gh` CLI authenticated with repo and project access
- git ≥ 2.31 (needed for `git rev-parse --path-format=absolute`, used by worktree path resolution)
- GitHub Projects v2 board (classic projects not supported)
- `jq` (optional) for post-processing `marvin --json` output in custom skill prose (marvin's list/object commands default to plain text)
- `glow` (optional) for rendered markdown previews in terminal
