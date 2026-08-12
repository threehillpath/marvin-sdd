// Package config loads and exposes the plan-workflow configuration file.
// Primary format: .claude/plan-workflow-config.yml (YAML).
// Fallback format: .claude/plan-workflow-config.md (legacy markdown table).
// Discovery walks up from the given start directory to the filesystem root.
package config

import (
	"fmt"
	"strings"

	"threehillpath.com/marvin-sdd/tool/internal/clierr"
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
	// Required; Load returns CLIError{Code:2} when worktree_base / Worktree base is absent.
	WorktreeBase string
}

// StatusOptionID resolves a human status name to its board option ID.
// Returns present=false when the configured value is "n/a" (column absent).
// Returns an error for unknown status names.
// Normalizes hyphens and spaces to underscores so user-facing forms like
// "in-progress" and "in progress" resolve to the config key "in_progress".
func (c *Config) StatusOptionID(name string) (id string, present bool, err error) {
	normalized := strings.NewReplacer("-", "_", " ", "_").Replace(strings.ToLower(name))
	val, ok := c.Statuses[normalized]
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
