# skills/

## Layout

```
SHARED/        Files referenced by multiple skills
  CONFIG.md                Template for per-project config; template-override precedence
  GLOSSARY.md              Names, paths, status state machine, docs/stories layout
  LABELS.md                Label rules
  PR_TEMPLATE.md           PR body templates
  RENDERING.md             Markdown output guidance
  REVIEW_RUBRIC.md         Code-review rubric (review-phase / review-impl / quick-task)
  REVIEW_FINDING_FORMAT.md Review findings JSON schema
  PLAN_RED_TEAM_RUBRIC.md / PLAN_RED_TEAM_FORMAT.md   red-team-plan rubric + findings schema
  PLAN_DRIFT_RUBRIC.md / PLAN_DRIFT_FORMAT.md         plan-drift rubric + findings schema
<skill-name>/
  SKILL.md     Authoritative skill prompt
  SUPPLEMENTS/ Templates and deeper guidance
```

## Authoring conventions

- Each `SKILL.md` declares its model (opus for planning, sonnet for implementation, haiku for board ops) and allowed tools in frontmatter.
- Skills reference SHARED files rather than re-defining names, statuses, PR templates, or label rules.
- Heavy code-reading is delegated to subagents (Explore for code digestion, general-purpose for autonomous work) so the orchestrating skill's context stays small.
- A sub-agent whose depth matters (review, red-team, drift, TDD implementation) is a plugin agent in `agents/` with `model` and `effort` in its frontmatter; prompt text such as "use extended thinking" does not change depth.

## Auxiliary and standalone skills

`move-issue`, `finish-phase`, `plan-drift`, and `unslop` are usable at any point — `plan-drift` is most valuable mid-phase or before opening a PR.

`quick-task` drives a Task issue from filed requirement to merged PR, including all of that issue's board transitions, in one invocation, with the same TDD and review rigor as the phased pipeline. It skips arch-plan → impl-plan → phase-split.
