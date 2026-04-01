# ADR Candidates

Flag a decision as an ADR candidate when it meets these criteria:

## When to Flag

- The decision establishes a pattern that will apply to future features (not just this one)
- The decision involves a meaningful trade-off others would need to understand to make consistent choices
- The decision overrides or refines an existing architectural principle

## When NOT to Flag

- The decision is obvious from the codebase or standard practice
- The decision is specific to this feature and won't recur
- The decision is temporary or experimental

## Examples

| Scenario | Flag? |
|---|---|
| Choosing async over sync for a new command type | Yes — establishes pattern for similar commands |
| Choosing a specific table name | No — implementation detail |
| Deciding that lifecycle end dates are NULL for active records | Yes — affects all lifecycle tables |
| Adding a validation rule for email format | No — feature-specific |

## What to Record in the Arch Plan

Just flag it:
```
- [ ] <Decision name> — <one sentence on why it's reusable>
```

Do not create the ADR file during arch-plan. ADR files are created separately after review and approval, to keep the planning step focused.
