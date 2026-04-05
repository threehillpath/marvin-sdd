# plan-workflow

A Claude Code plugin providing a structured architecture-to-implementation workflow using GitHub issues and project boards.

## Overview

This plugin implements a full planning and development workflow:

```
/arch-plan   →   /impl-plan   →   /phase-split   →   /start-impl
                                                           ↓
                                         /implement-phase (repeats per phase)
                                                           ↓
                                                    /finish-impl
```

Board management is available at any point via `/move-issue`.

## Skills

| Skill | Description |
|---|---|
| `/configure-plan-plugin` | Set up `.claude/plan-workflow-config.md` for a new project (run first) |
| `/arch-plan <issue>` | Create an architectural plan from a GitHub issue |
| `/impl-plan <arch-issue>` | Create a technical implementation plan from an arch plan |
| `/phase-split <impl-issue>` | Break an impl plan into phase issues on the board |
| `/start-impl <impl-issue>` | Create the implementation branch and confirm phase readiness |
| `/implement-phase <phase-issue>` | Run the TDD loop autonomously in a worktree, open a PR |
| `/finish-phase <phase-issue>` | Commit, push, open a PR, and move phase to In Review |
| `/finish-impl <impl-issue>` | Open a PR from the impl branch to main, move to In Review |
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
- Write everything to `.claude/plan-workflow-config.md`

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

- [`gh`](https://cli.github.com/) authenticated with access to your repo and project
- A GitHub Project (classic projects not supported — must be a Projects v2 board)
- [`jq`](https://jqlang.github.io/jq/) for JSON parsing in board commands
- [`glow`](https://github.com/charmbracelet/glow) (optional) for rendered markdown previews
