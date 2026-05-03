# Wrap-up Comment Template

Render the sub-agent's classification JSON into a markdown comment posted on the impl plan issue. Skip any section whose array is empty. If all four arrays are empty, render only the header and the "no notable history" line.

## Template

```markdown
## Phase wrap-up: [PLAN-XXXXX-N] <Phase Title>

Merged via #<pr-number>. <If all arrays empty:> No notable decisions, scope changes, deferred items, or corrections in this phase.

### Decisions

- **<summary>** — <reasoning>
- ...

### Scope changes

- **<direction: added | removed>:** <summary> — <reason>
- ...

### Deferred / watch items

- **<summary>** — track in <where_to_track>
- ...

### Corrections (<count>)

- **<what_changed>**: <why_wrong> → <correction>.
- ...
```

## Rules

- **Section order is fixed** as shown above. Do not reorder.
- **Omit empty sections entirely** — including the header — rather than rendering "(none)".
- **Corrections section header includes the count** (e.g. `### Corrections (3)`) so reviewers can see at a glance how much rework happened. If zero, omit the section.
- The **first line under the header** always references the merged PR number so the comment is self-contained.
- Use the phase issue's exact title as written in `gh issue view`.
