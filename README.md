# plan-workflow

A Claude Code plugin providing a structured architecture-to-implementation workflow using GitHub issues and project boards.

## Overview

This plugin implements a full planning and development workflow:

```
/arch-plan → /impl-plan → /red-team-plan → /phase-split → /start-impl →
    [ /implement-phase → (/plan-drift) → /review-phase → merge → /wrap-phase ]  per phase
    → /finish-impl → /review-impl → merge
```

`/move-issue`, `/finish-phase`, and `/plan-drift` are auxiliaries usable at any point — see each skill's `SKILL.md` for behavior.

## Skills

| Skill | Description |
|---|---|
| `/configure-plan-plugin` | Set up `.claude/plan-workflow-config.yml` for a new project (run first) |
| `/arch-plan <issue>` | Create an architectural plan from a GitHub issue |
| `/impl-plan <arch-issue>` | Create a technical implementation plan from an arch plan |
| `/red-team-plan <impl-issue>` | Critique an impl plan with a fresh-context opus sub-agent before phase-split |
| `/phase-split <impl-issue>` | Break an implementation plan into phases and create GitHub issues for each |
| `/start-impl <impl-issue>` | Summarize an implementation plan, confirm phases, and create the implementation branch |
| `/implement-phase <phase-issue>` | Run the TDD loop autonomously in a worktree, open a PR to the implementation branch |
| `/plan-drift <phase-issue>` | Audit a phase branch for coverage and scope drift against its spec |
| `/review-phase <phase-issue>` | Code-review a phase PR with a fresh-context opus sub-agent and post the review to GitHub |
| `/finish-phase <phase-issue>` | Commit, push, open a PR to the implementation branch, and move phase to In Review |
| `/wrap-phase <phase-issue> <impl-issue>` | Capture decisions from a merged phase PR, close the phase issue, move to Done, clean up the worktree |
| `/finish-impl <impl-issue>` | Open a PR from the implementation branch to main and move the impl plan to In Review |
| `/review-impl <impl-issue>` | Comprehensive code-review of the impl PR (trunk branch → main) after `/finish-impl` opens it |
| `/move-issue <issue> <status>` | Move any issue to a board column |

## Setup

### 1. Install the plugin

```bash
# From a local directory
claude --plugin-dir /path/to/plan-workflow
```

Or add to your project's `.claude/settings.json` once published to a marketplace.

### 2. Configure for your project

Run `/configure-plan-plugin` in your project. It will:
- Detect your GitHub repo from the git remote
- List your GitHub Projects and let you pick one
- Discover the project's status field and column IDs
- Ask for your test commands
- Write everything to `.claude/plan-workflow-config.yml`

### 3. Create GitHub labels

The first time each skill creates a plan issue, it will create any missing labels automatically. You can also create them manually:

```bash
gh label create "plan:arch" --repo <owner/repo> --description "Architecture plan" --color "0075ca"
gh label create "plan:impl" --repo <owner/repo> --description "Implementation plan" --color "0075ca"
gh label create "plan:phase" --repo <owner/repo> --description "Phase / implementation unit" --color "0075ca"
gh label create "status:upcoming" --repo <owner/repo> --description "Upcoming work" --color "e4e669"
```

## Plan numbering

Each plan traces back to a source GitHub issue. The source issue number becomes the PLAN number:

- Source issue #42 → `PLAN-00042`
- Arch plan: `[PLAN-00042-ARCH] Title`
- Impl plan: `[PLAN-00042] Title`
- Phases: `[PLAN-00042-1] Title`, `[PLAN-00042-2] Title`, ...

## Requirements

**Install-time** (needed when running `deploy.sh`):
- [Go SDK](https://go.dev/dl/) (`go` on PATH) — `deploy.sh` compiles the bundled `marvin` CLI and will fail with a clear error message if `go` is absent

**Runtime** (needed when using skills):
- [`gh`](https://cli.github.com/) authenticated with access to your repo and project
- A GitHub Project (classic projects not supported — must be a Projects v2 board)
- [`jq`](https://jqlang.github.io/jq/) (optional) for post-processing `marvin --json` output in custom skill prose (marvin's list/object commands default to plain text)
- [`glow`](https://github.com/charmbracelet/glow) (optional) for rendered markdown previews
