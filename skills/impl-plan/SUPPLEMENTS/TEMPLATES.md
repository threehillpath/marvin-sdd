# Implementation Plan Template

```markdown
# [PLAN-XXX] <Title>

**Objective**: <one sentence>
**Architecture Plan**: #<arch-issue-number>
**Source Issue**: #<source-issue-number>
**Author**: Claude
**Status**: Upcoming
**Last Updated**: <today>

---

## Scope

**Includes:**
- <bullet>

**Does NOT include:**
- <bullet> (covered by <reference>)

---

## 1. <Component or Layer Name>

### Specifications
<Files to create/modify. Key type names. Function signatures.>

### Behavior
<What it must do. Edge cases. Error handling requirements.>

### TDD Entry Point
<First test to write: what observable behavior does it assert? Where does the test live?>

---

## N. Verification Steps

### N.1 <Scenario>
<Command to run>
# Expected: <what should happen>

---

## Design Notes

<Rationale for non-obvious decisions. Reference the arch plan for higher-level decisions already made there.>

---

## Success Criteria

- [ ] <measurable requirement>
- [ ] <measurable requirement>
```
