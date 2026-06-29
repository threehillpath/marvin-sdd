// Package cli builds the Cobra command tree and maps CLIError exit codes to os.Exit.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCmd constructs the root Cobra command with all subcommand groups registered.
// stdout and stderr are injectable for testing.
func NewRootCmd(stdout, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:   "marvin",
		Short: "marvin — plan-workflow deterministic CLI tool",
		Long: `marvin encapsulates the deterministic shell operations used by
plan-workflow skills, providing a stable, testable interface for board
management, name derivation, config access, and plan-template rendering.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.SetOut(stdout)
	root.SetErr(stderr)

	// Register subcommand groups.
	root.AddCommand(newVersionCmd(stdout))
	root.AddCommand(newConfigCmd(stdout, stderr))
	root.AddCommand(newNamesCmd(stdout, stderr))
	root.AddCommand(newParseCmd(stdout, stderr))
	root.AddCommand(newTemplateCmd(stdout, stderr))
	root.AddCommand(newBoardCmd(stdout, stderr))
	root.AddCommand(newLabelCmd(stdout, stderr))
	root.AddCommand(newPRCmd(stdout, stderr))
	root.AddCommand(newFindingsCmd(stdout, stderr))
	root.AddCommand(newWorktreeCmd(stdout, stderr))
	root.AddCommand(newIssueCmd(stdout, stderr))

	return root
}

// Execute runs the root command with os.Stdin/Stdout/Stderr and exits with the
// appropriate code. This is the only function that calls os.Exit.
func Execute() {
	code := RunWithStreams(os.Stdout, os.Stderr, func() error {
		root := NewRootCmd(os.Stdout, os.Stderr)
		return root.Execute()
	})
	if code != 0 {
		os.Exit(code)
	}
}

// newVersionCmd returns the version subcommand.
func newVersionCmd(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print marvin version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(stdout, "marvin v0.1.0 (plan-workflow deterministic CLI)")
			return nil
		},
	}
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

	var suffix, worktreeBase string
	var phase int

	deriveCmd := &cobra.Command{
		Use:   "derive <issue>",
		Short: "Derive all names for an issue number",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNamesDerive(stdout, stderr, args[0], suffix, worktreeBase, phase)
		},
	}
	deriveCmd.Flags().StringVar(&suffix, "suffix", "", "Multi-impl suffix (e.g. a, b)")
	deriveCmd.Flags().IntVar(&phase, "phase", 0, "Phase number (0 = no phase)")
	deriveCmd.Flags().StringVar(&worktreeBase, "worktree-base", "", "Worktree base directory (overrides config value; required when --phase is set and no config is present)")

	names.AddCommand(deriveCmd)
	return names
}

// newParseCmd returns the parse subcommand group.
func newParseCmd(stdout, stderr io.Writer) *cobra.Command {
	parse := &cobra.Command{
		Use:   "parse",
		Short: "Parse plan identifiers from text",
	}

	parse.AddCommand(&cobra.Command{
		Use:   "title <title>",
		Short: "Extract plan ident from an issue title",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runParseTitle(stdout, stderr, args[0])
		},
	})

	parse.AddCommand(&cobra.Command{
		Use:   "phase-list",
		Short: "Extract phase issue numbers from a Phases created: comment (stdin)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runParsePhaseList(stdout, stderr)
		},
	})

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
