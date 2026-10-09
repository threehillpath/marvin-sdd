# tool/ — the marvin CLI

`marvin` is compiled from this module by `tool/build.sh`, which writes the binary to a caller-supplied output path and skips the build when that binary is already newer than every file under `tool/`. The `SessionStart` hook (`hooks/hooks.json`) builds it into `${CLAUDE_PLUGIN_ROOT}/bin/marvin` on every session start.

Skills call it for all deterministic operations — board moves, issue create and edit, template render and validate, label management, config access, name derivation, PR lookup, worktree lifecycle, findings cache — so none of that logic is re-synthesized from shell in skill prose.

Subcommand groups: `config`, `names`, `parse`, `template`, `board`, `issue`, `label`, `pr`, `findings`, `worktree`, `version`.

## Contract

- Exit codes: `0` success, `1` operational error, `2` config missing or malformed, `3` a draft or body that does not conform to its plan schema (constants in `internal/clierr/`).
- `stdout` = data, `stderr` = diagnostics. The one exception is `marvin template validate`, whose findings are its data and go to stdout.
- Errors must not fail silently (see root `CLAUDE.md`).

## Layout

```
cmd/marvin/main.go   Entry point
internal/
  board/      GitHub Projects v2 operations (add, move, list, status)
  issue/      Issue reads (label/prefix/state filters) and plain gh create/edit wrappers
  cli/        Cobra handlers (issue create/edit run the template/ check here before calling issue/)
  clierr/     Exit-code constants
  config/     YAML loader, legacy markdown fallback, CWD-walk discovery
  exec/       Runner interface (injectable for tests)
  exectest/   Fake runner for unit tests (no network / no git state)
  findings/   Findings cache read/write
  gh/         gh JSON client wrapper
  label/      Label ensure-exists
  names/      Plan name derivation (PLAN-XXXXX, branch, worktree path, prefix)
  parse/      Identifier parsing (issue titles → plan numbers)
  pr/         PR discovery and target resolution
  template/   Plan-template render; validation of YAML drafts and markdown bodies
    schemas/  Built-in schemas, embedded via go:embed — canonical source
              (arch-plan, impl-plan, impl-phase, quick-task)
  worktree/   Worktree lifecycle (create, remove, prune, repo-root path resolution)
```

## Templates

`marvin template render` checks the consuming project for `.claude/plan-workflow-templates/{type}.yml` (CWD-walk) before falling back to the built-in schema compiled into the binary, so rendering never depends on plugin source being on disk. `template validate`, `issue create --template` and `issue edit --template` use the same precedence, so a draft is checked against the schema that rendered its skeleton. Full precedence and draft format: `skills/SHARED/CONFIG.md`.
