---
name: move-issue
description: Move a GitHub issue to a specific column on the project board
argument-hint: <issue-number> <status>
allowed-tools: Bash, Read
model: haiku
---

Move a GitHub issue to the specified status column on the kanban board.

**Before starting**: Read `.claude/plan-workflow-config.md` for project configuration values (repo, project number, project ID, owner, field ID, and status option IDs). Read `../SHARED/GLOSSARY.md` for the canonical status state machine.

## Arguments

- `$0` — Issue number (e.g. `68`)
- `$1` — Target status (case-insensitive). Accepted values:
  - `backlog`
  - `ready`
  - `in progress` / `in-progress`
  - `in review` / `in-review`
  - `done`

## Steps

1. **Read `.claude/plan-workflow-config.md`** to get the project number, owner, project ID, status field ID, and status option IDs.

2. **Resolve the status name** in `$1` to the correct option ID. Match case-insensitively. If unrecognized, list valid options and stop.

3. **Add the issue to the board (idempotent) and capture its item ID.** `item-add` returns the existing item ID if the issue is already on the board, or adds it and returns the new ID. Always use this in preference to `item-list` — newly added items can lag in `item-list` results by several seconds, and the JSON response includes issue body content with control characters that break shell `jq`.

   ```bash
   ITEM_ID=$(gh project item-add <project-number> --owner <owner> \
     --url https://github.com/<repo>/issues/$0 \
     --format json --jq '.id')
   ```

4. **Update the status**:
   ```bash
   gh project item-edit \
     --id "$ITEM_ID" \
     --project-id <project-id> \
     --field-id <status-field-id> \
     --single-select-option-id <option-id>
   ```

5. **Sync the GitHub issue state**:
   - If target is `done`: close the issue
     ```bash
     gh issue close $0 --repo <repo> --reason completed
     ```
   - Otherwise: reopen if currently closed (check state first to avoid a no-op error)
     ```bash
     gh issue reopen $0 --repo <repo>
     ```

6. **Confirm**: report the issue number, title, new board status, and issue state (open/closed).
