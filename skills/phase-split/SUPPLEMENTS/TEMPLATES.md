# Phase Issue Template

```markdown
# [PLAN-XXX-N] <Phase Title>

**Implementation Plan**: #<impl-plan-issue>
**Plan Number**: PLAN-XXX, Phase N
**Status**: Upcoming

---

## Objective

<One sentence: what this phase delivers and why it's a natural unit.>

## Scope

**Includes:**
- <bullet>

**Depends on:** Phase <N-1> / none

**Next phase:** Phase <N+1> builds on this by <brief description> / none (final phase)

---

## TDD Entry Point

<First test to write before any implementation. Omit this section for UI-only phases.>

What: <one sentence on the observable behavior being asserted>
Where: <test file path pattern>
Passes when: <observable outcome>

---

## Components

<Brief spec for each component in this phase. Reference the impl plan for full detail.>

---

## Verification

<Concrete command(s) to confirm this phase is complete and ready to merge.>

---

## Success Criteria

- [ ] TDD entry point test written and failing (backend phases only)
- [ ] Implementation complete and test passing
- [ ] PR reviewed and merged
```
