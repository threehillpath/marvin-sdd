package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/cli"
	"threehillpath.com/marvin-sdd/tool/internal/exectest"
)

const overrideSchemaFixture = `
type: quick-task
metadata:
  - Source Issue
sections:
  - id: override_marker
    heading: OVERRIDE-MARKER-SECTION
    required: false
    repeatable: false
    numbered: false
`

// TestTemplateRenderProjectOverrideWinsOverEmbeddedDefault verifies that a
// project-supplied .claude/plan-workflow-templates/{type}.yml still takes
// precedence over marvin's go:embed'd built-in schema — the fallback to the
// embedded default must not short-circuit the override lookup.
func TestTemplateRenderProjectOverrideWinsOverEmbeddedDefault(t *testing.T) {
	dir := t.TempDir()
	overrideDir := filepath.Join(dir, ".claude", "plan-workflow-templates")
	if err := os.MkdirAll(overrideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overrideDir, "quick-task.yml"), []byte(overrideSchemaFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	// Chdir into a subdirectory of dir, not dir itself, so the override is
	// found only by walking upward — exercising findSchemaOverride's ascent
	// loop rather than matching on its very first candidate.
	sub := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"template", "render", "quick-task", "--skeleton"})

	if err := root.Execute(); err != nil {
		t.Fatalf("template render quick-task --skeleton returned error: %v\nstderr: %s", err, stderr.String())
	}

	got := stdout.String()
	if !strings.Contains(got, "OVERRIDE-MARKER-SECTION") {
		t.Errorf("expected project override content in output, got:\n%s", got)
	}
	if strings.Contains(got, "Problem Statement") {
		t.Errorf("expected embedded default's sections to be absent when an override is present, got:\n%s", got)
	}
}

// TestTemplateRenderFallsBackToEmbeddedDefault verifies that, absent a
// project override, marvin template render still succeeds using the
// embedded built-in schema — no dependency on the caller's CWD relative
// to the plugin's own skills/ directory.
func TestTemplateRenderFallsBackToEmbeddedDefault(t *testing.T) {
	dir := t.TempDir()

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"template", "render", "quick-task", "--skeleton"})

	if err := root.Execute(); err != nil {
		t.Fatalf("template render quick-task --skeleton returned error: %v\nstderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Problem Statement") {
		t.Errorf("expected embedded default schema's Problem Statement heading, got:\n%s", stdout.String())
	}
}

// TestTemplateRenderUnreadableOverrideFails verifies that a present but
// unreadable project override is reported as an error rather than silently
// falling back to the embedded default — the two cases must not be
// observably identical (CLAUDE.md's no-silent-failure rule).
func TestTemplateRenderUnreadableOverrideFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 0000 does not block reads")
	}

	dir := t.TempDir()
	overrideDir := filepath.Join(dir, ".claude", "plan-workflow-templates")
	if err := os.MkdirAll(overrideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	overridePath := filepath.Join(overrideDir, "quick-task.yml")
	if err := os.WriteFile(overridePath, []byte(overrideSchemaFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(overridePath, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(overridePath, 0o644) })

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"template", "render", "quick-task", "--skeleton"})

	if err := root.Execute(); err == nil {
		t.Fatalf("expected an error for an unreadable override, got success with stdout:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "Problem Statement") {
		t.Errorf("expected no output from the embedded default when the override is unreadable, got:\n%s", stdout.String())
	}
}
