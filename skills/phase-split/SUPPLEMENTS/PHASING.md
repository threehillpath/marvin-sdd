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

Each phase should begin with writing the entry point test from the impl plan before any implementation. The phase issue carries the TDD entry point forward from the impl plan.

The TDD exemption is narrow: it applies **only** to rendered controls — JSX/template markup, styling, and rendering side-effects. A phase whose only deliverable is markup may omit a TDD entry point for that work. But any phase that touches event handlers, derived state, validation, formatters, parsers, or conditional-render predicates **must** include a TDD entry point — even when that logic currently lives inside a component file. Plan to extract such logic to a tested helper as part of the phase. See `../../impl-plan/SUPPLEMENTS/TDD.md` for the full rule.

## Anti-patterns to avoid

- Phases that can't be tested or merged without the next phase also being complete
- Phases so small they're just housekeeping (combine them)
- Phases so large they touch multiple layers and multiple behaviors (split them)
- Grouping unrelated changes just because they're small
