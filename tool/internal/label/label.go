// Package label provides idempotent GitHub label management.
// Ensure guarantees that a label with the given name exists; if it is already
// present (regardless of color or description), it is left unchanged.
package label

import (
	"context"
	"encoding/json"
	"fmt"

	"threehillpath.com/claude-plan-workflow/tool/internal/config"
	"threehillpath.com/claude-plan-workflow/tool/internal/exec"
)

// ghLabel is the JSON shape returned by gh label list.
type ghLabel struct {
	Name string `json:"name"`
}

// Ensure creates a label with the given name, description, and hex color if it
// does not exist. If it already exists, it is left unchanged (no edit).
// color should be a 6-digit hex string without the leading '#' (e.g. "0075ca").
func Ensure(ctx context.Context, runner exec.Runner, cfg *config.Config, name, description, color string) error {
	// 1. Check whether the label already exists.
	listArgs := []string{"label", "list", "--repo", cfg.Repo, "--json", "name", "--search", name}
	stdout, stderr, code, err := runner.Run(ctx, "gh", listArgs...)
	if err != nil {
		return fmt.Errorf("label ensure: list: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("label ensure: list exited %d: %s", code, stderr)
	}

	var labels []ghLabel
	if err := json.Unmarshal(stdout, &labels); err != nil {
		return fmt.Errorf("label ensure: parse list response: %w", err)
	}

	// Check for exact name match (gh --search may do substring matching).
	for _, l := range labels {
		if l.Name == name {
			// Already exists — no-op.
			return nil
		}
	}

	// 2. Create the label.
	createArgs := []string{
		"label", "create", name,
		"--repo", cfg.Repo,
		"--description", description,
		"--color", color,
	}
	_, stderr, code, err = runner.Run(ctx, "gh", createArgs...)
	if err != nil {
		return fmt.Errorf("label ensure: create: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("label ensure: create exited %d: %s", code, stderr)
	}
	return nil
}
