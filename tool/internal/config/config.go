// Package config loads and exposes the plan-workflow configuration file.
// Primary format: .claude/plan-workflow-config.yml (YAML).
// Fallback format: .claude/plan-workflow-config.md (legacy markdown table).
// Discovery walks up from the given start directory to the filesystem root.
package config

import (
	"fmt"
	"strings"

	"github.com/threehillpath/claude-plan-workflow/tool/internal/clierr"
)

// Config holds the parsed plan-workflow configuration.
type Config struct {
	Repo          string
	ProjectNumber int
	ProjectID     string
	StatusFieldID string
	// Statuses maps human status name (e.g. "in_progress") to option ID or "n/a".
	Statuses     map[string]string
	TestCommands map[string]string
	// WorktreeBase is the repo-relative directory under which phase worktrees are created.
	// Defaults to DefaultWorktreeBase when not set in the config file.
	WorktreeBase string
}

// StatusOptionID resolves a human status name to its board option ID.
// Returns present=false when the configured value is "n/a" (column absent).
// Returns an error for unknown status names.
func (c *Config) StatusOptionID(name string) (id string, present bool, err error) {
	val, ok := c.Statuses[name]
	if !ok {
		return "", false, fmt.Errorf("unknown status %q", name)
	}
	if val == "n/a" {
		return "", false, nil
	}
	return val, true, nil
}

// Owner returns the owner segment of the Repo field (everything before the first "/").
func (c *Config) Owner() string {
	parts := strings.SplitN(c.Repo, "/", 2)
	if len(parts) == 0 {
		return c.Repo
	}
	return parts[0]
}

// configMissingError returns a CLIError{Code:2} for missing config.
func configMissingError(startDir string) error {
	return clierr.ConfigMissing(startDir)
}

// configBadError returns a CLIError{Code:2} for a malformed config file.
func configBadError(path string, cause error) error {
	return clierr.ConfigBad(path, cause)
}
