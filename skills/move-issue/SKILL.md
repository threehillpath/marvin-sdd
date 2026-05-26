---
name: move-issue
description: Move a GitHub issue to a specific column on the project board
argument-hint: <issue-number> <status>
allowed-tools: Bash, Read
model: haiku
---

Move a GitHub issue to the specified status column on the kanban board.

Read `../SHARED/GLOSSARY.md` for the canonical status state machine.

## Arguments

- `$0` — Issue number (e.g. `68`)
- `$1` — Target status (case-insensitive). Accepted values:
  - `backlog`
  - `ready`
  - `in progress` / `in-progress`
  - `in review` / `in-review`
  - `done`

## Steps

1. **Resolve the status name** in `$1`. Match case-insensitively. If unrecognized, list valid options and stop.

2. **Move the issue on the board**:

   ```bash
   marvin board move $0 "$1"
   ```

   If `marvin` exits with code 2, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

3. **Confirm**: report the issue number, new board status, and issue state (open/closed).
