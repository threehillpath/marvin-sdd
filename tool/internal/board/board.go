// Package board provides operations for managing GitHub Projects v2 board state.
// All operations are mediated through an exec.Runner so they can be tested without
// real network calls.
package board

import (
	"context"
	"fmt"

	"threehillpath.com/claude-plan-workflow/tool/internal/config"
	"threehillpath.com/claude-plan-workflow/tool/internal/exec"
	"threehillpath.com/claude-plan-workflow/tool/internal/gh"
)

// issueURL builds the canonical GitHub issue URL from the config repo and issue number.
func issueURL(cfg *config.Config, issueNumber int) string {
	return fmt.Sprintf("https://github.com/%s/issues/%d", cfg.Repo, issueNumber)
}

// AddItem adds an issue to a GitHub Projects v2 board and returns the item ID.
func AddItem(ctx context.Context, runner exec.Runner, cfg *config.Config, issueNumber int) (string, error) {
	client := gh.New(runner)
	return client.ProjectItemAdd(ctx, cfg.ProjectNumber, cfg.Owner(), issueURL(cfg, issueNumber))
}

// SetStatus sets the board status field for an item to a named status.
// If the status maps to "n/a" it is a successful no-op.
// Returns an error for unknown status names.
func SetStatus(ctx context.Context, runner exec.Runner, cfg *config.Config, itemID, status string) error {
	optionID, present, err := cfg.StatusOptionID(status)
	if err != nil {
		return fmt.Errorf("board set-status: %w", err)
	}
	if !present {
		// n/a column — no-op
		return nil
	}
	client := gh.New(runner)
	return client.ProjectItemEdit(ctx, cfg.ProjectID, itemID, cfg.StatusFieldID, optionID)
}

// Move adds an issue to the board (if not already present), sets its status to the
// named status, and syncs the GitHub issue open/closed state:
//   - status "done" → close the issue
//   - any other status → reopen the issue
//
// If the status maps to "n/a", the entire operation is a no-op.
func Move(ctx context.Context, runner exec.Runner, cfg *config.Config, issueNumber int, status string) error {
	// Check if n/a before doing any work.
	_, present, err := cfg.StatusOptionID(status)
	if err != nil {
		return fmt.Errorf("board move: %w", err)
	}
	if !present {
		// n/a — no-op
		return nil
	}

	// 1. Add to project (idempotent by gh CLI).
	itemID, err := AddItem(ctx, runner, cfg, issueNumber)
	if err != nil {
		return fmt.Errorf("board move: add item: %w", err)
	}

	// 2. Set status.
	if err := SetStatus(ctx, runner, cfg, itemID, status); err != nil {
		return fmt.Errorf("board move: set status: %w", err)
	}

	// 3. Sync issue open/closed state.
	client := gh.New(runner)
	if status == "done" {
		if err := client.IssueClose(ctx, cfg.Repo, issueNumber); err != nil {
			return fmt.Errorf("board move: close issue: %w", err)
		}
	} else {
		if err := client.IssueReopen(ctx, cfg.Repo, issueNumber); err != nil {
			return fmt.Errorf("board move: reopen issue: %w", err)
		}
	}

	return nil
}
