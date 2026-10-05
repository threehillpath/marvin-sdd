package template

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"threehillpath.com/marvin-sdd/tool/internal/names"
	"threehillpath.com/marvin-sdd/tool/internal/parse"
)

// LoadSchema parses and validates schema bytes. origin names where the bytes
// came from ("built-in" or "project override: <path>") and prefixes every
// error.
func LoadSchema(origin string, data []byte) (*Schema, error) {
	pre := ""
	if origin != "" {
		pre = origin + ": "
	}
	var sc Schema
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&sc); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%sparsing schema: %w%s", pre, err, unknownFieldHint(err))
	}
	var second yaml.Node
	if err := dec.Decode(&second); err == nil {
		return nil, fmt.Errorf("%scontains a second YAML document (after a \"---\" line), which would be ignored silently. Remove the \"---\" line and everything after it, or move that content into the first document", pre)
	}
	if strings.TrimSpace(sc.Type) == "" {
		return nil, fmt.Errorf("%smissing \"type\". Add a \"type:\" line naming this schema; for a project override use the file's base name (impl-phase for impl-phase.yml). The built-in types are %s", pre, strings.Join(DefaultSchemaNames(), ", "))
	}
	if strings.TrimSpace(sc.TitlePrefix) == "" {
		return nil, fmt.Errorf("%smissing \"title_prefix\" for type %q. %s", pre, sc.Type, titlePrefixHint(sc.Type))
	}
	seenKey := map[string]bool{}
	for _, k := range sc.Metadata {
		if strings.TrimSpace(k) == "" {
			return nil, fmt.Errorf("%sa metadata key in \"metadata\" is empty. Remove it or give it a name", pre)
		}
		if seenKey[k] {
			return nil, fmt.Errorf("%smetadata key %q is listed twice. Remove the duplicate from \"metadata\"", pre, k)
		}
		seenKey[k] = true
	}
	seenID := map[string]bool{}
	seenHeading := map[string]bool{}
	var named []string
	for i, sec := range sc.Sections {
		if strings.TrimSpace(sec.ID) == "" {
			return nil, fmt.Errorf("%ssection %d has an empty \"id\". Set a unique id such as \"scope\"", pre, i+1)
		}
		if strings.TrimSpace(sec.Heading) == "" {
			return nil, fmt.Errorf("%ssection %q has an empty \"heading\". Set the heading text to render", pre, sec.ID)
		}
		if seenID[sec.ID] {
			return nil, fmt.Errorf("%sthe \"id\" %q appears twice in \"sections\". Rename one so every section id is unique", pre, sec.ID)
		}
		seenID[sec.ID] = true
		heading := strings.TrimSpace(sec.Heading)
		if seenHeading[heading] {
			return nil, fmt.Errorf("%sthe \"heading\" %q appears twice in \"sections\" (ignoring surrounding spaces). Rename one so every heading is unique", pre, heading)
		}
		seenHeading[heading] = true
		if sec.Numbered && sec.Named == nil {
			return nil, fmt.Errorf("%ssection %q is numbered but has no \"named\" field. Add \"named: true\" if headings come from content, else \"named: false\"", pre, sec.ID)
		}
		if sec.Numbered && sec.Named != nil && *sec.Named {
			named = append(named, fmt.Sprintf("%q", sec.ID))
		}
	}
	if len(named) > 1 {
		return nil, fmt.Errorf("%sthe numbered sections %s all set \"named: true\", but at most one may. Set \"named: false\" on all but one", pre, strings.Join(named, ", "))
	}
	kind, err := expectedKind(sc.TitlePrefix)
	if err != nil {
		return nil, fmt.Errorf("%s%w", pre, err)
	}
	sc.ExpectedKind = kind
	sc.loaded = true
	return &sc, nil
}

// unknownFieldHint is the advice appended to a decode error that includes an
// unknown field (and only then: a syntax or type error has no misspelt key).
func unknownFieldHint(err error) string {
	var te *yaml.TypeError
	if !errors.As(err, &te) {
		return ""
	}
	for _, e := range te.Errors {
		if strings.Contains(e, "not found in type") {
			return ". A field marvin does not know is an error (a misspelt key would otherwise be ignored silently): correct or remove it"
		}
	}
	return ""
}

// titlePrefixHint tells the caller what to add for a missing title_prefix:
// the built-in prefix for typ when there is one, else every accepted form.
func titlePrefixHint(typ string) string {
	if data, ok := DefaultSchema(typ); ok {
		var built Schema
		if err := yaml.Unmarshal(data, &built); err == nil && strings.TrimSpace(built.TitlePrefix) != "" {
			return fmt.Sprintf("Add a line like title_prefix: %q (the built-in prefix for this type; XXXXX is the issue number, N a phase ordinal)", built.TitlePrefix)
		}
	}
	return "Add a title_prefix starting with one of [PLAN-XXXXX-ARCH] (arch), [PLAN-XXXXX] (impl), [PLAN-XXXXX-N] (phase) or [TASK-XXXXX] (task), followed by a title placeholder like <Title>. XXXXX is the issue number, N a phase ordinal"
}

var (
	leadingBracket = regexp.MustCompile(`^\s*\[[^\]]*\]`)
	xRun           = regexp.MustCompile(`X+`)
)

// expectedKind classifies a title_prefix after replacing its placeholders:
// each run of X becomes zeros of the same length and a standalone N segment
// becomes 1.
func expectedKind(prefix string) (names.Kind, error) {
	tok := leadingBracket.FindString(prefix)
	if tok == "" {
		return names.Arch, fmt.Errorf("title_prefix %q has no leading [..] identifier. Set title_prefix to something like \"[PLAN-XXXXX] <Title>\"", prefix)
	}
	tok = xRun.ReplaceAllStringFunc(tok, func(x string) string { return strings.Repeat("0", len(x)) })
	segs := strings.Split(tok, "-")
	for i, seg := range segs {
		if strings.TrimRight(seg, "]") == "N" {
			segs[i] = "1" + seg[1:]
		}
	}
	tok = strings.Join(segs, "-")
	kind, ok := parse.Classify(tok)
	if !ok {
		return names.Arch, fmt.Errorf("title_prefix %q does not classify as an arch, impl, phase or task title. Use [PLAN-XXXXX-ARCH], [PLAN-XXXXX], [PLAN-XXXXX-N] or [TASK-XXXXX]", prefix)
	}
	return kind, nil
}
