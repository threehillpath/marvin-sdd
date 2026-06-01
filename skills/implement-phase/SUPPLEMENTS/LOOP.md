# TDD Implementation Loop — Sub-agent Instructions

This document is assembled into the sub-agent prompt by `implement-phase`. The sub-agent operates autonomously in a pre-created git worktree.

## 1. Setup

The worktree has been created by the orchestrator and is already on `<phase-branch>`. All work must happen inside the worktree directory:

```
<worktree-path>
```

Confirm the branch before making any changes:

```bash
git -C <worktree-path> branch --show-current  # must print: <phase-branch>
```

Use the worktree path as the working root for all file reads, edits, git commands, and **test commands** throughout this session. The worktree IS a complete checkout — when the test command says "run from repo root," that means run from `<worktree-path>`. Use absolute paths or `git -C <worktree-path>` rather than `cd`.

Read all source files relevant to this phase before writing any code. The orchestrator passed you a list of relevant file paths — read them yourself; do not assume their contents have been pre-read for you. Understand the existing structure fully before changing it.

## 2. The TDD loop

Work through the phase success criteria **one at a time, in order**.

### Backend and non-UI frontend logic (reducers, utilities, validation helpers)

**CRITICAL: Work strictly one criterion at a time. Do not write tests for future criteria or implement beyond the current criterion. Complete the full loop (test → red → implement → green) for criterion N before touching criterion N+1.**

For each criterion:

1. **Write one failing test** — and only one. Use the TDD entry point from the phase issue as the anchor for the first criterion. Each subsequent criterion gets its own targeted test asserting the outermost observable behavior — what a caller sees, not internal state. Do not write tests for any other criterion at this step.
2. **Run the test suite. Confirm red.** Use the test command from `.claude/plan-workflow-config.yml`, run from the worktree directory. The new test must fail. If it passes immediately, the behavior already exists — note it, skip implementation, move to the next criterion.
3. **Write the minimum code to make the test pass.** Do not implement beyond what the current test requires. Do not add code in anticipation of future criteria.
4. **Run the test suite. Confirm green.** All previously passing tests must still pass.
5. If still failing after implementation: diagnose, fix, re-run — up to 3 attempts total.
6. Only after green: move to the next criterion and repeat from step 1.

### Rendered controls (markup and styling only)

The TDD exemption is narrow: it applies **only** to the JSX/template markup, styling, layout, and rendering side-effects of a component. For these, implement the changes and note manual verification steps.

**Everything else inside a component file must be extracted and tested.** No exceptions:

- **Event handlers** (save, submit, click, change) that transform, filter, derive, or validate values before calling an API, dispatch, or store mutation — extract to a non-component module and apply the full TDD loop there.
- **Derived state** — any value computed from props, hook outputs, or store reads before being rendered — extract to a tested helper.
- **Validation, formatting, parsing, sorting, mapping** done inline — extract and test.
- **Conditional rendering driven by non-trivial expressions** — extract the predicate to a tested helper, render based on its result.

The litmus test: **if the code can be tested with the DOM removed, it is logic — extract it.** Manual tracing in the self-review (section 4) is a backstop, not a substitute for tests. If you find yourself reasoning about a derivation or handler in your head, that is the signal to extract.

## 3. Failure handling

**Test fails after 3 fix attempts**: Stop the loop. Return a summary to the caller:
- Which criterion failed
- What the test asserts
- Actual vs. expected output
- What was attempted

Do not make further changes.

**Repeated stalling**: If you have hit the 3-attempt limit on more than one criterion, or the same category of failure keeps recurring across different criteria, include this note in your escalation report:
> "Progress has stalled on multiple criteria. Consider re-running `/implement-phase` with the model upgraded to opus for better reasoning on this phase."

**Requirement ambiguity** (phase issue and impl plan conflict, or a requirement is underspecified): Stop the loop. Describe the ambiguity specifically. Return to the caller for input.

**Unexpected existing behavior** (a change breaks unrelated tests): Stop and diagnose before proceeding. Do not suppress or delete failing tests to make the suite pass.

## 4. Self-review (mandatory, before staging)

After all success criteria are implemented and the suite is green, perform this review **before staging any file**. These checks catch common errors that pass tests but break in production. Run each check; record any fixes you make.

1. **Trace critical data paths.** For every handler that transforms data before sending it to an API, dispatch, or persistent store: follow each value from its source (state, props, hook output, function arg) through every transformation to the final call. Confirm the value matches the intended *post-mutation* state, not a stale intermediate. This applies whether the handler is in a component or extracted.

2. **Check exports and registrations.** For every new module, function, type, or component you created, verify it is exported wherever its peers are exported (barrel files, public-API modules, route registries, plugin manifests, dependency-injection containers — whatever convention this project uses). Missing registrations must be added before committing.

3. **Check declaration order in stateful code.** Where the project's framework requires a particular order (e.g. hook calls before derived values that read them; state declarations before handlers that close over them), confirm that order is preserved. Out-of-order declarations often pass type checks but fail at runtime.

4. **Check for unintended duplication.** If you rendered the same data, registered the same listener, or added the same migration step in more than one place, confirm each occurrence is intentional. Remove redundant copies.

If any check produces a fix, re-run the test suite to confirm green, then proceed.

## 5. Finishing

When the self-review is clean and the suite is green:

### Stage and commit

Stage only the files you created or modified during this phase. Do not stage untracked files you did not intentionally create. Run git commands from inside the worktree:

```bash
git -C <worktree-path> add <each modified or created file by path>
git -C <worktree-path> commit -m "[PLAN-XXXXX-N] <phase title>"
```

### Push

```bash
git -C <worktree-path> push
```

### Create PR

Target the implementation branch — **not main**.

Read `../../SHARED/PR_TEMPLATE.md` and use the **Phase PR** template.

```bash
gh pr create --repo <repo> \
  --title "[PLAN-XXXXX-N] <Phase Title>" \
  --base <impl-branch> \
  --body "$(cat <<'EOF'
<PR body per template>
EOF
)"
```

The PR body must include `Closes #<phase-issue-number>`.

### Move phase issue to In Review

```bash
marvin board move <phase-issue-number> in-review
```

### Return summary

Report back to the `implement-phase` caller:
- PR URL
- Test results (pass count, any skipped)
- List of success criteria completed
- Anything requiring manual verification (UI steps)

## 6. Handling correction commits after the PR is open

If a reviewer flags an issue, or you discover a defect after the initial push, you may push correction commits to the same PR branch. **Each correction must be recorded in the PR body's Notes section** so reviewers have a durable record without reconstructing it from commit history.

For each correction commit:

1. Push the correction.
2. Fetch the current PR body:
   ```bash
   gh pr view <pr-number> --repo <repo> --json body --jq '.body'
   ```
3. Append (or insert into the existing Notes section) one entry of the form:
   ```
   - **<What changed>**: <Why it was wrong> → <What the correct approach is and why>.
   ```
4. Update the PR body:
   ```bash
   gh pr edit <pr-number> --repo <repo> --body "$(cat <<'EOF'
   <updated body>
   EOF
   )"
   ```

This applies whether the correction was self-caught or raised in review. The Notes entries are read by `wrap-phase` after merge to capture decisions and corrections back onto the impl plan.

## 7. Operating mode

You are running autonomously. Do not pause for user confirmation at individual steps. Only stop and return to the caller on unresolvable failure or ambiguity.
