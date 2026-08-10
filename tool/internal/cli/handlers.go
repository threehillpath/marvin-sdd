package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"threehillpath.com/claude-plan-workflow/tool/internal/config"
	"threehillpath.com/claude-plan-workflow/tool/internal/names"
	"threehillpath.com/claude-plan-workflow/tool/internal/parse"
	tmplpkg "threehillpath.com/claude-plan-workflow/tool/internal/template"
)

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
// jsonOut is threaded through for the --json flag; plain-text-by-default
// output is added in a later phase, so output is unconditionally JSON today.
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

// runParseTitle extracts a plan ident from a title string. jsonOut is threaded
// through for the --json flag; output is unconditionally JSON today.
func runParseTitle(stdout, stderr io.Writer, title string, jsonOut bool) error {
	ident, ok := parse.PlanIdent(title)
	out := parseTitleOutput{Found: ok}
	if ok {
		out.Plan = ident.Plan
		out.PlanNumber = names.PlanID(ident.Plan)
		out.Suffix = ident.Suffix
		out.Phase = ident.Phase
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// runParsePhaseList reads stdin and extracts phase issue numbers. stdin is
// injected (not os.Stdin directly) so tests can supply canned input. jsonOut
// is threaded through for the --json flag; output is unconditionally JSON today.
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
