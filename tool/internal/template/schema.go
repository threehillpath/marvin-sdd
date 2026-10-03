package template

import (
	"fmt"
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
	var sc Schema
	if err := yaml.Unmarshal(data, &sc); err != nil {
		return nil, fmt.Errorf("%s: parsing schema: %w", origin, err)
	}
	kind, err := expectedKind(sc.TitlePrefix)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", origin, err)
	}
	sc.ExpectedKind = kind
	return &sc, nil
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
