// Package gh provides a typed wrapper around the gh CLI for GitHub API operations.
// All network I/O is mediated through an exec.Runner so callers can be tested
// with a fake runner and recorded fixtures.
package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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

// graphQLErrorEnvelope is the {data, errors} shape gh api graphql writes to
// stdout on a GraphQL-level failure (as opposed to a transport/auth failure,
// which never produces this shape).
type graphQLErrorEnvelope struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// GraphQL runs `gh api graphql -f query=<query>` plus one `-f <key>=<value>`
// per vars entry (sorted by key for deterministic argument order) and returns
// the raw stdout bytes.
//
// Verified live, `gh api graphql` exits non-zero whenever the response carries
// a GraphQL errors array — a genuine success never reaches the unmarshal path
// with errors populated. So on non-zero exit, GraphQL first attempts to parse
// stdout as the {data, errors} envelope; if that succeeds and errors is
// non-empty, the returned error is built from errors[].message (the specific
// GraphQL failure reason, which lives on stdout, not gh's abbreviated stderr
// summary). If stdout doesn't parse as that envelope — a non-GraphQL gh
// failure such as auth or network — GraphQL falls back to the existing
// wrap-non-zero-exit-with-stderr idiom unchanged.
func (c *Client) GraphQL(ctx context.Context, query string, vars map[string]string) ([]byte, error) {
	args := []string{"api", "graphql", "-f", "query=" + query}
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "-f", fmt.Sprintf("%s=%s", k, vars[k]))
	}

	stdout, stderr, code, err := c.runner.Run(ctx, "gh", args...)
	if err != nil {
		return nil, fmt.Errorf("gh api graphql: %w", err)
	}
	if code != 0 {
		var envelope graphQLErrorEnvelope
		if jsonErr := json.Unmarshal(stdout, &envelope); jsonErr == nil && len(envelope.Errors) > 0 {
			msgs := make([]string, len(envelope.Errors))
			for i, e := range envelope.Errors {
				msgs[i] = e.Message
			}
			return nil, fmt.Errorf("gh api graphql: %s", strings.Join(msgs, "; "))
		}
		return nil, fmt.Errorf("gh api graphql exited %d: %s", code, stderr)
	}
	return stdout, nil
}

// SubIssueRef is a lightweight reference to an issue reached via a sub-issue
// relationship (parent or child), as returned by SubIssues and ParentIssue.
type SubIssueRef struct {
	Number int
	Title  string
	State  string
}

// issueRefResponse is the JSON shape returned by
// gh issue view --json id,number,title,state.
type issueRefResponse struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
}

// IssueRef wraps `gh issue view <number> --repo <repo> --json id,number,title,state`,
// returning both the GraphQL node ID and the ref (number/title/state) in one
// call. AddSubIssue, SubIssues, and ParentIssue use this internally to resolve
// node IDs, since every caller that needs a node ID also has the issue number
// in hand and gets the ref for free.
func (c *Client) IssueRef(ctx context.Context, repo string, number int) (ref SubIssueRef, nodeID string, err error) {
	args := []string{"issue", "view", fmt.Sprintf("%d", number), "--repo", repo, "--json", "id,number,title,state"}
	stdout, stderr, code, runErr := c.runner.Run(ctx, "gh", args...)
	if runErr != nil {
		return SubIssueRef{}, "", fmt.Errorf("gh issue view: %w", runErr)
	}
	if code != 0 {
		return SubIssueRef{}, "", fmt.Errorf("gh issue view exited %d: %s", code, stderr)
	}
	var resp issueRefResponse
	if jsonErr := json.Unmarshal(stdout, &resp); jsonErr != nil {
		return SubIssueRef{}, "", fmt.Errorf("gh issue view: parse response: %w", jsonErr)
	}
	return SubIssueRef{Number: resp.Number, Title: resp.Title, State: resp.State}, resp.ID, nil
}

// addSubIssueMutation is GitHub's native sub-issue mutation. issueId is the
// parent's node ID; subIssueId is the child's node ID.
const addSubIssueMutation = `mutation($parentId: ID!, $childId: ID!) {
  addSubIssue(input: {issueId: $parentId, subIssueId: $childId}) {
    subIssue { id }
  }
}`

// AddSubIssue calls the addSubIssue GraphQL mutation, linking childNodeID as
// a native sub-issue of parentNodeID. Callers resolve both node IDs via
// IssueRef before calling this.
func (c *Client) AddSubIssue(ctx context.Context, parentNodeID, childNodeID string) error {
	_, err := c.GraphQL(ctx, addSubIssueMutation, map[string]string{
		"parentId": parentNodeID,
		"childId":  childNodeID,
	})
	return err
}

// subIssuesQuery inlines a fixed first: 50 rather than passing it through
// vars — GraphQL's vars map[string]string sends every value as a String via
// -f, which cannot express the Int type GraphQL's first argument requires.
// Results beyond 50 sub-issues are not paginated.
const subIssuesQuery = `query($id: ID!) {
  node(id: $id) {
    ... on Issue {
      subIssues(first: 50) {
        nodes { number title state }
      }
    }
  }
}`

// subIssuesResponse is the JSON shape returned by subIssuesQuery.
type subIssuesResponse struct {
	Data struct {
		Node struct {
			SubIssues struct {
				Nodes []struct {
					Number int    `json:"number"`
					Title  string `json:"title"`
					State  string `json:"state"`
				} `json:"nodes"`
			} `json:"subIssues"`
		} `json:"node"`
	} `json:"data"`
}

// SubIssues resolves number's GraphQL node ID via IssueRef, then queries its
// subIssues field and returns the linked sub-issues (native GitHub sub-issue
// relationships, not a title-string match).
func (c *Client) SubIssues(ctx context.Context, repo string, number int) ([]SubIssueRef, error) {
	_, nodeID, err := c.IssueRef(ctx, repo, number)
	if err != nil {
		return nil, err
	}
	stdout, err := c.GraphQL(ctx, subIssuesQuery, map[string]string{"id": nodeID})
	if err != nil {
		return nil, err
	}
	var resp subIssuesResponse
	if jsonErr := json.Unmarshal(stdout, &resp); jsonErr != nil {
		return nil, fmt.Errorf("gh api graphql: parse response: %w", jsonErr)
	}
	refs := make([]SubIssueRef, 0, len(resp.Data.Node.SubIssues.Nodes))
	for _, n := range resp.Data.Node.SubIssues.Nodes {
		refs = append(refs, SubIssueRef{Number: n.Number, Title: n.Title, State: n.State})
	}
	return refs, nil
}

// parentIssueQuery fetches the parent sub-issue relationship, if any.
const parentIssueQuery = `query($id: ID!) {
  node(id: $id) {
    ... on Issue {
      parent { number title state }
    }
  }
}`

// parentIssueResponse is the JSON shape returned by parentIssueQuery.
type parentIssueResponse struct {
	Data struct {
		Node struct {
			Parent *struct {
				Number int    `json:"number"`
				Title  string `json:"title"`
				State  string `json:"state"`
			} `json:"parent"`
		} `json:"node"`
	} `json:"data"`
}

// ParentIssue resolves number's GraphQL node ID via IssueRef, then queries
// its parent field. ok is false with no error when the issue has no parent.
func (c *Client) ParentIssue(ctx context.Context, repo string, number int) (SubIssueRef, bool, error) {
	_, nodeID, err := c.IssueRef(ctx, repo, number)
	if err != nil {
		return SubIssueRef{}, false, err
	}
	stdout, err := c.GraphQL(ctx, parentIssueQuery, map[string]string{"id": nodeID})
	if err != nil {
		return SubIssueRef{}, false, err
	}
	var resp parentIssueResponse
	if jsonErr := json.Unmarshal(stdout, &resp); jsonErr != nil {
		return SubIssueRef{}, false, fmt.Errorf("gh api graphql: parse response: %w", jsonErr)
	}
	p := resp.Data.Node.Parent
	if p == nil {
		return SubIssueRef{}, false, nil
	}
	return SubIssueRef{Number: p.Number, Title: p.Title, State: p.State}, true, nil
}
