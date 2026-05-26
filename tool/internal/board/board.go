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

// SetStatus adds an issue to the board (if not already present) and sets its
// status field to the named status. If the status maps to "n/a" it is a
// successful no-op. Returns an error for unknown status names.
func SetStatus(ctx context.Context, runner exec.Runner, cfg *config.Config, issueNumber int, status string) error {
	optionID, present, err := cfg.StatusOptionID(status)
	if err != nil {
		return fmt.Errorf("board set-status: %w", err)
	}
	if !present {
		// n/a column — no-op
		return nil
	}
	itemID, err := AddItem(ctx, runner, cfg, issueNumber)
	if err != nil {
		return fmt.Errorf("board set-status: add item: %w", err)
	}
	client := gh.New(runner)
	return client.ProjectItemEdit(ctx, cfg.ProjectID, itemID, cfg.StatusFieldID, optionID)
}

// Move sets an issue's board status and syncs its GitHub open/closed state:
//   - status "done" → close the issue
//   - any other status → reopen the issue
//
// If the status maps to "n/a", the entire operation is a no-op.
func Move(ctx context.Context, runner exec.Runner, cfg *config.Config, issueNumber int, status string) error {
	// SetStatus handles the n/a check, add-to-board, and set-status.
	if err := SetStatus(ctx, runner, cfg, issueNumber, status); err != nil {
		return fmt.Errorf("board move: %w", err)
	}

	// Sync issue open/closed state (skipped when SetStatus was a no-op).
	_, present, _ := cfg.StatusOptionID(status)
	if !present {
		return nil
	}
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
