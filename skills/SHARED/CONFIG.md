# Skill Set Configuration

This file is a template. The active config for each project lives at `.claude/plan-workflow-config.md` in the project directory. Run `/configure-plan-plugin` to generate it.

| Setting | Value |
|---|---|
| GitHub repo | `` |
| Project number | `` |
| Project ID | `` |
| Status field ID | `` |
| "Backlog" option ID | `` |
| "Ready" option ID | `` |
| "In Progress" option ID | `` |
| "In Review" option ID | `` |
| "Done" option ID | `` |

## Plan Template Resolution

Skills that render plan issue bodies resolve the template for each plan type using this order:

1. **Project override** (wins if present): `.claude/plan-workflow-templates/{type}.yml` in the consuming project — sibling to `.claude/plan-workflow-config.md`.
2. **Plugin default** (always present): `skills/SHARED/templates/{type}.yml` in the plugin.

Where `{type}` is `arch`, `impl`, or `phase`. A skill checks for the project override first; if the file is absent, it reads the plugin default. The plugin default is always present, so rendering never fails for lack of a template.

This phase ships no override files — only the documented contract and the three plugin defaults.

## Test Commands

Run from the **repository root**. Use the framework's native invocation pattern so commands start with the test runner binary and match any permission allowlists.

| Scope | Command (run from repo root) |
|---|---|
| Backend | *(configure for your project)* |
| Frontend | *(configure for your project)* |

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
