# Phase Sizing and Boundary Guidance

## What makes a good phase boundary

A phase should be:
- **Logically atomic**: one coherent unit of change — a layer, a feature slice, or a single behavior
- **Independently verifiable**: something testable on its own before the next phase starts
- **Sized for one context**: roughly one focused work session — not so large it requires context-switching mid-phase
- **Branch and PR worthy**: produces a clean, reviewable diff with a clear description

## Natural boundary signals

Split at these points:
- Schema migration → separately verifiable before building on top of it
- Domain/repo layer → testable before the handler or worker that calls it
- Backend complete → separately mergeable before UI work begins
- Async path added → can be verified independently of the sync path
- A component that other phases depend on

## TDD and phases

Each phase (except UI-only phases) should begin with writing the entry point test from the impl plan before any implementation. The phase issue carries the TDD entry point forward from the impl plan.

UI rendering phases are exempt from TDD requirements — components and DOM-dependent code are not practical to unit test. However, a phase that includes frontend logic without DOM dependency (reducers, utility functions, validation helpers, data transformations) is **not** exempt and requires a TDD entry point for those components.

## Anti-patterns to avoid

- Phases that can't be tested or merged without the next phase also being complete
- Phases so small they're just housekeeping (combine them)
- Phases so large they touch multiple layers and multiple behaviors (split them)
- Grouping unrelated changes just because they're small
