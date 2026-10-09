# agents/

Plugin sub-agents with fixed `model` and `effort` in frontmatter. Claude Code takes thinking depth from `effort`.

| Agent | Model / effort | Spawned by | Rubric → output contract |
|---|---|---|---|
| `code-reviewer` | opus / xhigh | `review-phase`, `review-impl`, `quick-task` | `SHARED/REVIEW_RUBRIC.md` → `SHARED/REVIEW_FINDING_FORMAT.md` |
| `plan-red-teamer` | opus / xhigh | `red-team-plan` | `SHARED/PLAN_RED_TEAM_RUBRIC.md` → `SHARED/PLAN_RED_TEAM_FORMAT.md` |
| `drift-auditor` | sonnet | `plan-drift` | `SHARED/PLAN_DRIFT_RUBRIC.md` → `SHARED/PLAN_DRIFT_FORMAT.md` |
| `tdd-implementer` | sonnet | `implement-phase`, `quick-task` | LOOP.md pasted into its prompt |

All paths under `skills/`. The first three are fresh-context, read-only spec checkers:

- **red-team-plan** catches hidden assumptions, missing dependencies, weak TDD entry points, and unfalsifiable success criteria *before* phase-split, where errors compound.
- **plan-drift** tracks per-criterion coverage and out-of-scope/interface-divergence containment. It complements but does not replace `review-phase`.
- **code-reviewer** in `quick-task` reuses the same rubric, extended with a Task-specific spec-drift clause.

Each findings JSON is the stable contract a future auto-fix loop will consume.
