# Config File Template

Use this structure when writing `.claude/plan-workflow-config.md`. Replace all placeholder values with the discovered configuration.

```markdown
# Skill Set Configuration

Run `/configure-plan-plugin` to populate this file automatically for a new project.

| Setting | Value |
|---|---|
| GitHub repo | `<owner/repo>` |
| Project number | `<N>` |
| Project ID | `<PVT_...>` |
| Status field ID | `<PVTSSF_...>` |
| "Backlog" option ID | `<option-id>` |
| "Ready" option ID | `<option-id>` |
| "In Progress" option ID | `<option-id>` |
| "In Review" option ID | `<option-id>` |
| "Done" option ID | `<option-id>` |

## Test Commands

Run from the **repository root**. Use the framework's native invocation pattern so commands start with the test runner binary and match any permission allowlists.

| Scope | Command (run from repo root) |
|---|---|
| Backend | `<command>` |
| Frontend | `<command or none>` |

## Board Management Commands

### Add an issue to the board and capture its item ID

Use `--jq '.id'` to extract the item ID directly from the response. Do NOT pipe the output
to a separate `jq` call — the response body contains issue content with control characters
that cause shell `jq` to fail with a parse error.

```bash
ITEM_ID=$(gh project item-add <project-number> --owner <owner> \
  --url https://github.com/<repo>/issues/<issue-number> \
  --format json --jq '.id')
```

Use `$ITEM_ID` immediately in the next command. Do not re-query `item-list` to find the ID —
newly added items may not appear in list results for a few seconds.

### Set board status

```bash
gh project item-edit --id <item-id> \
  --project-id <project-id> \
  --field-id <status-field-id> \
  --single-select-option-id <option-id>
```

### Check if issue is already on the board

Use `--jq` to filter server-side and avoid piping through shell `jq`:

```bash
ITEM_ID=$(gh project item-list <project-number> --owner <owner> \
  --format json --limit 100 \
  --jq '.items[] | select(.content.number == <issue-number>) | .id')
# If ITEM_ID is empty, the issue is not on the board — add it first, then set status
```
```
