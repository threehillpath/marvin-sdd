package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"threehillpath.com/claude-plan-workflow/tool/internal/board"
	"threehillpath.com/claude-plan-workflow/tool/internal/config"
	"threehillpath.com/claude-plan-workflow/tool/internal/exec"
	"threehillpath.com/claude-plan-workflow/tool/internal/findings"
	"threehillpath.com/claude-plan-workflow/tool/internal/issue"
	"threehillpath.com/claude-plan-workflow/tool/internal/label"
	"threehillpath.com/claude-plan-workflow/tool/internal/pr"
	"threehillpath.com/claude-plan-workflow/tool/internal/worktree"
)

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
		Short: "List board items as JSON, optionally filtered by status",
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

// runBoardList lists board items, encoding them as JSON. jsonOut is threaded
// through for the --json flag; plain-text-by-default output is added in a
// later phase, so output is unconditionally JSON today.
func runBoardList(stdout, stderr io.Writer, cfg *config.Config, statusFilter string, limit int, jsonOut bool, runner exec.Runner) error {
	items, err := board.List(context.Background(), runner, cfg, statusFilter, limit)
	if err != nil {
		return &CLIError{Code: 1, Msg: err.Error()}
	}
	if items == nil {
		items = []board.BoardItem{}
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

// runPRFind finds a PR and encodes the result as JSON. jsonOut is threaded
// through for the --json flag; plain-text-by-default output is added in a
// later phase, so output is unconditionally JSON today.
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

// runPRBase resolves the PR base branch and encodes the result as JSON. jsonOut
// is threaded through for the --json flag; plain-text-by-default output is
// added in a later phase, so output is unconditionally JSON today. runner is
// unused today — pr.Base is pure string logic with no gh/git calls — but is
// threaded through for consistency with the other three Component 4
// extractions (board list, pr find, issue list).
func runPRBase(stdout, stderr io.Writer, branch string, jsonOut bool, runner exec.Runner) error {
	base, err := pr.Base(branch)
	if err != nil {
		return err // already CLIError{Code:1}
	}
	out := prBaseOutput{Base: base}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// ── findings ─────────────────────────────────────────────────────────────────

// newFindingsCmd returns the findings subcommand group.
func newFindingsCmd(stdout, stderr io.Writer) *cobra.Command {
	findingsCmd := &cobra.Command{
		Use:   "findings",
		Short: "Manage plan-scoped JSON findings cache",
	}
	findingsCmd.AddCommand(newFindingsCacheCmd(stdout, stderr))
	findingsCmd.AddCommand(newFindingsClearCmd(stdout, stderr))
	return findingsCmd
}

func newFindingsCacheCmd(stdout, stderr io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "cache <plan-number> <kind> <name>",
		Short: "Validate and write JSON from stdin to the findings cache",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return &CLIError{Code: 1, Msg: fmt.Sprintf("reading stdin: %v", err)}
			}
			if err := findings.Cache(args[0], args[1], args[2], payload); err != nil {
				return &CLIError{Code: 1, Msg: err.Error()}
			}
			path := findings.CachePath(args[0], args[1], args[2])
			fmt.Fprintln(stdout, path)
			return nil
		},
	}
}

func newFindingsClearCmd(stdout, stderr io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "clear <plan-number>",
		Short: "Remove all cached findings for a plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := findings.Clear(args[0]); err != nil {
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

// ── issue ─────────────────────────────────────────────────────────────────────

// newIssueCmd returns the issue subcommand group.
func newIssueCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	issueCmd := &cobra.Command{
		Use:   "issue",
		Short: "Read GitHub issues",
	}
	issueCmd.AddCommand(newIssueListCmd(stdout, stderr, runner))
	return issueCmd
}

func newIssueListCmd(stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	var labelFilter, titlePrefix, state string
	var limit int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues as JSON, with optional label, title-prefix, and state filters",
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

// runIssueList lists issues, encoding them as JSON. jsonOut is threaded
// through for the --json flag; plain-text-by-default output is added in a
// later phase, so output is unconditionally JSON today.
func runIssueList(stdout, stderr io.Writer, cfg *config.Config, labelFilter, titlePrefix, state string, limit int, jsonOut bool, runner exec.Runner) error {
	items, err := issue.List(context.Background(), runner, cfg, labelFilter, titlePrefix, state, limit)
	if err != nil {
		return &CLIError{Code: 1, Msg: err.Error()}
	}
	if items == nil {
		items = []issue.Item{}
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(items)
}
