# Markdown Rendering

Skills that present drafted documents for review should render them readably when possible.

## glow (optional)

`glow` is a terminal markdown renderer. It is not required by this plugin but significantly improves readability of plan drafts.

Install: `brew install glow`

Usage:
```bash
which glow && echo "<content>" | glow - || echo "<content>"
```

If not installed, print raw markdown and note the install command. The GitHub issue URL (available after creation) provides a rendered view in the browser as an alternative.

## When to use

Use rendered output when presenting a full document draft for approval (arch plans, impl plans, phase proposals). Plain text is fine for short status confirmations.
