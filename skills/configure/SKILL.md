---
name: configure
description: Walk through configuring SHARED/CONFIG.md for a new project
argument-hint: (no arguments)
allowed-tools: Bash, Read, Write
model: sonnet
---

Interactively discover and write all values needed for `../SHARED/CONFIG.md`. Most values are discovered automatically from the `gh` CLI; the user confirms choices along the way.

This skill is self-contained — it does not depend on `../SHARED/CONFIG.md` since it is creating it.

## Steps

### 1. Detect or confirm the GitHub repo

Try to detect from the git remote:
```bash
git remote get-url origin
```

Parse the owner and repo from the URL (handles both HTTPS and SSH formats). Show the detected value and ask: "Is this the correct repo? (owner/repo)"

If detection fails or the user corrects it, use their input.

### 2. Confirm authentication

```bash
gh auth status
```

If not authenticated, stop and instruct the user to run `gh auth login` before continuing.

### 3. List available projects

```bash
gh project list --owner <owner> --format json --limit 20
```

Display the projects (number and title). Ask the user which project to use for this skill set.

### 4. Get the project ID

```bash
gh project list --owner <owner> --format json --limit 20 \
  | jq -r '.projects[] | select(.number == <chosen-number>) | .id'
```

### 5. Discover status field and option IDs

```bash
gh project field-list <project-number> --owner <owner> --format json
```

Find the single-select field that represents board status (typically named "Status"). Show the field names and ask the user to confirm which is the status field if there is ambiguity.

From the confirmed field, extract:
- Field ID
- All option names and their IDs

Display the discovered options and ask the user to confirm which option name maps to each of: Backlog, Ready, In Progress, In Review, Done.

If the board uses different column names (e.g., "To Do" instead of "Backlog"), accept the user's mapping.

If any expected status is missing from the board, note it — that status will not be usable until the column is added on the board.

### 6. Ask for test commands

Ask the user:
- "What command runs your backend tests from the repo root?"
- "What command runs your frontend tests from the repo root? (or 'none')"

These will be stored in CONFIG.md and used by the `implement-phase` skill during the TDD loop.

### 7. Show discovered configuration

Present all values for confirmation before writing:

```
Repo:              owner/repo
Project number:    N
Project ID:        PVT_...
Status field ID:   PVTSSF_...
Backlog:           <option-id>
Ready:             <option-id>
In Progress:       <option-id>
In Review:         <option-id>
Done:              <option-id>

Test commands:
  Backend:   <command>
  Frontend:  <command or none>
```

Ask: "Does this look correct? Type 'yes' to write CONFIG.md or describe any corrections."

Iterate until confirmed.

### 8. Write CONFIG.md

Overwrite `../SHARED/CONFIG.md` with the confirmed values, preserving the Board Management Commands section exactly as-is (only the configuration table and test commands change).

Read the current file first to extract the Board Management Commands section, then write the complete updated file.

### 9. Confirm

Report that `../SHARED/CONFIG.md` has been updated. The skill set is now configured for `<owner/repo>`. Plan type labels (`plan:arch`, `plan:impl`, `plan:phase`) will be created automatically the first time each skill creates an issue.
