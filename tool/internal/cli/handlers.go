package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"threehillpath.com/claude-plan-workflow/tool/internal/config"
	"threehillpath.com/claude-plan-workflow/tool/internal/names"
	"threehillpath.com/claude-plan-workflow/tool/internal/parse"
	tmplpkg "threehillpath.com/claude-plan-workflow/tool/internal/template"
)

// kv is one line of plain-text object-command output: "Key: Value", printed
// unless Omit is true. Omit encodes the same "populated" rule as the JSON
// struct's `omitempty` tag: a field with no omitempty tag is never omitted
// (Omit is always false); a field with omitempty is omitted exactly when its
// value is the zero value, matching encoding/json's own omission rule.
type kv struct {
	Key   string
	Value string
	Omit  bool
}

// writeKV writes one "key: value" line to w for each entry not marked Omit,
// in the given order. Shared by the object commands (names derive, parse
// title, pr find) and by parse phase-list, whose output is key:value rather
// than columnar, so the populated-field rule is expressed once.
func writeKV(w io.Writer, entries []kv) {
	for _, e := range entries {
		if e.Omit {
			continue
		}
		fmt.Fprintf(w, "%s: %s\n", e.Key, e.Value)
	}
}

// runConfigGet prints a single config value to stdout.
func runConfigGet(stdout, stderr io.Writer, key string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return &CLIError{Code: 1, Msg: fmt.Sprintf("cannot determine working directory: %v", err)}
	}
	cfg, err := config.Load(cwd)
	if err != nil {
		return err // already a CLIError{Code:2} from config package
	}

	var val string
	switch key {
	case "repo":
		val = cfg.Repo
	case "project_number":
		val = strconv.Itoa(cfg.ProjectNumber)
	case "project_id":
		val = cfg.ProjectID
	case "status_field_id":
		val = cfg.StatusFieldID
	case "owner":
		val = cfg.Owner()
	case "worktree_base":
		val = cfg.WorktreeBase
	default:
		id, present, serr := cfg.StatusOptionID(key)
		if serr != nil {
			return &CLIError{Code: 1, Msg: fmt.Sprintf("unknown config key %q", key)}
		}
		if !present {
			val = "n/a"
		} else {
			val = id
		}
	}
	fmt.Fprintln(stdout, val)
	return nil
}

// namesOutput is the JSON shape emitted by names derive.
type namesOutput struct {
	PlanNumber   string      `json:"plan_number"`
	Type         string      `json:"type"`
	MainBranch   string      `json:"main_branch"`
	PhaseBranch  string      `json:"phase_branch,omitempty"`
	WorktreePath string      `json:"worktree_path,omitempty"`
	TitlePrefix  titlePrefix `json:"title_prefix"`
}

type titlePrefix struct {
	Arch  string `json:"arch"`
	Impl  string `json:"impl"`
	Phase string `json:"phase,omitempty"`
}

// resolveWorktreeBase returns the worktree base to use.
// The flag value takes precedence; otherwise the config file is required.
func resolveWorktreeBase(flagVal string) (string, error) {
	if flagVal != "" {
		return flagVal, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", &CLIError{Code: 1, Msg: fmt.Sprintf("cannot determine working directory: %v", err)}
	}
	cfg, err := config.Load(cwd)
	if err != nil {
		return "", err // already a CLIError{Code:2}
	}
	return cfg.WorktreeBase, nil
}

// runNamesDerive derives all canonical names for an issue number.
// typ is "feature" or "bug"; empty defaults to "feature" (defaulting logic
// lives in names.ResolveType). A non-empty typ that isn't "feature" or "bug"
// is rejected here, at the CLI layer where user input first arrives.
// jsonOut selects JSON output (--json); by default, output is plain text:
// one key:value line per populated field, with title_prefix flattened to
// three lines.
func runNamesDerive(stdout, stderr io.Writer, issueStr, typ, suffix, worktreeBaseFlag string, phase int, jsonOut bool) error {
	issue, err := strconv.Atoi(issueStr)
	if err != nil {
		return &CLIError{Code: 1, Msg: fmt.Sprintf("invalid issue number %q: %v", issueStr, err)}
	}

	if typ != "" && typ != "feature" && typ != "bug" {
		return &CLIError{Code: 1, Msg: fmt.Sprintf("invalid --type %q: must be \"feature\" or \"bug\"", typ)}
	}

	var base string
	if phase > 0 {
		var berr error
		base, berr = resolveWorktreeBase(worktreeBaseFlag)
		if berr != nil {
			return berr
		}
	}

	out := namesOutput{
		PlanNumber: names.PlanID(issue),
		Type:       names.ResolveType(typ),
		MainBranch: names.TrunkBranch(typ, issue, suffix),
		TitlePrefix: titlePrefix{
			Arch: names.TitlePrefix(names.Arch, issue, suffix, phase),
			Impl: names.TitlePrefix(names.Impl, issue, suffix, phase),
		},
	}
	if phase > 0 {
		out.PhaseBranch = names.PhaseBranch(typ, issue, suffix, phase)
		out.WorktreePath = names.WorktreePath(base, issue, suffix, phase)
		out.TitlePrefix.Phase = names.TitlePrefix(names.Phase, issue, suffix, phase)
	}

	if !jsonOut {
		writeKV(stdout, []kv{
			{Key: "plan_number", Value: out.PlanNumber},
			{Key: "type", Value: out.Type},
			{Key: "main_branch", Value: out.MainBranch},
			{Key: "phase_branch", Value: out.PhaseBranch, Omit: out.PhaseBranch == ""},
			{Key: "worktree_path", Value: out.WorktreePath, Omit: out.WorktreePath == ""},
			{Key: "title_prefix_arch", Value: out.TitlePrefix.Arch},
			{Key: "title_prefix_impl", Value: out.TitlePrefix.Impl},
			{Key: "title_prefix_phase", Value: out.TitlePrefix.Phase, Omit: out.TitlePrefix.Phase == ""},
		})
		return nil
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return &CLIError{Code: 1, Msg: fmt.Sprintf("encoding output: %v", err)}
	}
	return nil
}

// parseTitleOutput is the JSON shape for parse title.
type parseTitleOutput struct {
	Found      bool   `json:"found"`
	Plan       int    `json:"plan,omitempty"`
	PlanNumber string `json:"plan_number,omitempty"` // lowercase path form, e.g. "plan-00042"
	Suffix     string `json:"suffix,omitempty"`
	Phase      int    `json:"phase,omitempty"`
}

// runParseTitle extracts a plan ident from a title string. jsonOut selects
// JSON output (--json); by default, output is plain text: one key:value line
// per populated field. found is always printed, even when false.
func runParseTitle(stdout, stderr io.Writer, title string, jsonOut bool) error {
	ident, ok := parse.PlanIdent(title)
	out := parseTitleOutput{Found: ok}
	if ok {
		out.Plan = ident.Plan
		out.PlanNumber = names.PlanID(ident.Plan)
		out.Suffix = ident.Suffix
		out.Phase = ident.Phase
	}

	if !jsonOut {
		writeKV(stdout, []kv{
			{Key: "found", Value: strconv.FormatBool(out.Found)},
			{Key: "plan", Value: strconv.Itoa(out.Plan), Omit: out.Plan == 0},
			{Key: "plan_number", Value: out.PlanNumber, Omit: out.PlanNumber == ""},
			{Key: "suffix", Value: out.Suffix, Omit: out.Suffix == ""},
			{Key: "phase", Value: strconv.Itoa(out.Phase), Omit: out.Phase == 0},
		})
		return nil
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// runParsePhaseList reads stdin and extracts phase issue numbers. stdin is
// injected (not os.Stdin directly) so tests can supply canned input. jsonOut
// selects JSON output (--json); by default, output stays key:value (not
// columnar, unlike the list commands): found: true|false then an issues line
// ("issues: 39,40,41" or "issues: (none)" when empty).
func runParsePhaseList(stdin io.Reader, stdout, stderr io.Writer, jsonOut bool) error {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return &CLIError{Code: 1, Msg: fmt.Sprintf("reading stdin: %v", err)}
	}
	nums, ok := parse.PhaseListFromComment(string(data))
	out := struct {
		Found  bool  `json:"found"`
		Issues []int `json:"issues"`
	}{Found: ok, Issues: nums}
	if out.Issues == nil {
		out.Issues = []int{}
	}

	if !jsonOut {
		issuesStr := "(none)"
		if len(out.Issues) > 0 {
			strs := make([]string, len(out.Issues))
			for i, n := range out.Issues {
				strs[i] = strconv.Itoa(n)
			}
			issuesStr = strings.Join(strs, ",")
		}
		writeKV(stdout, []kv{
			{Key: "found", Value: strconv.FormatBool(out.Found)},
			{Key: "issues", Value: issuesStr},
		})
		return nil
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// runTemplateRender renders a plan template from schema.
// When skeleton is true, emits empty section headings without requiring content.
func runTemplateRender(stdout, stderr io.Writer, schemaName, metaFile, sectionsFile string, skeleton bool) error {
	schemaPath, err := resolveSchemaPath(schemaName)
	if err != nil {
		return &CLIError{Code: 1, Msg: err.Error()}
	}

	if skeleton {
		out, err := tmplpkg.Skeleton(schemaPath)
		if err != nil {
			return &CLIError{Code: 1, Msg: err.Error()}
		}
		fmt.Fprint(stdout, out)
		return nil
	}

	var meta []tmplpkg.KV
	if metaFile != "" {
		data, err := os.ReadFile(metaFile)
		if err != nil {
			return &CLIError{Code: 1, Msg: fmt.Sprintf("reading meta file: %v", err)}
		}
		if err := json.Unmarshal(data, &meta); err != nil {
			return &CLIError{Code: 1, Msg: fmt.Sprintf("parsing meta JSON: %v", err)}
		}
	}

	var sections map[string][]string
	if sectionsFile != "" {
		data, err := os.ReadFile(sectionsFile)
		if err != nil {
			return &CLIError{Code: 1, Msg: fmt.Sprintf("reading sections file: %v", err)}
		}
		if err := json.Unmarshal(data, &sections); err != nil {
			return &CLIError{Code: 1, Msg: fmt.Sprintf("parsing sections JSON: %v", err)}
		}
	}
	if sections == nil {
		sections = map[string][]string{}
	}

	out, err := tmplpkg.Render(schemaPath, meta, sections)
	if err != nil {
		return &CLIError{Code: 1, Msg: err.Error()}
	}
	fmt.Fprint(stdout, out)
	return nil
}

// resolveSchemaPath finds the YAML schema file by walking up from cwd.
func resolveSchemaPath(schemaName string) (string, error) {
	filename := schemaName + ".yml"
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %v", err)
	}
	dir := cwd
	for {
		candidate := filepath.Join(dir, "skills", "SHARED", "templates", filename)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("schema %q not found in skills/SHARED/templates/ (searched from %s)", schemaName, cwd)
}
