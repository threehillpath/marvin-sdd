# Naming Edge Cases

## Multiple Impl Plans from One Arch Plan

When one architecture plan spawns more than one independent implementation plan (e.g., backend and frontend are large enough to be separate plans), append a letter suffix:

- Arch plan: `[PLAN-014-ARCH] Feature Name`
- Impl plan A: `[PLAN-014A] Feature Name — Backend`
- Impl plan B: `[PLAN-014B] Feature Name — Frontend`
- Phases for A: `[PLAN-014A-1]`, `[PLAN-014A-2]`, ...
- Phases for B: `[PLAN-014B-1]`, `[PLAN-014B-2]`, ...

This should be rare — phases within a single impl plan are preferred over splitting into multiple impl plans. Use this only when the two bodies of work are truly independent and could be developed in parallel by different people.
