# Markdown Rendering

Skills that present drafted documents for review must include the full draft directly in their response so the user can read it.

## Delivery mechanism

The content must appear directly in the agent's own text response to the user — that is what the user sees. A Bash tool call's stdout is never shown to the user in the Claude Code CLI harness, only the agent's own response text is, so piping a draft through a Bash command (`cat`, `echo`, `glow`, etc.) as the way to present it will leave the user seeing nothing. Always write or paste the draft's markdown directly into your reply.

## glow (optional local preview)

`glow` is a terminal markdown renderer. It is not part of the delivery mechanism above and the agent should not invoke it to show content to the user. It is only useful as a convenience for a human who is running commands directly in their own terminal and wants a rendered local preview — e.g. `glow some-file.md`. If a human asks how to preview a draft locally, `glow` (`brew install glow`) is a reasonable suggestion; the GitHub issue URL (available after creation) also provides a rendered view in the browser.

## When to use

Include the full markdown in your response text when presenting a full document draft for approval (arch plans, impl plans, phase proposals). Plain text is fine for short status confirmations.
