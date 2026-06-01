// Package gh provides a typed wrapper around the gh CLI for GitHub API operations.
// All network I/O is mediated through an exec.Runner so callers can be tested
// with a fake runner and recorded fixtures.
package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"threehillpath.com/claude-plan-workflow/tool/internal/exec"
)

// Client wraps an exec.Runner and provides typed gh CLI operations.
type Client struct {
	runner exec.Runner
}

// New constructs a Client backed by the given runner.
func New(runner exec.Runner) *Client {
	return &Client{runner: runner}
}

// IssueJSON runs `gh issue view <issue> --json <fields>` and returns the raw JSON bytes.
func (c *Client) IssueJSON(ctx context.Context, issue int, fields ...string) ([]byte, error) {
	args := []string{"issue", "view", fmt.Sprintf("%d", issue), "--json", strings.Join(fields, ",")}
	stdout, stderr, code, err := c.runner.Run(ctx, "gh", args...)
	if err != nil {
		return nil, fmt.Errorf("gh issue view: %w", err)
	}
	if code != 0 {
		return nil, fmt.Errorf("gh issue view exited %d: %s", code, stderr)
	}
	return stdout, nil
}

// projectItemAddResponse is the JSON shape returned by gh project item-add.
type projectItemAddResponse struct {
	ID string `json:"id"`
}

// ProjectItemAdd runs `gh project item-add <projectNumber> --owner <owner> --url <url> --format json`
// and returns the item ID from the JSON response.
func (c *Client) ProjectItemAdd(ctx context.Context, projectNumber int, owner, url string) (string, error) {
	args := []string{
		"project", "item-add",
		fmt.Sprintf("%d", projectNumber),
		"--owner", owner,
		"--url", url,
		"--format", "json",
	}
	stdout, stderr, code, err := c.runner.Run(ctx, "gh", args...)
	if err != nil {
		return "", fmt.Errorf("gh project item-add: %w", err)
	}
	if code != 0 {
		return "", fmt.Errorf("gh project item-add exited %d: %s", code, stderr)
	}

	var resp projectItemAddResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return "", fmt.Errorf("gh project item-add: parse response: %w", err)
	}
	if resp.ID == "" {
		return "", fmt.Errorf("gh project item-add: response missing id field")
	}
	return resp.ID, nil
}

// ProjectItemEdit runs `gh project item-edit` to set a field value on a project item.
func (c *Client) ProjectItemEdit(ctx context.Context, projectID, itemID, fieldID, optionID string) error {
	args := []string{
		"project", "item-edit",
		"--project-id", projectID,
		"--id", itemID,
		"--field-id", fieldID,
		"--single-select-option-id", optionID,
	}
	_, stderr, code, err := c.runner.Run(ctx, "gh", args...)
	if err != nil {
		return fmt.Errorf("gh project item-edit: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("gh project item-edit exited %d: %s", code, stderr)
	}
	return nil
}

// IssueClose closes a GitHub issue by number.
func (c *Client) IssueClose(ctx context.Context, repo string, issue int) error {
	args := []string{"issue", "close", fmt.Sprintf("%d", issue), "--repo", repo}
	_, stderr, code, err := c.runner.Run(ctx, "gh", args...)
	if err != nil {
		return fmt.Errorf("gh issue close: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("gh issue close exited %d: %s", code, stderr)
	}
	return nil
}

// IssueReopen reopens a GitHub issue by number.
func (c *Client) IssueReopen(ctx context.Context, repo string, issue int) error {
	args := []string{"issue", "reopen", fmt.Sprintf("%d", issue), "--repo", repo}
	_, stderr, code, err := c.runner.Run(ctx, "gh", args...)
	if err != nil {
		return fmt.Errorf("gh issue reopen: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("gh issue reopen exited %d: %s", code, stderr)
	}
	return nil
}
