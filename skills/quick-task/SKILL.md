---
name: quick-task
description: Create and drive a single-cycle Task from a source issue through implementation and review to a merged PR, without the arch-plan/impl-plan/phase-split hierarchy
argument-hint: <source-issue-number>
allowed-tools: Bash, Read, Glob, Grep, Agent
model: sonnet
---

Take a bug report or small feature request straight from a source issue to an opened, reviewed PR — one Task issue, one branch, one PR to `main` — skipping the arch-plan → impl-plan → phase-split hierarchy entirely. One invocation drives a Task through both of its stages: **Mode A** creates the Task and opens its PR; **Mode B**, run again after merge, closes it out.

**Before starting**: Read `.claude/plan-workflow-config.yml` for project configuration (repo, owner) and test commands. Read `../SHARED/GLOSSARY.md` for naming and status conventions.

## Arguments

- `$0` — Source issue number (the bug/feature-request issue this Task addresses — not a plan or phase issue)

## Mode detection

```bash
marvin issue list --label plan:task --title-prefix "[TASK-XXXXX]" --state all --limit 500
```

Where `XXXXX` is `$0` zero-padded to 5 digits. Both `--label` and `--limit` are load-bearing here: `--title-prefix` filters client-side after the fetch, and `gh issue list` (which `marvin issue list` wraps) defaults to a 100-issue limit — in a repo with many closed Tasks, the real target could silently fall outside that window before the title-prefix filter ever runs.

If the filtered result is non-empty, a Task issue already exists for `$0` — go to **Mode B**. If empty, go to **Mode A**.

If `marvin` exits with code 2 at any step below, surface to the user: "Configuration missing — run `/configure-plan-plugin` first."

## Mode A — no existing Task issue for `$0`

### A1. Fetch the source issue

```bash
gh issue view $0 --repo <repo> --json number,title,body,labels
```

Resolve `<type>` from `$0`'s labels: `bug` present → `bug`; otherwise (including when neither `bug` nor `enhancement` is present) → `feature` — same convention every other skill in this plugin uses.

### A2. Derive Task naming

```bash
marvin names derive $0 --task --type <type> --json
```

Capture `task_branch`, `worktree_path` (repo-root-relative, not yet absolute), and `title_prefix.task`.

### A3. Draft the Task issue

```bash
marvin template render quick-task --skeleton
```

Use the rendered skeleton (six required sections) as the structural frame:

- **Problem Statement** and **Scope** — filled from `$0`'s body.
- **Technical Analysis** — identify the paths of source files likely relevant to `$0` (do not read them yourself), then spawn an **Explore** sub-agent to digest them, following the same pattern `impl-plan/SKILL.md` uses for its own code digest, at smaller scope:

  > Goal: produce a digest that a planner will use to write a Task's Technical Analysis section. Do not draft the Task — just describe what currently exists.
  >
  > For each of the following files, return:
  > - **Path**
  > - **Purpose** (1 sentence)
  > - **Key types / function signatures / exports** (signatures only, not bodies)
  > - **Notable behavior** the planner needs to know — invariants, side effects, hidden coupling, error-handling patterns, tests that exist
  > - **Likely change surface** for the work described below — which functions or types would need to be touched, added, or replaced
  >
  > Files:
  > <bullet list of paths>
  >
  > Work to address (from source issue #$0):
  > <paste the source issue body, or the relevant excerpt>
  >
  > Keep the total response under ~1500 words. Skip files that turn out to be trivial. If you find a file that should also be read but wasn't on the list, mention it by path with a one-sentence reason — don't read it.

  Use the digest to draft Technical Analysis; you do not need to read the underlying files yourself unless the digest flags something needing deeper inspection.
- **TDD Entry Point** and **Implementation Notes** — drafted from the digest and `$0`'s content.

### A4. Present for review

Read `../SHARED/LABELS.md`. Present the draft to the user: title `"<title_prefix.task> <Title>"`, proposed labels `plan:task, status:upcoming, <domain-labels>, <type-label from $0>` (the `bug`/`enhancement` label carried forward from `$0`, per `LABELS.md`'s "Source issue labels" rule). Iterate on content and labels until the user approves — same pattern `arch-plan`/`impl-plan` use for their own "present for review" steps.

### A5. Create the issue

```bash
marvin label ensure --builtins
```

For any domain label not covered by `--builtins`, ensure it individually:

```bash
marvin label ensure "<name>" --description "<desc>" --color "<hex>"
```

`quick-task` has no `Write` tool. Write the approved, rendered body to a scratch file via a `Bash` heredoc:

```bash
cat > /tmp/quick-task-body.md <<'EOF'
<approved body>
EOF
```

Then create the issue, capturing the returned number and URL:

```bash
marvin issue create --title "<title_prefix.task> <Title>" --body-file /tmp/quick-task-body.md --label "plan:task,status:upcoming,<domain-labels>,<type-label>"
```

Comment-link the source issue — informational only, not a GitHub-native sub-issue link, since Tasks are deliberately outside the arch/impl/phase hierarchy `marvin issue tree` walks:

```bash
gh issue comment $0 --repo <repo> --body "Task created: #<new-issue> ([TASK-XXXXX])"
```

### A6. Board

```bash
marvin board move <new-issue> ready
marvin board move <new-issue> in-progress
```

### A7. Set up the worktree

```bash
marvin worktree add <worktree_path> <task_branch> main
```

### A8. Spawn the implement-loop sub-agent

Resolve the worktree path to absolute — `worktree_path` from A2 is repo-root-relative, not absolute:

```bash
marvin worktree resolve <worktree_path>
```

Read `SUPPLEMENTS/LOOP.md` for the TDD loop instructions.

Spawn a **general-purpose** agent **without** `isolation: "worktree"` (the worktree was created in A7) and model **sonnet**. Assemble the task prompt by **referencing** the inputs the sub-agent should fetch — do not paste full file contents into the prompt — mirroring `implement-phase/SKILL.md`'s equivalent spawn step in shape:

1. Task issue number (and `gh issue view` command for it) — the sub-agent fetches title, Problem Statement, Scope, Technical Analysis, TDD Entry Point, and Success Criteria itself.
2. Source issue number `$0` (and `gh issue view` command for it).
3. **Paths only** for the relevant source files identified in A3 — the sub-agent reads them inside the worktree.
4. Task branch: `<task_branch>`.
5. **Absolute** worktree path (resolved above — never the repo-relative form from A2).
6. Repo: `<repo>`.
7. Task title.
8. Test commands from `.claude/plan-workflow-config.yml`.
9. Full instructions from `SUPPLEMENTS/LOOP.md` (paste verbatim — it is the sub-agent's primary procedural guide).

The sub-agent must not pause for user confirmation except on unresolvable failure or ambiguity. Capture its returned PR number, PR URL, and summary.

### A9. Spawn the review sub-agent

Immediately after the implement-loop sub-agent returns, spawn a second, independent sub-agent to review the PR it opened:

- `subagent_type: "general-purpose"`, `model: "opus"`, no worktree isolation (review is read-only, fresh context).
- Prompt: read `../SHARED/REVIEW_FINDING_FORMAT.md` first (output schema), then `../SHARED/REVIEW_RUBRIC.md` (rubric) — apply the rubric to the PR diff (`gh pr diff <pr-number> --repo <repo>`). For `spec-drift`, anchor on the Task issue's Success Criteria section read together with source issue `$0`'s Problem Statement and Scope, per the rubric's Task clause. The sub-agent fetches the Task issue (`gh issue view <task-issue> --repo <repo> --json title,body`) and the PR (`gh pr view <pr-number> --repo <repo> --json title,body,commits,files`) itself. Return only the JSON object specified in `REVIEW_FINDING_FORMAT.md` — no surrounding prose, no code fence.

Parse and validate the response the same way `review-phase` does: shape matches `REVIEW_FINDING_FORMAT.md` (`summary`, `verdict`, `blocking`, `nits`); `verdict` is consistent with the arrays (`approve` requires both empty, `request-changes` requires `blocking` non-empty, `comment` requires `blocking` empty and `nits` non-empty); every finding has all required fields and a recognized `category`. If validation fails, re-spawn once with feedback; after two failures, stop and show the user the raw response.

Render the findings into a **single PR comment**: a summary paragraph, followed by one block per finding (`blocking` first, then `nits`, in order):

```
**[<id>] <category> — <severity>**

<summary>

<details>

**Suggested fix**: <suggested_fix>

_Evidence_: <evidence>
```

This is deliberately lighter than `review-phase`'s mechanism (per-finding inline-anchored review comments plus a separate PR review submission) — no diff-position math is needed for a single summary comment. Write the rendered body to a scratch file via a `Bash` heredoc and post it **automatically, with no user-approval gate** — this is one autonomous invocation that ends with the PR opened and reviewed, unlike `review-phase`, which a human invokes separately and specifically to decide whether to post:

```bash
gh pr comment <pr-number> --repo <repo> --body-file <path>
```

### A10. Stop

Report to the user:
- PR URL and number
- Review verdict, blocking count, nit count

Message: "Task PR opened and reviewed — see findings above. Address any blocking findings, then merge, then re-run `/quick-task $0` to finish wrap-up."

## Mode B — a Task issue already exists for `$0`

### B11. Re-derive naming

Independently of Mode A's state — Mode B is a separate invocation with no memory of a prior run:

```bash
gh issue view $0 --repo <repo> --json labels
marvin names derive $0 --task --type <type> --json
```

Resolve `<type>` from `$0`'s labels exactly as in A1. Capture `title_prefix.task`, `worktree_path`, `task_branch`.

### B12. Check the Task PR's state

```bash
marvin pr find "<title_prefix.task>" --state any
```

`pr.ParseState` accepts `open`, `merged`, `any` — the literal flag value is `any`, never `all`.

- **`found: false`** — a prior invocation created the Task issue but crashed before the PR was opened. **Resume Mode A from the first incomplete step.** Determine the resume point by checking the Task issue's board status and `git worktree list` for `<worktree_path>` / `<task_branch>`:
  - Worktree already registered → resume at **A8** (spawn the implement-loop sub-agent).
  - Otherwise → resume at **A6** (board move).

  `marvin board move` and `marvin worktree add` are both idempotent, so an imprecise resume point is safe — re-running an already-completed step is a no-op, not an error, never a duplicate.
- **Found and not merged**: report the PR's URL and state to the user, stop.
- **Found and merged**: proceed to **B13**.

### B13. Cleanup

```bash
marvin board move <task-issue> done
gh issue close <task-issue> --repo <repo> --reason completed
marvin worktree remove <worktree_path>
marvin worktree prune
```

`marvin worktree remove` resolves `<worktree_path>` against the repo root internally regardless of the invoking CWD — no need to resolve it to absolute manually first, unlike A8's sub-agent spawn, which hands the path to a sub-agent that isn't necessarily running from the repo root.

### Confirm

Report the outcome of whichever step this invocation stopped at (A10, B12's status report, or B13's cleanup): PR URL, verdict, board/issue state, worktree status.
