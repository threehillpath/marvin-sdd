# Phase Doc Template

Render the phase's original spec plus the sub-agent's classification JSON (`SUPPLEMENTS/CLASSIFY.md`) into `docs/stories/<plan>/phase-NN-<slug>.md` — a standalone, deep-dive record of one phase meant to remain useful once the source GitHub issue and PR are no longer easily reachable. Keep this file distinct from the wrap-up comment (`COMMENT_TEMPLATE.md`, posted to the impl plan issue): the comment carries only the four classification categories; this file carries the full spec plus the implementation narrative and test results too.

## Template

```markdown
# Phase N: <Phase Title>

**Status**: Completed
**Phase issue**: [#<phase-issue-number>](<phase-issue-url>)
**Pull request**: [#<pr-number>](<pr-url>)
**Implementation plan**: [#<impl-plan-issue-number>](<impl-plan-issue-url>)

<Phase issue body verbatim, starting at its first `##` section (Objective, Scope, TDD Entry Point, Components, Verification, Success Criteria) — everything the phase issue's own body already contains except its title line and metadata block, which are superseded by the header above.>

---

## Implementation

<implementation_summary, verbatim>

### Test Plan / Verification

- [x] <checked test_plan step>
- [ ] <unchecked test_plan step>
- ...

### Decisions

- **<summary>** — <reasoning>
- ...

### Scope Changes

- **<direction: added | removed>:** <summary> — <reason>
- ...

### Deferred / Watch Items

- **<summary>** — track in <where_to_track>
- ...

### Corrections

- **<what_changed>**: <why_wrong> → <correction>.
- ...
```

## Rules

- **Section order is fixed** as shown above. Do not reorder.
- **`N` is the phase's parsed ordinal** (from `marvin parse title`), not the GitHub issue number. Zero-pad it to two digits in the filename (`phase-01-...`, `phase-12-...`) but write it unpadded in the `# Phase N:` heading.
- **`<slug>` comes from `marvin parse title`'s `slug:` output** for the phase issue's title — never hand-slugify. This is what lets `finish-impl` link to the exact filename this skill wrote, without the two skills needing to agree on slugification logic independently.
- **Reuse the phase issue body verbatim** for everything from its first `##` heading onward — do not re-summarize or re-word the spec. Writing the spec here, once, is what lets `finish-impl` skip re-fetching each phase issue's body entirely when it later builds `docs/stories/<plan>/README.md`'s phase index.
- **`### Test Plan / Verification` renders every `test_plan` entry**, checked or not, in the order the PR body listed them — including entries that stayed unchecked at merge (e.g. a deferred manual check). This is a factual record of what was and wasn't verified, not a checklist to complete now.
- **Omit an empty `###` section entirely** (Decisions, Scope Changes, Deferred / Watch Items, Corrections) — including its heading — exactly as `COMMENT_TEMPLATE.md` does. `## Implementation` and `### Test Plan / Verification` are never omitted (even an empty-string `implementation_summary` still gets its heading, per `CLASSIFY.md`'s note on that edge case), since they exist independently of whether anything notable happened category-wise.
- **Corrections has no `(<count>)` suffix here**, unlike the wrap-up comment — the comment needs the count as an at-a-glance signal on a GitHub issue thread; this file's reader is already reading the full detail.
