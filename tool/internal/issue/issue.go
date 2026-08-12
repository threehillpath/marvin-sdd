// Package issue provides operations for reading GitHub issues via the gh CLI.
package issue

import (
	"context"
	"fmt"
	"strings"

	"threehillpath.com/marvin-sdd/tool/internal/config"
	"threehillpath.com/marvin-sdd/tool/internal/exec"
	"threehillpath.com/marvin-sdd/tool/internal/gh"
)

// Item is a simplified view of a GitHub issue returned by List.
type Item struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	State  string   `json:"state"`
	Labels []string `json:"labels"`
}

// List returns issues from the project repo, optionally filtered by label and
// title prefix. state must be "open", "closed", or "all"; empty defaults to
// "open". label and titlePrefix may be empty to skip those filters. limit caps
// results; 0 uses the default of 100.
func List(ctx context.Context, runner exec.Runner, cfg *config.Config, label, titlePrefix, state string, limit int) ([]Item, error) {
	client := gh.New(runner)
	raw, err := client.IssueList(ctx, cfg.Repo, label, state, limit)
	if err != nil {
		return nil, fmt.Errorf("issue list: %w", err)
	}
	result := make([]Item, 0, len(raw))
	for _, r := range raw {
		if titlePrefix != "" && !strings.HasPrefix(r.Title, titlePrefix) {
			continue
		}
		labels := make([]string, len(r.Labels))
		for i, l := range r.Labels {
			labels[i] = l.Name
		}
		result = append(result, Item{
			Number: r.Number,
			Title:  r.Title,
			State:  r.State,
			Labels: labels,
		})
	}
	return result, nil
}
