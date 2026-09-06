// Package template renders plan issue bodies from YAML schema definitions.
// It assembles structure deterministically — ordered metadata, correct headings,
// repeatable/numbered section handling — but never authors section content.
package template

import (
	"embed"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// defaultSchemas embeds the plugin's built-in YAML schemas at compile time,
// so rendering never depends on skills/SHARED/templates/ (or any other path)
// being reachable on disk from the caller's working directory.
//
//go:embed schemas/*.yml
var defaultSchemas embed.FS

// DefaultSchema returns the plugin's built-in schema bytes for name (e.g.
// "impl-plan"), and false if no built-in schema exists under that name.
func DefaultSchema(name string) ([]byte, bool) {
	data, err := defaultSchemas.ReadFile("schemas/" + name + ".yml")
	if err != nil {
		return nil, false
	}
	return data, true
}

// KV is an ordered metadata key-value pair.
type KV struct {
	Key   string
	Value string
}

// schemaSection mirrors the YAML section definition.
type schemaSection struct {
	ID         string `yaml:"id"`
	Heading    string `yaml:"heading"`
	Required   bool   `yaml:"required"`
	Repeatable bool   `yaml:"repeatable"`
	Numbered   bool   `yaml:"numbered"`
}

// schema is the top-level YAML structure.
type schema struct {
	Metadata []string        `yaml:"metadata"`
	Sections []schemaSection `yaml:"sections"`
}

// Render assembles a plan issue body from:
//   - schemaYAML: the schema's YAML bytes (from DefaultSchema or a project override)
//   - meta: ordered key-value pairs for the bold metadata block
//   - sections: map from section id → one or more content blocks
//
// Returns an error if a required section is absent, a non-repeatable section
// has more than one block, or the schema YAML cannot be parsed.
func Render(schemaYAML []byte, meta []KV, sections map[string][]string) (string, error) {
	var sc schema
	if err := yaml.Unmarshal(schemaYAML, &sc); err != nil {
		return "", fmt.Errorf("parsing schema: %w", err)
	}

	// Validate required sections.
	for _, sec := range sc.Sections {
		if sec.Required {
			if _, ok := sections[sec.ID]; !ok {
				return "", fmt.Errorf("required section %q is missing", sec.ID)
			}
		}
	}

	var sb strings.Builder

	// Metadata block: bold key-value pairs.
	for _, kv := range meta {
		fmt.Fprintf(&sb, "**%s:** %s\n", kv.Key, kv.Value)
	}

	// Running ordinal for numbered sections (components + verification_steps in impl-plan).
	ordinal := 0

	for _, sec := range sc.Sections {
		blocks, supplied := sections[sec.ID]
		if !supplied {
			// Optional sections not supplied are silently omitted.
			continue
		}

		if !sec.Repeatable {
			if len(blocks) > 1 {
				return "", fmt.Errorf("section %q is not repeatable but %d blocks were supplied", sec.ID, len(blocks))
			}
			// Plain (non-numbered, non-repeatable) section.
			fmt.Fprintf(&sb, "\n## %s\n\n%s\n", sec.Heading, blocks[0])
			continue
		}

		// Repeatable section.
		for _, block := range blocks {
			if sec.Numbered {
				ordinal++
				// First line of each block is the per-instance heading text;
				// the schema heading is a placeholder and is not used here.
				heading, body, _ := strings.Cut(block, "\n")
				body = strings.TrimLeft(body, "\n")
				if body == "" {
					fmt.Fprintf(&sb, "\n## %d. %s\n", ordinal, heading)
				} else {
					fmt.Fprintf(&sb, "\n## %d. %s\n\n%s\n", ordinal, heading, body)
				}
			} else {
				fmt.Fprintf(&sb, "\n## %s\n\n%s\n", sec.Heading, block)
			}
		}
	}

	return sb.String(), nil
}

// Skeleton returns a minimal scaffold for schemaYAML: bold metadata key
// placeholders followed by empty section headings. Unlike Render, it does not
// validate required sections or accept section content.
func Skeleton(schemaYAML []byte) (string, error) {
	var sc schema
	if err := yaml.Unmarshal(schemaYAML, &sc); err != nil {
		return "", fmt.Errorf("parsing schema: %w", err)
	}

	var sb strings.Builder

	// Metadata block: one placeholder line per key.
	for _, key := range sc.Metadata {
		fmt.Fprintf(&sb, "**%s:**\n", key)
	}

	ordinal := 0
	for _, sec := range sc.Sections {
		if sec.Numbered {
			ordinal++
			fmt.Fprintf(&sb, "\n## %d. %s\n", ordinal, sec.Heading)
		} else {
			fmt.Fprintf(&sb, "\n## %s\n", sec.Heading)
		}
	}
	return sb.String(), nil
}
