# PR Body Templates

These templates are referenced by `implement-phase` (LOOP.md), `finish-phase`, `finish-impl`, and `quick-task` (LOOP.md). Use the matching template for the level of PR being opened.

## Phase PR (phase branch → implementation branch)

```markdown
## Summary

<2-4 bullet points describing what this phase delivers. Focus on behavior, not file changes.>

## Phase

Closes #<phase-issue-number> — [PLAN-XXXXX-N] <Phase Title>

**Implementation plan**: #<impl-plan-issue>

## Test plan

<Bulleted checklist of what to verify before merging. Include the TDD entry point test if this is a backend phase. UI phases should describe manual verification steps.>

- [ ] <verification step>
- [ ] <verification step>

## Notes

<Optional: anything a reviewer needs to know — migration steps, env var changes, known limitations.

If correction commits are pushed to this PR after it is opened (to address review feedback or self-caught issues), append an entry here for each correction in the form:

- **<What changed>**: <Why it was wrong> → <What the correct approach is and why>.>
```

## Task PR (task branch → main)

```markdown
## Summary

<2-4 bullet points describing what this Task delivers. Focus on behavior, not file changes.>

## Task

Closes #<task-issue-number> — [TASK-XXXXX] <Task Title>

## Test plan

<Bulleted checklist of what to verify before merging. Include the TDD entry point test.>

- [ ] <verification step>
- [ ] <verification step>

## Notes

<Optional: anything a reviewer needs to know — migration steps, env var changes, known limitations.

If correction commits are pushed to this PR after it is opened (to address review feedback or self-caught issues), append an entry here for each correction in the form:

- **<What changed>**: <Why it was wrong> → <What the correct approach is and why>.>
```

A Task PR always targets `main` — a Task has no phase or implementation-plan hierarchy, so unlike the Phase PR template above, it carries no "Phase:"/"Implementation plan:" metadata lines.

## Implementation PR (implementation branch → main)

```markdown
## Summary

<3-6 bullet points describing what the full implementation delivers across all phases. Focus on user-visible behavior and architectural changes, not file-by-file detail.>

## Implementation

Closes #<impl-plan-issue> — [PLAN-XXXXX] <Impl Plan Title>

**Architecture plan**: #<arch-plan-issue>
**Source issue**: #<source-issue>

**Phases merged:**
- #<phase-1-issue> [PLAN-XXXXX-1] <Title>
- #<phase-2-issue> [PLAN-XXXXX-2] <Title>
- ...

## Test plan

<Bulleted checklist of end-to-end verification steps. Each phase has already passed its own tests; this section covers integration scenarios that span phases.>

- [ ] <verification step>
- [ ] <verification step>

## Notes

<Optional: deployment ordering, env var changes, migration steps, follow-up work intentionally deferred.>
```
