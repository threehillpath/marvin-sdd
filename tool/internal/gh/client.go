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

// ProjectItemContent holds the linked issue details within a project board item.
type ProjectItemContent struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

// ProjectItem represents a single item from the GitHub Projects v2 board.
type ProjectItem struct {
	ID      string             `json:"id"`
	Title   string             `json:"title"`
	Status  string             `json:"status"`
	Content ProjectItemContent `json:"content"`
}

// projectItemListResponse is the JSON shape returned by gh project item-list.
type projectItemListResponse struct {
	Items []ProjectItem `json:"items"`
}

// ProjectItemList runs `gh project item-list` and returns all parsed items.
// limit caps the number of items fetched; 0 defaults to 100.
func (c *Client) ProjectItemList(ctx context.Context, projectNumber int, owner string, limit int) ([]ProjectItem, error) {
	if limit <= 0 {
		limit = 100
	}
	args := []string{
		"project", "item-list",
		fmt.Sprintf("%d", projectNumber),
		"--owner", owner,
		"--format", "json",
		"--limit", fmt.Sprintf("%d", limit),
	}
	stdout, stderr, code, err := c.runner.Run(ctx, "gh", args...)
	if err != nil {
		return nil, fmt.Errorf("gh project item-list: %w", err)
	}
	if code != 0 {
		return nil, fmt.Errorf("gh project item-list exited %d: %s", code, stderr)
	}
	var resp projectItemListResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return nil, fmt.Errorf("gh project item-list: parse response: %w", err)
	}
	return resp.Items, nil
}

// IssueListItem represents a single issue from gh issue list output.
type IssueListItem struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	State  string   `json:"state"`
	Labels []ghLabel `json:"labels"`
}

// ghLabel is the label shape returned by gh issue list --json labels.
type ghLabel struct {
	Name string `json:"name"`
}

// IssueList runs `gh issue list --repo <repo>` with optional label and state filters
// and returns the parsed items. label may be empty to skip label filtering.
// state must be "open", "closed", or "all"; empty defaults to "open".
// limit caps results; 0 defaults to 100.
func (c *Client) IssueList(ctx context.Context, repo, label, state string, limit int) ([]IssueListItem, error) {
	if limit <= 0 {
		limit = 100
	}
	if state == "" {
		state = "open"
	}
	args := []string{
		"issue", "list",
		"--repo", repo,
		"--state", state,
		"--json", "number,title,state,labels",
		"--limit", fmt.Sprintf("%d", limit),
	}
	if label != "" {
		args = append(args, "--label", label)
	}
	stdout, stderr, code, err := c.runner.Run(ctx, "gh", args...)
	if err != nil {
		return nil, fmt.Errorf("gh issue list: %w", err)
	}
	if code != 0 {
		return nil, fmt.Errorf("gh issue list exited %d: %s", code, stderr)
	}
	var items []IssueListItem
	if err := json.Unmarshal(stdout, &items); err != nil {
		return nil, fmt.Errorf("gh issue list: parse response: %w", err)
	}
	return items, nil
}
