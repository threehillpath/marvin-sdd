---
name: move-issue
description: Move a GitHub issue to a specific column on the project board
argument-hint: <issue-number> <status>
allowed-tools: Bash, Read
model: haiku
---

Move a GitHub issue to the specified status column on the kanban board.

**Before starting**: Read `../SHARED/CONFIG.md` for project configuration values (repo, project number, project ID, owner, field ID, and status option IDs).

## Arguments

- `$0` — Issue number (e.g. `68`)
- `$1` — Target status (case-insensitive). Accepted values:
  - `backlog`
  - `ready`
  - `in progress` / `in-progress`
  - `in review` / `in-review`
  - `done`

## Steps

1. **Read `../SHARED/CONFIG.md`** to get the project number, owner, project ID, status field ID, and status option IDs.

2. **Resolve the status name** in `$1` to the correct option ID. Match case-insensitively. If unrecognized, list valid options and stop.

3. **Find the project item ID** for issue `$0`:
   ```bash
   gh project item-list <project-number> --owner <owner> --format json --limit 100
   ```
   Parse the JSON and find the item where `content.number == $0`. Extract its `id` field.

4. **If the issue is not on the board**, add it first:
   ```bash
   gh project item-add <project-number> --owner <owner> \
     --url https://github.com/<repo>/issues/$0 --format json
   ```
   Extract the `id` from the response.

5. **Update the status**:
   ```bash
   gh project item-edit \
     --id <item-id> \
     --project-id <project-id> \
     --field-id <status-field-id> \
     --single-select-option-id <option-id>
   ```

6. **Sync the GitHub issue state**:
   - If target is `done`: close the issue
     ```bash
     gh issue close $0 --repo <repo> --reason completed
     ```
   - Otherwise: reopen if currently closed (check state first to avoid a no-op error)
     ```bash
     gh issue reopen $0 --repo <repo>
     ```

7. **Confirm**: report the issue number, title, new board status, and issue state (open/closed).
