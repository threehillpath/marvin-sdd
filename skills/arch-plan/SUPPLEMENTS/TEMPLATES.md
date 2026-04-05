# Arch Plan Template

```markdown
# [PLAN-XXXXX-ARCH] <Title>

**Source Issue**: #<N> — <title>
**Plan Number**: PLAN-XXXXX
**Author**: Claude (Architect)
**Status**: Draft
**Date**: <today>

---

## Problem Statement

<One paragraph: what need is being addressed and why it matters now.>

## Scope

**Includes:**
- <bullet>

**Excludes:**
- <bullet> (covered by <reference> / deferred / out of scope)

## Domain Model Impacts

<Which entities are created, modified, or related. What lifecycle or relationship rules apply.>

## Integration Points

<Which layers are touched: handlers, workers, repos, domain types, schema. What connects — not how.>

## Cross-Cutting Concerns

<Auth/authz requirements, validation boundaries, sync vs async, error handling strategy, ordering or concurrency concerns.>

## Architectural Decisions

| Decision | Chosen Approach | Rationale |
|---|---|---|
| | | |

## ADR Candidates

Decisions broad enough to become reusable architectural records. Flag here; create ADR files separately.

- [ ] <Decision> — <why it's reusable across features>

## Constraints and Trade-offs

<What shapes the solution. What trade-offs are accepted and why.>

## TDD Strategy

<The outermost observable behavior to assert first. What integration or acceptance test anchors this work?>

## Open Questions

<Anything unresolved that the implementation plan must address.>
```
