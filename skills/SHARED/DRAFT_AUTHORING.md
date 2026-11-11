# Draft authoring

How to fill a plan-issue YAML draft and create the issue from it. Used by `arch-plan`, `impl-plan`, `phase-split` and `quick-task`; each skill supplies its own schema name, title placeholders, metadata specifics and draft path. `CONFIG.md` describes the schema format itself.

## Filling a draft

Besides every rule `marvin template render <schema> --guidance` prints (its code-fence, heading-underline and HTML rules are not repeated here), the loader enforces key, tab, tag, anchor and alias rules that `--guidance` does not print. The rules most often broken:

- Section content is always a `|` block, never `>` or an inline value. A repeatable section is one list under one key, not the key repeated.
- The title and every metadata value are always double-quoted. Write `\"` for a quote and `\\` for a backslash inside them.
- Every metadata value must be non-empty. The skill names the metadata keys and how each is derived.
- Each key appears once. Indent with spaces, never tabs.
- No `#` or `##` heading lines in content; use `###` or deeper. Escape a literal `#` at the start of a line as `\#`, except inside a code fence, where a `#` line is code and must stay unescaped (`\#` would print as is).
- No YAML comments, no `---` or `...` at column 0, no tags, anchors or aliases.

Write the finished draft to `<project-root>/.claude/cache/<plan-or-task>/<name>.yml`, where `<project-root>` is the main checkout's git root (not a linked worktree), as defined in `CONFIG.md`. `Write` creates the directory. If the file already exists, `Read` it first (`Write` refuses to overwrite a file it has not read), then overwrite it. Quote the path in every command, because it may contain spaces.

## Presenting a draft

Show the user the rendered issue, not the YAML: `marvin template render <schema> --draft "<path>"`. Stdout is the issue body as markdown. Show the `schema:` line from stderr (and any warnings) above the pasted body, so the user sees which schema shaped it. Paste the draft's `title:` and that markdown into your reply, because a Bash result is not shown to the user (see `RENDERING.md`).

- Exit 1 (for example a malformed project override): show stderr to the user and stop.
- Exit 3: the draft does not conform. Rewrite the draft file with `Write` to fix the findings on stderr and render again. Make at most 3 fix-and-render attempts per round of user changes; if it still exits 3, show the user the findings and ask how to proceed.
- When the user asks for changes, rewrite the draft file with `Write` and render again.

## Creating the issue

`--label` takes one comma-joined string of only the labels that exist. Leave out any absent part, because marvin would pass an empty `--label` to `gh` for an empty entry or a leading or trailing comma. Put no spaces around the commas. Example: `--label "plan:impl,status:upcoming,domain:backend"`. Which labels apply is the skill's call (see `LABELS.md`).

The title comes from the draft, so do not pass `--title` or `--body`. On success stdout is the new issue number, then its URL; capture both. Warnings on stderr are fine. Handle the exit code:

- **0** — created.
- **3** — the draft does not conform; nothing was created. The findings are on stderr, most naming a draft line, all saying how to fix it. Rewrite the draft file with `Write` to fix the findings. If the fix changes only how the draft is written (quoting, escaping, block style, heading depth) and not what it says, run the same command again. If it changes what the draft says (content added, removed or reworded, or the title), render the draft again, show the user, and get approval before running the command again. Make at most 3 fix attempts per round of user changes; if it still exits 3, show the user the findings and stop.
- **1** — a usage or operational error (for example an unreadable draft, or several usage problems listed together under a header like `issue create: 3 problems:`). Findings may be printed with it. Show stderr to the user and stop. Do not retry: the same input fails the same way.
- **2** — configuration missing. Surface: "Configuration missing — run `/configure-plan-plugin` first." Do not retry: it stays missing until the user configures the project.
