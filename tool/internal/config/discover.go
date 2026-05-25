package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// yamlConfig is the raw YAML structure for unmarshalling.
type yamlConfig struct {
	Repo          string            `yaml:"repo"`
	ProjectNumber int               `yaml:"project_number"`
	ProjectID     string            `yaml:"project_id"`
	StatusFieldID string            `yaml:"status_field_id"`
	Statuses      map[string]string `yaml:"statuses"`
	TestCommands  map[string]string `yaml:"test_commands"`
	WorktreeBase  string            `yaml:"worktree_base"`
}

// Load walks up from startDir, looking for:
//  1. .claude/plan-workflow-config.yml (YAML, preferred)
//  2. .claude/plan-workflow-config.md  (legacy markdown table)
//
// Returns CLIError{Code:2} if neither is found, or if the found file is malformed.
func Load(startDir string) (*Config, error) {
	dir := startDir
	for {
		// Try YAML first.
		yamlPath := filepath.Join(dir, ".claude", "plan-workflow-config.yml")
		if _, err := os.Stat(yamlPath); err == nil {
			return loadYAML(yamlPath)
		}

		// Try legacy markdown.
		mdPath := filepath.Join(dir, ".claude", "plan-workflow-config.md")
		if _, err := os.Stat(mdPath); err == nil {
			return loadMarkdown(mdPath)
		}

		// Walk up.
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}
	return nil, configMissingError(startDir)
}

// loadYAML reads and parses a YAML config file.
func loadYAML(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, configBadError(path, err)
	}
	var raw yamlConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, configBadError(path, err)
	}
	if raw.Repo == "" {
		return nil, configBadError(path, fmt.Errorf("missing required field: repo"))
	}
	if raw.WorktreeBase == "" {
		return nil, configBadError(path, fmt.Errorf("missing required field: worktree_base"))
	}
	return &Config{
		Repo:          raw.Repo,
		ProjectNumber: raw.ProjectNumber,
		ProjectID:     raw.ProjectID,
		StatusFieldID: raw.StatusFieldID,
		Statuses:      raw.Statuses,
		TestCommands:  raw.TestCommands,
		WorktreeBase:  raw.WorktreeBase,
	}, nil
}

// tableRowRe matches a markdown table row like: | Key | `value` |
var tableRowRe = regexp.MustCompile("^\\|\\s*(.+?)\\s*\\|\\s*`(.+?)`\\s*\\|")

// loadMarkdown parses the legacy markdown table config format.
// It looks for rows matching: | Setting | Value | and maps well-known keys.
func loadMarkdown(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, configBadError(path, err)
	}
	defer f.Close()

	kvs := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		m := tableRowRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key := strings.TrimSpace(m[1])
		val := strings.TrimSpace(m[2])
		kvs[key] = val
	}
	if err := sc.Err(); err != nil {
		return nil, configBadError(path, err)
	}

	repo, ok := kvs["GitHub repo"]
	if !ok || repo == "" {
		return nil, configBadError(path, fmt.Errorf("missing 'GitHub repo' row"))
	}

	worktreeBase := kvs["Worktree base"]
	if worktreeBase == "" {
		return nil, configBadError(path, fmt.Errorf("missing required field: Worktree base"))
	}

	projNumStr := kvs["Project number"]
	projNum, _ := strconv.Atoi(projNumStr)

	statuses := map[string]string{
		"backlog":     kvs[`"Backlog" option ID`],
		"ready":       kvs[`"Ready" option ID`],
		"in_progress": kvs[`"In Progress" option ID`],
		"in_review":   kvs[`"In Review" option ID`],
		"done":        kvs[`"Done" option ID`],
	}

	return &Config{
		Repo:          repo,
		ProjectNumber: projNum,
		ProjectID:     kvs["Project ID"],
		StatusFieldID: kvs["Status field ID"],
		Statuses:      statuses,
		WorktreeBase:  worktreeBase,
	}, nil
}
