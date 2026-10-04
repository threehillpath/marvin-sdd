# PLAN-00112: Schema-checked plan issue creation and edits

Development documentation for PLAN-00112, the Schema-checked plan issue creation and edits implementation plan. This directory is a historical and reference record of the plan's architecture, scope, phase-by-phase implementation, and the decisions/corrections made along the way — generated from the plan's GitHub issues, pull requests, and review history.

**Plan status**: All 7 phases complete.

## Contents

- **[architecture.md](./architecture.md)** — the original architecture plan (issue #131): system design, data model, and component boundaries decided before implementation began.
- **[implementation-plan.md](./implementation-plan.md)** — the implementation plan (issue #132): how the architecture was broken into phases, with a phase index table below.
- **[retrospective.md](./retrospective.md)** — a cross-phase synthesis of decisions, scope changes, deferred items, and corrections, plus the pre-implementation red-team critique if one was run.
- **`phase-01-*.md` through `phase-07-*.md`** — one document per phase: its original spec (objective, scope, TDD entry point, success criteria) plus an implementation summary, test results, decisions, scope changes, deferred items, and corrections made during code review.

## Phase index

| # | Phase | Doc |
|---|-------|-----|
| 1 | [PLAN-00112-1] Title identifier kinds | [phase-01-title-identifier-kinds.md](./phase-01-title-identifier-kinds.md) |
| 2 | [PLAN-00112-2] Section map and conformance check | [phase-02-section-map-and-conformance-check.md](./phase-02-section-map-and-conformance-check.md) |
| 3 | [PLAN-00112-3] YAML draft loader, renderer and skeleton | [phase-03-yaml-draft-loader-renderer-and-skeleton.md](./phase-03-yaml-draft-loader-renderer-and-skeleton.md) |
| 4 | [PLAN-00112-4] Markdown body parser | [phase-04-markdown-body-parser.md](./phase-04-markdown-body-parser.md) |
| 5 | [PLAN-00112-5] CLI validate, render --draft and exit code 3 | [phase-05-cli-validate-render-draft-and-exit-code-3.md](./phase-05-cli-validate-render-draft-and-exit-code-3.md) |
| 6 | [PLAN-00112-6] Checked issue create and issue edit | [phase-06-checked-issue-create-and-issue-edit.md](./phase-06-checked-issue-create-and-issue-edit.md) |
| 7 | [PLAN-00112-7] Skill adoption and docs | [phase-07-skill-adoption-and-docs.md](./phase-07-skill-adoption-and-docs.md) |
