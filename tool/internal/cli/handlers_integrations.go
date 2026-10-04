package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"threehillpath.com/marvin-sdd/tool/internal/board"
	"threehillpath.com/marvin-sdd/tool/internal/clierr"
	"threehillpath.com/marvin-sdd/tool/internal/config"
	"threehillpath.com/marvin-sdd/tool/internal/exec"
	"threehillpath.com/marvin-sdd/tool/internal/findings"
	"threehillpath.com/marvin-sdd/tool/internal/gh"
	"threehillpath.com/marvin-sdd/tool/internal/issue"
	"threehillpath.com/marvin-sdd/tool/internal/label"
	"threehillpath.com/marvin-sdd/tool/internal/pr"
	tmplpkg "threehillpath.com/marvin-sdd/tool/internal/template"
	"threehillpath.com/marvin-sdd/tool/internal/worktree"
)

// rejectEmptyFlags returns an exit-1 usage error for the first of the named
// flags that was passed with an empty value. An empty path must never fall
// through to a "neither given" path. prefix is prepended to the message.
func rejectEmptyFlags(cmd *cobra.Command, prefix string, names ...string) error {
	for _, n := range names {
		if f := cmd.Flags().Lookup(n); f != nil && f.Changed && f.Value.String() == "" {
			return &CLIError{Code: 1, Msg: emptyFlagMsg(prefix, n)}
		}
	}
	return nil
}

// loadConfig loads plan-workflow config from the current working directory.
func loadConfig() (*config.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, &CLIError{Code: 1, Msg: fmt.Sprintf("cannot determine working directory: %v", err)}
	}
	return config.Load(cwd)
}

// ── board ─────────────────────────────────────────────────────────────────────

// newBoardCmd returns the board subcommand group.
func newBoardCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	board := &cobra.Command{
		Use:   "board",
		Short: "Manage GitHub Projects v2 board state",
	}
	board.AddCommand(newBoardAddCmd(stdout, stderr, runner))
	board.AddCommand(newBoardSetStatusCmd(stdout, stderr, runner))
	board.AddCommand(newBoardMoveCmd(stdout, stderr, runner))
	board.AddCommand(newBoardListCmd(stdout, stderr, runner))
	board.AddCommand(newBoardStatusCmd(stdout, stderr, runner))
	return board
}

func newBoardAddCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "add <issue-number>",
		Short: "Add an issue to the board and print the item ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return &CLIError{Code: 1, Msg: fmt.Sprintf("invalid issue number %q: %v", args[0], err)}
			}
			id, err := board.AddItem(context.Background(), runner, cfg, n)
			if err != nil {
				return &CLIError{Code: 1, Msg: err.Error()}
			}
			fmt.Fprintln(stdout, id)
			return nil
		},
	}
}

func newBoardSetStatusCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "set-status <issue-number> <status>",
		Short: "Add an issue to the board and set its status field",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return &CLIError{Code: 1, Msg: fmt.Sprintf("invalid issue number %q: %v", args[0], err)}
			}
			if err := board.SetStatus(context.Background(), runner, cfg, n, args[1]); err != nil {
				return &CLIError{Code: 1, Msg: err.Error()}
			}
			return nil
		},
	}
}

func newBoardMoveCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "move <issue-number> <status>",
		Short: "Add issue to board, set status, and sync open/closed state",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			issueNumber, err := strconv.Atoi(args[0])
			if err != nil {
				return &CLIError{Code: 1, Msg: fmt.Sprintf("invalid issue number %q: %v", args[0], err)}
			}
			if err := board.Move(context.Background(), runner, cfg, issueNumber, args[1]); err != nil {
				return &CLIError{Code: 1, Msg: err.Error()}
			}
			return nil
		},
	}
}

func newBoardListCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	var statusFilter string
	var limit int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List board items as plain text (--json for JSON), optionally filtered by status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			return runBoardList(stdout, stderr, cfg, statusFilter, limit, jsonOut, runner)
		},
	}
	cmd.Flags().StringVar(&statusFilter, "status", "", "Filter by status (e.g. in_progress, in_review, \"In Progress\")")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of items to fetch from the API")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON instead of plain text")
	return cmd
}

// runBoardList lists board items. jsonOut selects JSON output (--json); by
// default, output is pipe-delimited plain text with no header row:
// "<number> | <status> | <title>", one line per item, url omitted (title is
// last so a literal "|" in a title is still safe to parse).
func runBoardList(stdout, stderr io.Writer, cfg *config.Config, statusFilter string, limit int, jsonOut bool, runner exec.Runner) error {
	items, err := board.List(context.Background(), runner, cfg, statusFilter, limit)
	if err != nil {
		return &CLIError{Code: 1, Msg: err.Error()}
	}
	if items == nil {
		items = []board.BoardItem{}
	}

	if !jsonOut {
		for _, item := range items {
			fmt.Fprintf(stdout, "%d | %s | %s\n", item.Number, item.Status, item.Title)
		}
		return nil
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(items)
}

func newBoardStatusCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "status <issue-number>",
		Short: "Print the current board status for an issue (\"not-on-board\" if absent)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return &CLIError{Code: 1, Msg: fmt.Sprintf("invalid issue number %q: %v", args[0], err)}
			}
			status, err := board.Status(context.Background(), runner, cfg, n)
			if err != nil {
				return &CLIError{Code: 1, Msg: err.Error()}
			}
			fmt.Fprintln(stdout, status)
			return nil
		},
	}
}

// ── label ─────────────────────────────────────────────────────────────────────

// newLabelCmd returns the label subcommand group.
func newLabelCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	lbl := &cobra.Command{
		Use:   "label",
		Short: "Manage GitHub labels",
	}
	lbl.AddCommand(newLabelEnsureCmd(stdout, stderr, runner))
	return lbl
}

// builtinLabels is the set of plan-workflow labels with their defaults.
// Keys are label names; values are [description, color].
var builtinLabels = map[string][2]string{
	"plan:arch":          {"Architecture plans", "0075ca"},
	"plan:impl":          {"Implementation plan", "0075ca"},
	"plan:phase":         {"Phase / implementation unit", "0075ca"},
	"plan:task":          {"Single-cycle task or bug (no phase hierarchy)", "5319e7"},
	"status:upcoming":    {"Issue is newly created and awaiting work", "ededed"},
	"status:backlog":     {"Issue is in the backlog", "e4e669"},
	"status:in-progress": {"Issue is in progress", "fbca04"},
	"status:in-review":   {"Issue is in review", "fef2c0"},
	"status:done":        {"Issue is done", "0e8a16"},
}

func newLabelEnsureCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	var description, color string
	var useBuiltins bool

	cmd := &cobra.Command{
		Use:   "ensure [<name>]",
		Short: "Ensure a label exists (create if absent, no-op if present)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			ctx := context.Background()

			if useBuiltins {
				for name, defaults := range builtinLabels {
					desc := defaults[0]
					col := defaults[1]
					if err := label.Ensure(ctx, runner, cfg, name, desc, col); err != nil {
						return &CLIError{Code: 1, Msg: err.Error()}
					}
				}
				return nil
			}

			if len(args) == 0 {
				return &CLIError{Code: 1, Msg: "label ensure requires a name argument or --builtins flag"}
			}
			if err := label.Ensure(ctx, runner, cfg, args[0], description, color); err != nil {
				return &CLIError{Code: 1, Msg: err.Error()}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&description, "description", "", "Label description")
	cmd.Flags().StringVar(&color, "color", "cccccc", "Label color (6-digit hex, no #)")
	cmd.Flags().BoolVar(&useBuiltins, "builtins", false, "Ensure all built-in plan-workflow labels exist")
	return cmd
}

// ── pr ────────────────────────────────────────────────────────────────────────

// newPRCmd returns the pr subcommand group.
func newPRCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	prCmd := &cobra.Command{
		Use:   "pr",
		Short: "PR discovery and base-branch resolution",
	}
	prCmd.AddCommand(newPRFindCmd(stdout, stderr, runner))
	prCmd.AddCommand(newPRBaseCmd(stdout, stderr, runner))
	return prCmd
}

// prFindOutput is the JSON shape for pr find.
type prFindOutput struct {
	Found  bool   `json:"found"`
	Number int    `json:"number,omitempty"`
	Title  string `json:"title,omitempty"`
	URL    string `json:"url,omitempty"`
	Head   string `json:"head,omitempty"`
	Base   string `json:"base,omitempty"`
	State  string `json:"state,omitempty"`
}

func newPRFindCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	var stateStr string
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "find <ident>",
		Short: "Find a PR whose title matches ident (e.g. [PLAN-00002-3])",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			state, err := pr.ParseState(stateStr)
			if err != nil {
				return &CLIError{Code: 1, Msg: err.Error()}
			}
			return runPRFind(stdout, stderr, cfg, args[0], state, jsonOut, runner)
		},
	}
	cmd.Flags().StringVar(&stateStr, "state", "any", "Filter by PR state: open, merged, or any")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON instead of plain text")
	return cmd
}

// runPRFind finds a PR and prints the result. jsonOut selects JSON output
// (--json); by default, output is plain text: one key:value line per
// populated field, in struct-field order. found is always printed, even when
// false. url prints like any other populated field — no exceptions.
func runPRFind(stdout, stderr io.Writer, cfg *config.Config, ident string, state pr.State, jsonOut bool, runner exec.Runner) error {
	result, err := pr.Find(context.Background(), runner, cfg, ident, state)
	if err != nil {
		return &CLIError{Code: 1, Msg: err.Error()}
	}
	out := prFindOutput{
		Found:  result.Found,
		Number: result.Number,
		Title:  result.Title,
		URL:    result.URL,
		Head:   result.Head,
		Base:   result.Base,
		State:  result.State,
	}

	if !jsonOut {
		writeKV(stdout, []kv{
			{Key: "found", Value: strconv.FormatBool(out.Found)},
			{Key: "number", Value: strconv.Itoa(out.Number), Omit: out.Number == 0},
			{Key: "title", Value: out.Title, Omit: out.Title == ""},
			{Key: "url", Value: out.URL, Omit: out.URL == ""},
			{Key: "head", Value: out.Head, Omit: out.Head == ""},
			{Key: "base", Value: out.Base, Omit: out.Base == ""},
			{Key: "state", Value: out.State, Omit: out.State == ""},
		})
		return nil
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// prBaseOutput is the JSON shape for pr base.
type prBaseOutput struct {
	Base string `json:"base"`
}

func newPRBaseCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "base <branch>",
		Short: "Resolve the PR base branch for a plan branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPRBase(stdout, stderr, args[0], jsonOut, runner)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON instead of plain text")
	return cmd
}

// runPRBase resolves the PR base branch and prints it. jsonOut selects JSON
// output (--json); by default, the single field collapses to a bare value
// with no label, matching the board status/config get precedent. runner is
// unused today — pr.Base is pure string logic with no gh/git calls — but is
// threaded through for consistency with the other three Component 4
// extractions (board list, pr find, issue list).
func runPRBase(stdout, stderr io.Writer, branch string, jsonOut bool, runner exec.Runner) error {
	base, err := pr.Base(branch)
	if err != nil {
		return err // already CLIError{Code:1}
	}

	if !jsonOut {
		fmt.Fprintln(stdout, base)
		return nil
	}

	out := prBaseOutput{Base: base}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// ── findings ─────────────────────────────────────────────────────────────────

// newFindingsCmd returns the findings subcommand group.
func newFindingsCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	findingsCmd := &cobra.Command{
		Use:   "findings",
		Short: "Manage plan-scoped JSON findings cache",
	}
	findingsCmd.AddCommand(newFindingsCacheCmd(stdout, stderr, runner))
	findingsCmd.AddCommand(newFindingsClearCmd(stdout, stderr, runner))
	return findingsCmd
}

func newFindingsCacheCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "cache <plan-number> <kind> <name>",
		Short: "Validate and write JSON from stdin to the findings cache",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return &CLIError{Code: 1, Msg: fmt.Sprintf("reading stdin: %v", err)}
			}
			if err := findings.Cache(context.Background(), runner, args[0], args[1], args[2], payload); err != nil {
				return &CLIError{Code: 1, Msg: err.Error()}
			}
			path := findings.CachePath(args[0], args[1], args[2])
			fmt.Fprintln(stdout, path)
			return nil
		},
	}
}

func newFindingsClearCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <plan-number>",
		Short: "Remove all cached findings for a plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := findings.Clear(context.Background(), runner, args[0]); err != nil {
				return &CLIError{Code: 1, Msg: err.Error()}
			}
			return nil
		},
	}
}

// ── worktree ─────────────────────────────────────────────────────────────────

// newWorktreeCmd returns the worktree subcommand group.
func newWorktreeCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	wtCmd := &cobra.Command{
		Use:   "worktree",
		Short: "Manage git worktrees for plan phases",
	}
	wtCmd.AddCommand(newWorktreeAddCmd(stdout, stderr, runner))
	wtCmd.AddCommand(newWorktreeRemoveCmd(stdout, stderr, runner))
	wtCmd.AddCommand(newWorktreePruneCmd(stdout, stderr, runner))
	wtCmd.AddCommand(newWorktreeResolveCmd(stdout, stderr, runner))
	return wtCmd
}

func newWorktreeAddCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "add <path> <branch> <base-branch>",
		Short: "Create a git worktree, handling all branch-state cases",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := worktree.Add(context.Background(), runner, args[0], args[1], args[2]); err != nil {
				return err // already CLIError or wrapped error
			}
			return nil
		},
	}
}

func newWorktreeRemoveCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <path>",
		Short: "Remove a git worktree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := worktree.Remove(context.Background(), runner, args[0]); err != nil {
				return &CLIError{Code: 1, Msg: err.Error()}
			}
			return nil
		},
	}
}

func newWorktreePruneCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "Prune stale git worktree administrative files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := worktree.Prune(context.Background(), runner); err != nil {
				return &CLIError{Code: 1, Msg: err.Error()}
			}
			return nil
		},
	}
}

func newWorktreeResolveCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <path>",
		Short: "Print a repo-relative path as absolute; rejects absolute input, path traversal, or paths outside this process's cwd/descendants",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := worktree.Resolve(context.Background(), runner, args[0])
			if err != nil {
				return &CLIError{Code: 1, Msg: err.Error()}
			}
			fmt.Fprintln(stdout, resolved)
			return nil
		},
	}
}

// ── issue ─────────────────────────────────────────────────────────────────────

// newIssueCmd returns the issue subcommand group.
func newIssueCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	issueCmd := &cobra.Command{
		Use:   "issue",
		Short: "Read GitHub issues",
	}
	issueCmd.AddCommand(newIssueListCmd(stdout, stderr, runner))
	issueCmd.AddCommand(newIssueTreeCmd(stdout, stderr, runner))
	issueCmd.AddCommand(newIssueLinkParentCmd(stdout, stderr, runner))
	issueCmd.AddCommand(newIssueCreateCmd(stdout, stderr, runner))
	issueCmd.AddCommand(newIssueEditCmd(stdout, stderr, runner))
	return issueCmd
}

// issueCreateOutput is the JSON shape for issue create.
type issueCreateOutput struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// issueCreateFlags is the state of issue create's flags, including which were
// passed at all (a flag passed with an empty value is not the same as absent).
type issueCreateFlags struct {
	title, body, bodyFile, labels, tmplType, draft string
	titleSet, tmplSet, draftSet, bodyFileSet       bool
	jsonOut                                        bool
}

func newIssueCreateCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	var f issueCreateFlags

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a GitHub issue (--title with --body/--body-file, or --template with --draft/--body-file to check it first)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			f.titleSet = cmd.Flags().Changed("title")
			f.tmplSet = cmd.Flags().Changed("template")
			f.draftSet = cmd.Flags().Changed("draft")
			f.bodyFileSet = cmd.Flags().Changed("body-file")
			return runIssueCreate(stdout, stderr, cfg, f, runner)
		},
	}
	cmd.Flags().StringVar(&f.tmplType, "template", "", "Check the body against this plan schema (e.g. impl-phase) before creating; exit 3 if it does not conform")
	cmd.Flags().StringVar(&f.draft, "draft", "", "YAML draft file (with --template); the title comes from the draft")
	cmd.Flags().StringVar(&f.title, "title", "", "Issue title (required unless --template --draft, where it must equal the draft's title)")
	cmd.Flags().StringVar(&f.body, "body", "", "Issue body (mutually exclusive with --body-file)")
	cmd.Flags().StringVar(&f.bodyFile, "body-file", "", "Path to a file containing the issue body (mutually exclusive with --body)")
	cmd.Flags().StringVar(&f.labels, "label", "", "Comma-separated label names")
	cmd.Flags().BoolVar(&f.jsonOut, "json", false, "Output JSON instead of plain text")
	return cmd
}

// runIssueCreate validates the flags, resolves the body (reading --body-file's
// contents when given), creates the issue, and prints the result. jsonOut
// selects JSON output (--json); by default, plain-text mode prints the issue
// number on one line then the URL on the next.
//
// With --template <type> the input (--draft or --body-file) is checked against
// that schema first, and the checked title and body are what gets created.
// Every usage problem found in the invocation is reported together in one
// exit-1 error, and when the input could be loaded its conformance findings
// are printed with them. Otherwise a non-conforming input exits 3 with the
// findings on stderr. Either way no gh call is made. Warnings and the schema
// line go to stderr.
func runIssueCreate(stdout, stderr io.Writer, cfg *config.Config, f issueCreateFlags, runner exec.Runner) error {
	const prefix = "issue create: "
	var p problems
	tmplGiven := f.tmplSet || f.tmplType != ""

	var sc *tmplpkg.Schema
	var origin string
	if tmplGiven {
		if f.tmplType == "" {
			p.add(emptyTemplateMsg(prefix))
		} else if s, o, err := loadSchema(f.tmplType); err != nil {
			p.add(errMsg(err))
		} else {
			sc, origin = s, o
		}
	}
	if f.draftSet && f.draft == "" {
		p.add(emptyFlagMsg(prefix, "draft"))
	}
	if f.bodyFileSet && f.bodyFile == "" {
		p.add(emptyFlagMsg(prefix, "body-file"))
	}
	if f.draft != "" && f.bodyFile != "" {
		p.add(prefix + "--draft and --body-file are mutually exclusive: " + exactlyOneMsg)
	}
	if tmplGiven {
		if f.body != "" {
			p.add(prefix + "--template does not accept an inline --body: use --draft <file.yml> or --body-file <file.md>")
		}
		if !f.draftSet && !f.bodyFileSet {
			p.add(prefix + "--template needs an input: " + exactlyOneMsg)
		}
		if f.draft == "" && f.bodyFile != "" && f.title == "" {
			p.add(prefix + "--template with --body-file requires --title (with --draft the title comes from the draft)")
		}
	} else {
		if f.draft != "" {
			p.add(prefix + "--draft requires --template <type>: pass --template, or use --title with --body/--body-file")
		}
		if f.title == "" {
			p.add("issue create requires --title")
		}
		if f.body != "" && f.bodyFile != "" {
			p.add(prefix + "--body and --body-file are mutually exclusive")
		}
		if f.body == "" && f.bodyFile == "" && !f.bodyFileSet {
			p.add("issue create requires --body or --body-file")
		}
	}

	// Check the input whenever it can be loaded, even with usage problems
	// already found, so they are all reported in one run.
	var checkedBody, checkedTitle string
	var res tmplpkg.Result
	checked := false
	if sc != nil && (f.draft != "") != (f.bodyFile != "") {
		body, title, r, err := checkInput(sc, origin, f.draft, f.bodyFile, f.title, false)
		if err != nil {
			p.add(errMsg(err))
		} else {
			checkedBody, checkedTitle, res, checked = body, title, r, true
		}
	}
	// A loaded draft's title is compared whether or not the draft conforms.
	if checked && f.draft != "" && f.titleSet && checkedTitle != "" && f.title != checkedTitle {
		p.add(fmt.Sprintf("%s--title %q does not match the draft's title %q: the title comes from the draft, so remove --title or make it equal to the draft's \"title:\"", prefix, f.title, checkedTitle))
	}
	if checked {
		fmt.Fprint(stderr, res.Format())
	}
	if err := p.err(); err != nil {
		return err
	}
	if res.HasErrors() {
		return clierr.NonConforming(fmt.Sprintf("the issue does not conform to the %s schema; nothing was created. Fix the findings above and run again", f.tmplType))
	}

	title, body, bodyFile := f.title, f.body, f.bodyFile
	if checked {
		// Send exactly what was checked, so a body file that changes after the
		// check cannot be created unchecked.
		title, body, bodyFile = checkedTitle, checkedBody, ""
	}
	resolvedBody := body
	if bodyFile != "" {
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return &CLIError{Code: 1, Msg: fmt.Sprintf("issue create: reading --body-file %q: %v", bodyFile, err)}
		}
		resolvedBody = string(data)
	}

	var labels []string
	if f.labels != "" {
		labels = strings.Split(f.labels, ",")
	}

	number, url, err := issue.Create(context.Background(), runner, cfg, title, resolvedBody, labels)
	if err != nil {
		return &CLIError{Code: 1, Msg: err.Error()}
	}

	if !f.jsonOut {
		fmt.Fprintln(stdout, number)
		fmt.Fprintln(stdout, url)
		return nil
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(issueCreateOutput{Number: number, URL: url})
}

// issueEditFlags is the state of issue edit's flags, including which were
// passed at all.
type issueEditFlags struct {
	tmplType, draft, bodyFile      string
	tmplSet, draftSet, bodyFileSet bool
}

func newIssueEditCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	var f issueEditFlags

	cmd := &cobra.Command{
		Use:   "edit <issue-number>",
		Short: "Replace an issue's body after checking it against a plan schema (--template, with --draft or --body-file)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			f.tmplSet = cmd.Flags().Changed("template")
			f.draftSet = cmd.Flags().Changed("draft")
			f.bodyFileSet = cmd.Flags().Changed("body-file")
			return runIssueEdit(stderr, cfg, args[0], f, runner)
		},
	}
	cmd.Flags().StringVar(&f.tmplType, "template", "", "Plan schema to check against (required), e.g. impl-phase")
	cmd.Flags().StringVar(&f.draft, "draft", "", "YAML draft file: sets the rendered body and the title")
	cmd.Flags().StringVar(&f.bodyFile, "body-file", "", "Markdown body file: sets the body only, the title is unchanged")
	return cmd
}

// runIssueEdit checks the input against the schema and only then edits the
// issue. Every usage problem found is reported together in one exit-1 error,
// with the conformance findings when the input could be checked; a check
// error alone exits 3. Either way no mutating gh call is made. For
// --body-file the title is read from the issue (a non-mutating gh call) only
// when the flags themselves are fine.
func runIssueEdit(stderr io.Writer, cfg *config.Config, numberArg string, f issueEditFlags, runner exec.Runner) error {
	const prefix = "issue edit: "
	var p problems

	number, numErr := strconv.Atoi(numberArg)
	if numErr != nil {
		p.add(fmt.Sprintf("invalid issue number %q: %v", numberArg, numErr))
	}
	var sc *tmplpkg.Schema
	var origin string
	switch {
	case f.tmplSet && f.tmplType == "":
		p.add(emptyTemplateMsg(prefix))
	case f.tmplType == "":
		p.add("issue edit requires --template <type>: it checks the new body against that plan schema")
	default:
		if s, o, err := loadSchema(f.tmplType); err != nil {
			p.add(errMsg(err))
		} else {
			sc, origin = s, o
		}
	}
	if f.draftSet && f.draft == "" {
		p.add(emptyFlagMsg(prefix, "draft"))
	}
	if f.bodyFileSet && f.bodyFile == "" {
		p.add(emptyFlagMsg(prefix, "body-file"))
	}
	if f.draft != "" && f.bodyFile != "" {
		p.add(prefix + "--draft and --body-file are mutually exclusive: " + exactlyOneMsg)
	}
	if !f.draftSet && !f.bodyFileSet {
		p.add(prefix + exactlyOneMsg)
	}

	var body, title string
	var res tmplpkg.Result
	checked := false
	if sc != nil && (f.draft != "") != (f.bodyFile != "") {
		var currentTitle string
		proceed := true
		if f.bodyFile != "" {
			// Read the file before spending a network call on the title; checkInput
			// reads (and checks) it again below, and that read is the one sent.
			if _, readErr := os.ReadFile(f.bodyFile); readErr != nil {
				p.add(fmt.Sprintf("%sreading --body-file %q: %v", prefix, f.bodyFile, readErr))
				proceed = false
			} else if len(p) == 0 {
				t, err := issue.Title(context.Background(), runner, cfg, number)
				if err != nil {
					p.add(prefix + err.Error())
					proceed = false
				}
				currentTitle = t
			} else {
				// Other problems exist: do not call gh, and so cannot check a
				// body against the issue's title.
				proceed = false
			}
		}
		if proceed {
			b, t, r, err := checkInput(sc, origin, f.draft, f.bodyFile, currentTitle, false)
			if err != nil {
				p.add(errMsg(err))
			} else {
				body, title, res, checked = b, t, r, true
			}
		}
	}
	if checked {
		fmt.Fprint(stderr, res.Format())
	}
	if err := p.err(); err != nil {
		return err
	}
	if res.HasErrors() {
		return clierr.NonConforming(fmt.Sprintf("the input does not conform to the %s schema; issue #%d was not changed. Fix the findings above and run again", f.tmplType, number))
	}
	if f.bodyFile != "" {
		title = "" // a markdown body leaves the title unchanged
	}
	if err := issue.Edit(context.Background(), runner, cfg, number, title, body); err != nil {
		return &CLIError{Code: 1, Msg: err.Error()}
	}
	return nil
}

func newIssueListCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	var labelFilter, titlePrefix, state string
	var limit int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues as plain text (--json for JSON), with optional label, title-prefix, and state filters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			return runIssueList(stdout, stderr, cfg, labelFilter, titlePrefix, state, limit, jsonOut, runner)
		},
	}
	cmd.Flags().StringVar(&labelFilter, "label", "", "Filter by label name")
	cmd.Flags().StringVar(&titlePrefix, "title-prefix", "", "Filter by title prefix (e.g. \"[PLAN-00002]\")")
	cmd.Flags().StringVar(&state, "state", "open", "Issue state: open, closed, or all")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of issues to fetch")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON instead of plain text")
	return cmd
}

// runIssueList lists issues. jsonOut selects JSON output (--json); by
// default, output is pipe-delimited plain text with no header row:
// "<number> | <state> | <labels-comma-joined> | <title>", one line per
// issue, title last (safe against a literal "|" in a title).
func runIssueList(stdout, stderr io.Writer, cfg *config.Config, labelFilter, titlePrefix, state string, limit int, jsonOut bool, runner exec.Runner) error {
	items, err := issue.List(context.Background(), runner, cfg, labelFilter, titlePrefix, state, limit)
	if err != nil {
		return &CLIError{Code: 1, Msg: err.Error()}
	}
	if items == nil {
		items = []issue.Item{}
	}

	if !jsonOut {
		for _, item := range items {
			fmt.Fprintf(stdout, "%d | %s | %s | %s\n", item.Number, item.State, strings.Join(item.Labels, ","), item.Title)
		}
		return nil
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(items)
}

func newIssueTreeCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "tree <issue-number>",
		Short: "Resolve the full arch/impl/phase hierarchy for a plan issue via GitHub sub-issue links",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return &CLIError{Code: 1, Msg: fmt.Sprintf("invalid issue number %q: %v", args[0], err)}
			}
			return runIssueTree(stdout, stderr, cfg, n, jsonOut, runner)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON instead of plain text")
	return cmd
}

// runIssueTree resolves and prints the plan hierarchy for an issue. jsonOut
// selects JSON output (--json); by default, output is pipe-delimited plain
// text with no header row: "<kind> | #<number> | <state> | <status> |
// <title>", one line per node, title last (safe against a literal "|" in a
// title). Output is empty only when the target issue is not a plan issue at
// all — a plan issue with no sub-issue links still emits its own single node.
// The not-a-plan-issue case also writes a stderr diagnostic so a caller that
// passed the wrong issue number has a signal beyond empty stdout; exit code
// remains 0 since this is not an operational error.
func runIssueTree(stdout, stderr io.Writer, cfg *config.Config, number int, jsonOut bool, runner exec.Runner) error {
	nodes, err := issue.Tree(context.Background(), runner, cfg, cfg.Repo, number)
	if err != nil {
		return &CLIError{Code: 1, Msg: err.Error()}
	}
	if nodes == nil {
		nodes = []issue.Node{}
	}
	if len(nodes) == 0 {
		fmt.Fprintf(stderr, "issue tree: #%d is not a plan issue (title does not parse as a PLAN-XXXXX identifier)\n", number)
	}

	if !jsonOut {
		for _, n := range nodes {
			fmt.Fprintf(stdout, "%s | #%d | %s | %s | %s\n", n.Kind, n.Number, n.State, n.Status, n.Title)
		}
		return nil
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(nodes)
}

func newIssueLinkParentCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "link-parent <child-issue-number> <parent-issue-number>",
		Short: "Set a GitHub-native sub-issue link between two issues",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			childNum, err := strconv.Atoi(args[0])
			if err != nil {
				return &CLIError{Code: 1, Msg: fmt.Sprintf("invalid child issue number %q: %v", args[0], err)}
			}
			parentNum, err := strconv.Atoi(args[1])
			if err != nil {
				return &CLIError{Code: 1, Msg: fmt.Sprintf("invalid parent issue number %q: %v", args[1], err)}
			}
			return runIssueLinkParent(cfg, childNum, parentNum, runner)
		},
	}
}

// runIssueLinkParent resolves both issue numbers to GraphQL node IDs (child
// first, then parent) and links them as a native GitHub sub-issue
// relationship. Silent on success (no stdout), matching board move / label
// ensure's convention; non-zero exit with a stderr message on failure.
func runIssueLinkParent(cfg *config.Config, childNum, parentNum int, runner exec.Runner) error {
	ctx := context.Background()
	client := gh.New(runner)

	_, childNodeID, err := client.IssueRef(ctx, cfg.Repo, childNum)
	if err != nil {
		return &CLIError{Code: 1, Msg: fmt.Sprintf("resolving child issue #%d: %v", childNum, err)}
	}
	_, parentNodeID, err := client.IssueRef(ctx, cfg.Repo, parentNum)
	if err != nil {
		return &CLIError{Code: 1, Msg: fmt.Sprintf("resolving parent issue #%d: %v", parentNum, err)}
	}
	if err := client.AddSubIssue(ctx, parentNodeID, childNodeID); err != nil {
		return &CLIError{Code: 1, Msg: fmt.Sprintf("linking #%d as sub-issue of #%d: %v", childNum, parentNum, err)}
	}
	return nil
}
