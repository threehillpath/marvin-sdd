---
name: configure-plan-plugin
description: Walk through configuring .claude/plan-workflow-config.md for a new project
argument-hint: (no arguments)
allowed-tools: Bash, Read, Write
model: sonnet
---

Interactively discover and write all values needed for `.claude/plan-workflow-config.md` in the current project directory. Most values are discovered automatically from the `gh` CLI; the user confirms choices along the way.

This skill is self-contained — it does not depend on `.claude/plan-workflow-config.md` since it is creating it.

## Steps

### 1. Detect or confirm the GitHub repo

```bash
git remote get-url origin
```

Parse the owner and repo from the URL (handles both HTTPS and SSH formats). Show the detected value and ask: "Is this the correct repo? (owner/repo)" If detection fails or the user corrects it, use their input.

### 2. Confirm authentication

```bash
gh auth status
```

If not authenticated, stop and instruct the user to run `gh auth login` before continuing.

### 3. List available projects and get the project ID

```bash
gh project list --owner <owner> --format json --limit 20
```

Display the projects (number and title). Ask the user which project to use. Then extract its ID:

```bash
gh project list --owner <owner> --format json --limit 20 \
  | jq -r '.projects[] | select(.number == <chosen-number>) | .id'
```

### 4. Discover status field and option IDs

```bash
gh project field-list <project-number> --owner <owner> --format json
```

Find the single-select field for board status (typically "Status"). If ambiguous, ask the user to confirm which field. Extract the field ID and all option IDs. Ask the user to map each option to: Backlog, Ready, In Progress, In Review, Done. Accept alternate column names (e.g., "To Do" → Backlog). Note any missing statuses.

### 5. Ask for test commands

Ask: "What command runs your backend tests from the repo root?" and "What command runs your frontend tests? (or 'none')"

### 6. Confirm before writing

Present all discovered values and ask: "Does this look correct? Type 'yes' to write the config or describe any corrections." Iterate until confirmed.

### 7. Write .claude/plan-workflow-config.md

```bash
mkdir -p .claude
```

Read `SUPPLEMENTS/TEMPLATES.md` for the file structure. Fill in all confirmed values and write the complete file to `.claude/plan-workflow-config.md`. If the file already exists, preserve the Board Management Commands section from the existing file (only the configuration table and test commands change).

### 8. Confirm

Report that `.claude/plan-workflow-config.md` has been written for `<owner/repo>`. Plan type labels (`plan:arch`, `plan:impl`, `plan:phase`) will be created automatically the first time each skill creates an issue.
