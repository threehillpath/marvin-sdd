// Package cli builds the Cobra command tree and maps CLIError exit codes to os.Exit.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"threehillpath.com/marvin-sdd/tool/internal/exec"
)

// NewRootCmd constructs the root Cobra command with all subcommand groups registered.
// stdin, stdout, and stderr are injectable for testing, as is runner — the exec.Runner
// used by every subcommand that shells out to gh or git.
func NewRootCmd(stdin io.Reader, stdout, stderr io.Writer, runner exec.Runner) *cobra.Command {
	root := &cobra.Command{
		Use:   "marvin",
		Short: "marvin — plan-workflow deterministic CLI tool",
		Long: `marvin encapsulates the deterministic shell operations used by
plan-workflow skills, providing a stable, testable interface for board
management, name derivation, config access, and plan-template rendering.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)

	// Register subcommand groups.
	root.AddCommand(newVersionCmd(stdout))
	root.AddCommand(newConfigCmd(stdout, stderr))
	root.AddCommand(newNamesCmd(stdout, stderr))
	root.AddCommand(newParseCmd(stdin, stdout, stderr))
	root.AddCommand(newTemplateCmd(stdout, stderr))
	root.AddCommand(newBoardCmd(stdout, stderr, runner))
	root.AddCommand(newLabelCmd(stdout, stderr, runner))
	root.AddCommand(newPRCmd(stdout, stderr, runner))
	root.AddCommand(newFindingsCmd(stdout, stderr))
	root.AddCommand(newWorktreeCmd(stdout, stderr, runner))
	root.AddCommand(newIssueCmd(stdout, stderr, runner))

	return root
}

// Execute runs the root command with os.Stdin/Stdout/Stderr and exits with the
// appropriate code. This is the only function that calls os.Exit.
func Execute() {
	code := RunWithStreams(os.Stdout, os.Stderr, func() error {
		root := NewRootCmd(os.Stdin, os.Stdout, os.Stderr, exec.OSRunner{})
		return root.Execute()
	})
	if code != 0 {
		os.Exit(code)
	}
}

// versionString is the single source of truth for marvin's version, referenced
// by both the plain-text banner and the --json output so the two can never drift.
const versionString = "v0.1.0"

// versionOutput is the --json payload shape for the version subcommand.
type versionOutput struct {
	Version string `json:"version"`
}

// newVersionCmd returns the version subcommand.
func newVersionCmd(stdout io.Writer) *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print marvin version",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !jsonOut {
				fmt.Fprintln(stdout, "marvin "+versionString+" (plan-workflow deterministic CLI)")
				return nil
			}

			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(versionOutput{Version: versionString})
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON instead of plain text")

	return cmd
}

// newConfigCmd returns the config subcommand group.
func newConfigCmd(stdout, stderr io.Writer) *cobra.Command {
	config := &cobra.Command{
		Use:   "config",
		Short: "Access plan-workflow configuration",
	}

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Print a single config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigGet(stdout, stderr, args[0])
		},
	}

	config.AddCommand(getCmd)
	return config
}

// newNamesCmd returns the names subcommand group.
func newNamesCmd(stdout, stderr io.Writer) *cobra.Command {
	names := &cobra.Command{
		Use:   "names",
		Short: "Derive canonical plan names",
	}

	var typ, suffix, worktreeBase string
	var phase int
	var jsonOut bool

	deriveCmd := &cobra.Command{
		Use:   "derive <issue>",
		Short: "Derive all names for an issue number",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNamesDerive(stdout, stderr, args[0], typ, suffix, worktreeBase, phase, jsonOut)
		},
	}
	deriveCmd.Flags().StringVar(&typ, "type", "feature", "Branch type: feature or bug")
	deriveCmd.Flags().StringVar(&suffix, "suffix", "", "Multi-impl suffix (e.g. a, b)")
	deriveCmd.Flags().IntVar(&phase, "phase", 0, "Phase number (0 = no phase)")
	deriveCmd.Flags().StringVar(&worktreeBase, "worktree-base", "", "Worktree base directory (overrides config value; required when --phase is set and no config is present)")
	deriveCmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON instead of plain text")

	names.AddCommand(deriveCmd)
	return names
}

// newParseCmd returns the parse subcommand group. stdin is the injected reader
// for `parse phase-list`, which reads a comment body from stdin.
func newParseCmd(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	parse := &cobra.Command{
		Use:   "parse",
		Short: "Parse plan identifiers from text",
	}

	var titleJSON bool
	titleCmd := &cobra.Command{
		Use:   "title <title>",
		Short: "Extract plan ident from an issue title",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runParseTitle(stdout, stderr, args[0], titleJSON)
		},
	}
	titleCmd.Flags().BoolVar(&titleJSON, "json", false, "Output JSON instead of plain text")
	parse.AddCommand(titleCmd)

	var phaseListJSON bool
	phaseListCmd := &cobra.Command{
		Use:   "phase-list",
		Short: "Extract phase issue numbers from a Phases created: comment (stdin)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runParsePhaseList(stdin, stdout, stderr, phaseListJSON)
		},
	}
	phaseListCmd.Flags().BoolVar(&phaseListJSON, "json", false, "Output JSON instead of plain text")
	parse.AddCommand(phaseListCmd)

	return parse
}

// newTemplateCmd returns the template subcommand group.
func newTemplateCmd(stdout, stderr io.Writer) *cobra.Command {
	tmpl := &cobra.Command{
		Use:   "template",
		Short: "Render plan issue templates",
	}

	var metaFile, sectionsFile string
	var skeleton bool

	renderCmd := &cobra.Command{
		Use:   "render <arch-plan|impl-plan|impl-phase>",
		Short: "Render a plan template from schema",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTemplateRender(stdout, stderr, args[0], metaFile, sectionsFile, skeleton)
		},
	}
	renderCmd.Flags().StringVar(&metaFile, "meta", "", "Path to meta JSON file")
	renderCmd.Flags().StringVar(&sectionsFile, "sections", "", "Path to sections JSON file")
	renderCmd.Flags().BoolVar(&skeleton, "skeleton", false, "Output empty section headings without requiring content")

	tmpl.AddCommand(renderCmd)
	return tmpl
}
