package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"threehillpath.com/marvin-sdd/tool/internal/clierr"
	"threehillpath.com/marvin-sdd/tool/internal/config"
	"threehillpath.com/marvin-sdd/tool/internal/names"
	"threehillpath.com/marvin-sdd/tool/internal/parse"
	tmplpkg "threehillpath.com/marvin-sdd/tool/internal/template"
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

// namesOutput is the JSON shape emitted by names derive. The PLAN-shaped
// fields (PlanNumber, MainBranch, PhaseBranch, WorktreePath) and the
// task-shaped fields (TaskNumber, TaskBranch) are mutually exclusive: the
// --task path populates only the latter, the normal path only the former,
// so both sets carry omitempty.
type namesOutput struct {
	PlanNumber   string      `json:"plan_number,omitempty"`
	TaskNumber   string      `json:"task_number,omitempty"`
	Type         string      `json:"type"`
	MainBranch   string      `json:"main_branch,omitempty"`
	PhaseBranch  string      `json:"phase_branch,omitempty"`
	TaskBranch   string      `json:"task_branch,omitempty"`
	WorktreePath string      `json:"worktree_path,omitempty"`
	TitlePrefix  titlePrefix `json:"title_prefix"`
}

type titlePrefix struct {
	Arch  string `json:"arch,omitempty"`
	Impl  string `json:"impl,omitempty"`
	Phase string `json:"phase,omitempty"`
	Task  string `json:"task,omitempty"`
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
// task selects the task-shaped output (--task): task_number, type,
// task_branch, worktree_path, title_prefix.task — a five-key set that is
// mutually exclusive with the PLAN-shaped fields and with --phase.
// jsonOut selects JSON output (--json); by default, output is plain text:
// one key:value line per populated field, with title_prefix flattened.
func runNamesDerive(stdout, stderr io.Writer, issueStr, typ, suffix, worktreeBaseFlag string, phase int, task, jsonOut bool) error {
	issue, err := strconv.Atoi(issueStr)
	if err != nil {
		return &CLIError{Code: 1, Msg: fmt.Sprintf("invalid issue number %q: %v", issueStr, err)}
	}

	if typ != "" && typ != "feature" && typ != "bug" {
		return &CLIError{Code: 1, Msg: fmt.Sprintf("invalid --type %q: must be \"feature\" or \"bug\"", typ)}
	}

	if task && phase > 0 {
		return &CLIError{Code: 1, Msg: "--task and --phase are mutually exclusive: a task issue is single-cycle and has no phases"}
	}

	var base string
	if phase > 0 || task {
		var berr error
		base, berr = resolveWorktreeBase(worktreeBaseFlag)
		if berr != nil {
			return berr
		}
	}

	if task {
		out := namesOutput{
			TaskNumber:   names.TaskNumber(issue),
			Type:         names.ResolveType(typ),
			TaskBranch:   names.TaskBranch(typ, issue),
			WorktreePath: names.TaskWorktreePath(base, issue),
			TitlePrefix: titlePrefix{
				Task: names.TitlePrefix(names.Task, issue, "", 0),
			},
		}

		if !jsonOut {
			writeKV(stdout, []kv{
				{Key: "task_number", Value: out.TaskNumber},
				{Key: "type", Value: out.Type},
				{Key: "task_branch", Value: out.TaskBranch},
				{Key: "worktree_path", Value: out.WorktreePath},
				{Key: "title_prefix_task", Value: out.TitlePrefix.Task},
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
	Kind       string `json:"kind,omitempty"`        // arch|impl|phase|task; only when the leading bracket classifies
	Task       int    `json:"task,omitempty"`        // task number, [TASK-XXXXX] titles only
	TaskNumber string `json:"task_number,omitempty"` // e.g. "TASK-00091"
	Slug       string `json:"slug,omitempty"`        // filesystem-safe slug of the title text after the bracket ident
}

// runParseTitle extracts a plan ident from a title string. jsonOut selects
// JSON output (--json); by default, output is plain text: one key:value line
// per populated field. found is always printed, even when false. slug is
// computed independently of the bracket-ident match (it just strips any
// leading "[...]" and slugifies the rest), so it can still be populated when
// found is false.
func runParseTitle(stdout, stderr io.Writer, title string, jsonOut bool) error {
	ident, ok := parse.PlanIdent(title)
	out := parseTitleOutput{Found: ok, Slug: parse.TitleSlug(title)}
	kind, classified := parse.Classify(title)
	if classified && kind == names.Task {
		// Task titles are the one case where found comes from the
		// classifier; PlanIdent may still match a later plan bracket, which
		// must not leak plan fields into the output.
		n, _ := parse.TaskIdent(title)
		out = parseTitleOutput{
			Found:      true,
			Kind:       "task",
			Task:       n,
			TaskNumber: names.TaskNumber(n),
			Slug:       out.Slug,
		}
	} else if ok {
		out.Plan = ident.Plan
		out.PlanNumber = names.PlanID(ident.Plan)
		out.Suffix = ident.Suffix
		out.Phase = ident.Phase
		if classified {
			out.Kind = kind.String()
		}
	}

	if !jsonOut {
		writeKV(stdout, []kv{
			{Key: "found", Value: strconv.FormatBool(out.Found)},
			{Key: "plan", Value: strconv.Itoa(out.Plan), Omit: out.Plan == 0},
			{Key: "plan_number", Value: out.PlanNumber, Omit: out.PlanNumber == ""},
			{Key: "suffix", Value: out.Suffix, Omit: out.Suffix == ""},
			{Key: "phase", Value: strconv.Itoa(out.Phase), Omit: out.Phase == 0},
			{Key: "kind", Value: out.Kind, Omit: out.Kind == ""},
			{Key: "task", Value: strconv.Itoa(out.Task), Omit: out.Task == 0},
			{Key: "task_number", Value: out.TaskNumber, Omit: out.TaskNumber == ""},
			{Key: "slug", Value: out.Slug, Omit: out.Slug == ""},
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

// runTemplateRender prints a schema's empty YAML draft (--skeleton), its
// plain-text guidance (--guidance), or a draft rendered to markdown (--draft).
// Rendered markdown goes to stdout and warnings to stderr; an error finding
// exits 3 with the findings on stderr and nothing on stdout.
func runTemplateRender(stdout, stderr io.Writer, schemaName string, skeleton, guidance bool, draftPath string) error {
	sc, origin, err := loadSchema(schemaName)
	if err != nil {
		return err
	}
	modes := 0
	for _, on := range []bool{skeleton, guidance, draftPath != ""} {
		if on {
			modes++
		}
	}
	if modes > 1 {
		return &CLIError{Code: 1, Msg: fmt.Sprintf("--skeleton, --guidance and --draft cannot be combined: pass only one. Run \"marvin template render %s --skeleton\" for the empty YAML draft, \"marvin template render %s --guidance\" for the help text, or \"marvin template render %s --draft <file.yml>\" to render a draft", schemaName, schemaName, schemaName)}
	}
	if modes == 0 {
		return &CLIError{Code: 1, Msg: fmt.Sprintf("nothing to render for %s: pass --draft <file.yml> to render a draft, \"marvin template render %s --skeleton\" to get an empty YAML draft, or \"marvin template render %s --guidance\" for how to fill it in", schemaName, schemaName, schemaName)}
	}
	if guidance {
		fmt.Fprint(stdout, tmplpkg.Guidance(sc))
		return nil
	}
	if skeleton {
		fmt.Fprint(stdout, tmplpkg.Skeleton(sc))
		return nil
	}
	body, res, err := checkInput(sc, origin, draftPath, "", "")
	if err != nil {
		return err
	}
	if res.HasErrors() {
		fmt.Fprint(stderr, res.Format())
		return clierr.NonConforming(fmt.Sprintf("the draft does not conform to the %s schema; nothing was rendered. Fix the findings above and run again", schemaName))
	}
	if len(res.Findings) > 0 {
		fmt.Fprint(stderr, res.Format())
	}
	fmt.Fprint(stdout, body)
	return nil
}

// resolveSchema returns the YAML schema bytes for schemaName and a short
// label identifying where they came from ("project override" or "built-in
// schema", used to make render/parse error messages actionable), per the
// precedence documented in skills/SHARED/CONFIG.md:
//  1. Project override: .claude/plan-workflow-templates/{schemaName}.yml,
//     found by walking up from cwd (sibling to the config file's own lookup).
//  2. Plugin default: the schema embedded in the marvin binary.
//
// The plugin default is always present for a known schema name and has no
// CWD dependency, so lookup only fails when schemaName has neither an
// override nor a built-in schema, or a present override cannot be read.
func resolveSchema(schemaName string) (data []byte, origin string, err error) {
	// A failure to determine the CWD does not block the embedded-default
	// fallback below, which needs no CWD at all — it only means a project
	// override (which does need one) cannot be searched for.
	if cwd, cwdErr := os.Getwd(); cwdErr == nil {
		overrideData, overridePath, ok, overrideErr := findSchemaOverride(cwd, schemaName)
		if overrideErr != nil {
			return nil, "", fmt.Errorf("resolving schema %q: %w", schemaName, overrideErr)
		}
		if ok {
			return overrideData, "project override: " + overridePath, nil
		}
	}
	if data, ok := tmplpkg.DefaultSchema(schemaName); ok {
		return data, "built-in", nil
	}
	return nil, "", fmt.Errorf("unknown schema %q: no project override and no plugin default", schemaName)
}

// findSchemaOverride walks up from startDir looking for a project-supplied
// .claude/plan-workflow-templates/{schemaName}.yml. A missing file at a
// given level is not an error — the walk continues upward — but any other
// read failure on a file that does exist there (permission denied, a
// directory in place of a file, ...) is reported rather than silently
// treated as "no override," which would otherwise fall through to the
// embedded default with no signal that the override was ignored.
func findSchemaOverride(startDir, schemaName string) ([]byte, string, bool, error) {
	filename := schemaName + ".yml"
	dir := startDir
	for {
		candidate := filepath.Join(dir, ".claude", "plan-workflow-templates", filename)
		data, err := os.ReadFile(candidate)
		if err == nil {
			return data, candidate, true, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, "", false, fmt.Errorf("reading project template override %q: %w", candidate, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, "", false, nil
		}
		dir = parent
	}
}

// checkInput checks a YAML draft (draftPath) or a markdown body (bodyPath,
// with title) against sc. Exactly one of the two paths must be set. For a
// draft it renders, so the goldmark verification backstop runs too; the
// rendered body is returned when the draft conforms.
func checkInput(sc *tmplpkg.Schema, origin, draftPath, bodyPath, title string) (string, tmplpkg.Result, error) {
	if (draftPath == "") == (bodyPath == "") {
		return "", tmplpkg.Result{}, &CLIError{Code: 1, Msg: "pass exactly one of --draft <file.yml> or --body-file <file.md>"}
	}
	if draftPath != "" {
		if title != "" {
			return "", tmplpkg.Result{}, &CLIError{Code: 1, Msg: "--title applies only to --body-file: with --draft the title comes from the draft's \"title:\" key. Remove --title, or edit \"title:\" in the draft"}
		}
		data, err := os.ReadFile(draftPath)
		if err != nil {
			return "", tmplpkg.Result{}, &CLIError{Code: 1, Msg: fmt.Sprintf("reading draft: %v", err)}
		}
		m, findings := tmplpkg.LoadDraft(sc, data)
		if len(findings) > 0 {
			return "", tmplpkg.Result{Type: sc.Type, Origin: origin, Findings: findings}, nil
		}
		body, res := tmplpkg.Render(sc, origin, m)
		return body, res, nil
	}
	data, err := os.ReadFile(bodyPath)
	if err != nil {
		return "", tmplpkg.Result{}, &CLIError{Code: 1, Msg: fmt.Sprintf("reading body file: %v", err)}
	}
	return "", tmplpkg.CheckMarkdown(sc, origin, title, string(data)), nil
}

// loadSchema resolves and loads the schema for schemaName; any failure is
// an operational error (exit 1), never a fallback to the built-in.
func loadSchema(schemaName string) (*tmplpkg.Schema, string, error) {
	schemaYAML, origin, err := resolveSchema(schemaName)
	if err != nil {
		return nil, "", &CLIError{Code: 1, Msg: err.Error()}
	}
	sc, err := tmplpkg.LoadSchema(origin, schemaYAML)
	if err != nil {
		return nil, "", &CLIError{Code: 1, Msg: err.Error()}
	}
	return sc, origin, nil
}

// runTemplateValidate prints the formatted check Result to stdout and exits 3
// when it holds an error finding.
func runTemplateValidate(stdout io.Writer, schemaName, draftPath, bodyPath, title string, jsonOut bool) error {
	sc, origin, err := loadSchema(schemaName)
	if err != nil {
		return err
	}
	_, res, err := checkInput(sc, origin, draftPath, bodyPath, title)
	if err != nil {
		return err
	}
	if jsonOut {
		type jsonFinding struct {
			Severity string `json:"severity"`
			Location string `json:"location"`
			Line     int    `json:"line"`
			Message  string `json:"message"`
		}
		out := struct {
			Schema   string        `json:"schema"`
			Origin   string        `json:"origin"`
			Findings []jsonFinding `json:"findings"`
		}{Schema: res.Type, Origin: res.Origin, Findings: []jsonFinding{}}
		for _, f := range res.Sorted() {
			out.Findings = append(out.Findings, jsonFinding{string(f.Severity), f.Location, f.Line, f.Message})
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			return err
		}
	} else {
		fmt.Fprint(stdout, res.Format())
	}
	if res.HasErrors() {
		return clierr.NonConforming(fmt.Sprintf("the input does not conform to the %s schema; see the findings above", schemaName))
	}
	return nil
}
