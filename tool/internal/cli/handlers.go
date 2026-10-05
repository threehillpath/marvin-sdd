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
// Rendered markdown goes to stdout; the schema line and any warnings go to
// stderr; an error finding exits 3 with the findings on stderr and nothing on
// stdout.
func runTemplateRender(stdout, stderr io.Writer, schemaName string, skeleton, guidance bool, draftPath string, draftSet bool) error {
	sc, origin, err := loadSchema(schemaName)
	if err != nil {
		return err
	}
	p := problems{prefix: "template render: "}
	if draftSet && draftPath == "" {
		p.add(emptyFlagMsg("draft"))
	}
	modes := 0
	for _, on := range []bool{skeleton, guidance, draftSet} {
		if on {
			modes++
		}
	}
	if modes > 1 {
		p.add(fmt.Sprintf("--skeleton, --guidance and --draft cannot be combined: pass only one. Run \"marvin template render %s --skeleton\" for the empty YAML draft, \"marvin template render %s --guidance\" for the help text, or \"marvin template render %s --draft <file.yml>\" to render a draft", schemaName, schemaName, schemaName))
	}
	if modes == 0 {
		p.add(fmt.Sprintf("nothing to render for %s: pass --draft <file.yml> to render a draft, \"marvin template render %s --skeleton\" to get an empty YAML draft, or \"marvin template render %s --guidance\" for how to fill it in", schemaName, schemaName, schemaName))
	}
	if err := p.err(); err != nil {
		return err
	}
	if guidance {
		fmt.Fprint(stdout, tmplpkg.Guidance(sc))
		return nil
	}
	if skeleton {
		fmt.Fprint(stdout, tmplpkg.Skeleton(sc))
		return nil
	}
	data, err := readInputFile("draft", draftPath)
	if err != nil {
		return err
	}
	body, _, res := checkDraftBytes(sc, origin, data)
	if res.HasErrors() {
		fmt.Fprint(stderr, res.Format())
		return clierr.NonConforming(fmt.Sprintf("the draft does not conform to the %s schema; nothing was rendered. Fix the findings above and run again", schemaName))
	}
	// Always print the schema line (and any warnings): whoever approves the
	// body should see which schema shaped it.
	fmt.Fprint(stderr, res.Format())
	fmt.Fprint(stdout, body)
	return nil
}

// resolveSchema returns the YAML schema bytes for schemaName and a short
// origin string: "project override: <path>" or "built-in". The origin is shown
// on the "schema:" line and in the JSON "origin" field, and it prefixes schema
// errors so they are actionable. Lookup follows the
// precedence documented in skills/SHARED/CONFIG.md:
//  1. Project override: .claude/plan-workflow-templates/{schemaName}.yml,
//     found by walking up from cwd (sibling to the config file's own lookup).
//  2. Plugin default: the schema embedded in the marvin binary.
//
// The plugin default is always present for a name in the fixed set and has no
// CWD dependency, so lookup fails only when schemaName is not one of the
// fixed types (checked first, before any override path is built) or a present
// override of a fixed type cannot be read.
func resolveSchema(schemaName string) (data []byte, origin string, err error) {
	// The type name carries business-rule weight, so it must be one of the
	// fixed built-in types. Check it before any path is built or any override
	// file is looked up: a name outside the set is an error even if a file of
	// that name exists, and a path-like name such as "../x" never reaches the
	// filesystem.
	builtin, ok := tmplpkg.DefaultSchema(schemaName)
	if !ok {
		return nil, "", fmt.Errorf("unknown schema %q: the template type must be one of %s", schemaName, strings.Join(tmplpkg.DefaultSchemaNames(), ", "))
	}
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
	return builtin, "built-in", nil
}

// findSchemaOverride walks up from startDir looking for a project-supplied
// .claude/plan-workflow-templates/{schemaName}.yml and returns its bytes and the
// path of the override it read. A missing file at a
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

// problems collects every usage problem found in one invocation so they are
// reported together rather than one rerun at a time. prefix names the command
// ("issue create: ") and is printed once: before a lone problem, or in the
// header of a list.
type problems struct {
	prefix string
	items  []string
}

func (p *problems) add(msg string) { p.items = append(p.items, msg) }

func (p problems) any() bool { return len(p.items) > 0 }

// err returns nil for no problems, prefix+message for one, and a single
// exit-1 CLIError with a "<prefix>N problems:" header and one bullet per
// problem for several.
func (p problems) err() error {
	switch len(p.items) {
	case 0:
		return nil
	case 1:
		return &CLIError{Code: 1, Msg: p.prefix + p.items[0]}
	}
	return &CLIError{Code: 1, Msg: fmt.Sprintf("%s%d problems:\n  - %s", p.prefix, len(p.items), strings.Join(p.items, "\n  - "))}
}

// errMsg is the message of err: a CLIError's Msg, else its Error().
func errMsg(err error) string {
	var ce *CLIError
	if errors.As(err, &ce) {
		return ce.Msg
	}
	return err.Error()
}

const (
	titleWithDraftMsg = "--title applies only to --body-file: with --draft the title comes from the draft's \"title:\" key. Remove --title, or edit \"title:\" in the draft"
	exactlyOneMsg     = "pass exactly one of --draft <file.yml> or --body-file <file.md>"
	bothInputsMsg     = "--draft and --body-file are mutually exclusive: " + exactlyOneMsg
)

const emptyTitleMsg = "--title was given an empty value: pass the issue title, or leave the flag out"

func emptyFlagMsg(name string) string {
	return fmt.Sprintf("--%s was given an empty value: pass a file path, or leave the flag out", name)
}

func emptyTemplateMsg() string {
	return fmt.Sprintf("--template needs one of %s; it was given an empty value", strings.Join(tmplpkg.DefaultSchemaNames(), ", "))
}

// readInputFile reads the file named by --<flag>. It is the only read of an
// input: callers check and send these same bytes, so a pipe or /dev/stdin,
// which cannot be read twice, works.
func readInputFile(flag, path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &CLIError{Code: 1, Msg: fmt.Sprintf("reading --%s %q: %v", flag, path, err)}
	}
	return data, nil
}

// checkDraftBytes checks a YAML draft against sc. It renders, so the goldmark
// verification backstop runs too; the rendered body is returned when the draft
// conforms. The returned title is the draft's "title:" (empty when the draft
// did not load).
func checkDraftBytes(sc *tmplpkg.Schema, origin string, data []byte) (body, title string, res tmplpkg.Result) {
	m, findings := tmplpkg.LoadDraft(sc, data)
	if len(findings) > 0 {
		return "", "", tmplpkg.Result{Type: sc.Type, Origin: origin, Findings: findings}
	}
	body, res = tmplpkg.Render(sc, origin, m)
	return body, m.Title, res
}

// checkBodyBytes checks a markdown body against sc with the given issue title.
func checkBodyBytes(sc *tmplpkg.Schema, origin, title string, data []byte) tmplpkg.Result {
	return tmplpkg.CheckMarkdown(sc, origin, title, string(data))
}

// titleChecksNote is printed to stderr when a missing title was reported as a
// usage problem and the title findings were dropped from the output, so a
// clean-looking body is not mistaken for a fully checked one.
const titleChecksNote = "note: checks that need the title (title/metadata cross-references and the rendered-structure check) did not run; they run once --title is given\n"

// dropTitleFindings removes the findings about the title, for the case where
// a missing title has already been reported as a usage problem.
func dropTitleFindings(res tmplpkg.Result) tmplpkg.Result {
	kept := res.Findings[:0:0]
	for _, f := range res.Findings {
		if f.Location != "title" {
			kept = append(kept, f)
		}
	}
	res.Findings = kept
	return res
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
	// The template names are a fixed set, so an override must declare the type
	// it is named for; a mismatch is never accepted and reported as another
	// schema.
	if sc.Type != schemaName {
		return nil, "", &CLIError{Code: 1, Msg: fmt.Sprintf("%s: declares type %q but was loaded for template %q: set \"type: %s\" in the file, or remove the file to use the built-in %s schema (expected %q)", origin, sc.Type, schemaName, schemaName, schemaName, schemaName)}
	}
	return sc, origin, nil
}

// validateFlags is the state of template validate's flags, including which
// were passed at all (a flag passed with an empty value is not absent).
type validateFlags struct {
	draft, body, title                string
	titleSet, draftSet, bodySet, json bool
}

// runTemplateValidate prints the formatted check Result to stdout and exits 3
// when it holds an error finding. Usage problems are collected and reported
// together; the input is checked anyway whenever it can be loaded, and then
// its findings go to stderr with the problems (stdout stays empty on exit 1).
func runTemplateValidate(stdout, stderr io.Writer, schemaName string, f validateFlags) error {
	p := problems{prefix: "template validate: "}
	sc, origin, err := loadSchema(schemaName)
	if err != nil {
		p.add(errMsg(err))
	}
	if f.draftSet && f.draft == "" {
		p.add(emptyFlagMsg("draft"))
	}
	if f.bodySet && f.body == "" {
		p.add(emptyFlagMsg("body-file"))
	}
	if f.draft != "" && f.body != "" {
		p.add(bothInputsMsg)
	}
	if !f.draftSet && !f.bodySet {
		p.add(exactlyOneMsg)
	}
	if f.titleSet && (f.draft != "" || (f.draftSet && !f.bodySet)) {
		p.add(titleWithDraftMsg)
	}

	// Read each input independently, so an unreadable file is reported even
	// when the schema or the flag combination is wrong.
	var draftData, bodyData []byte
	draftRead, bodyRead := false, false
	if f.draft != "" {
		if data, rerr := readInputFile("draft", f.draft); rerr != nil {
			p.add(errMsg(rerr))
		} else {
			draftData, draftRead = data, true
		}
	}
	if f.body != "" {
		if data, rerr := readInputFile("body-file", f.body); rerr != nil {
			p.add(errMsg(rerr))
		} else {
			bodyData, bodyRead = data, true
		}
	}
	var res tmplpkg.Result
	checked := false
	if sc != nil {
		switch {
		case draftRead && f.body == "":
			_, _, res = checkDraftBytes(sc, origin, draftData)
			checked = true
		case bodyRead && f.draft == "":
			res = checkBodyBytes(sc, origin, f.title, bodyData)
			checked = true
		}
	}
	if p.any() {
		if checked {
			fmt.Fprint(stderr, res.Format())
		}
		return p.err()
	}
	if f.json {
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
		for _, fd := range res.Sorted() {
			out.Findings = append(out.Findings, jsonFinding{string(fd.Severity), fd.Location, fd.Line, fd.Message})
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
