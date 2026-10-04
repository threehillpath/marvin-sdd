package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/cli"
	"threehillpath.com/marvin-sdd/tool/internal/exectest"
)

const overrideSchemaFixture = `
type: quick-task
title_prefix: "[TASK-XXXXX] <Title>"
metadata:
  - Source Issue
sections:
  - id: override_marker
    heading: OVERRIDE-MARKER-SECTION
    required: false
    repeatable: false
    numbered: false
    named: false
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
	if !strings.Contains(got, "override_marker: |") {
		t.Errorf("expected the override's section key in the YAML skeleton, got:\n%s", got)
	}
	if strings.Contains(got, "problem_statement") || strings.Contains(got, "Problem Statement") {
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

	if !strings.Contains(stdout.String(), "problem_statement: |") {
		t.Errorf("expected embedded default schema's problem_statement key, got:\n%s", stdout.String())
	}
	if !strings.HasPrefix(stdout.String(), "title: \"[TASK-XXXXX] <Title>\"\n") {
		t.Errorf("expected a double-quoted YAML title placeholder, got:\n%s", stdout.String())
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
	if strings.Contains(stdout.String(), "problem_statement") {
		t.Errorf("expected no output from the embedded default when the override is unreadable, got:\n%s", stdout.String())
	}
}

// TestTemplateRenderSkeletonRejectsMalformedOverride verifies that a project
// override missing title_prefix fails the render with exit 1 and a message
// that names the origin and the field, rather than rendering from a schema
// the conformance check could not use.
func TestTemplateRenderSkeletonRejectsMalformedOverride(t *testing.T) {
	dir := t.TempDir()
	overrideDir := filepath.Join(dir, ".claude", "plan-workflow-templates")
	if err := os.MkdirAll(overrideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bad := "type: quick-task\nmetadata:\n  - Source Issue\nsections: []\n"
	if err := os.WriteFile(filepath.Join(overrideDir, "quick-task.yml"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
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

	err = root.Execute()
	var cliErr *cli.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("want *CLIError, got %T: %v", err, err)
	}
	if cliErr.Code != 1 {
		t.Errorf("Code = %d, want 1", cliErr.Code)
	}
	for _, w := range []string{"project override", "title_prefix"} {
		if !strings.Contains(cliErr.Msg, w) {
			t.Errorf("message %q missing %q", cliErr.Msg, w)
		}
	}
	if stdout.Len() != 0 {
		t.Errorf("want no stdout, got %q", stdout.String())
	}
}

// TestTemplateRenderRemovedFlagsAreUnknown verifies that the JSON input path
// is gone: --sections and --meta are unknown flags, a usage error (exit 1).
func TestTemplateRenderRemovedFlagsAreUnknown(t *testing.T) {
	for _, flag := range []string{"--sections", "--meta"} {
		t.Run(flag, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
			root.SetArgs([]string{"template", "render", "impl-plan", flag, "x.json"})
			err := root.Execute()
			if err == nil {
				t.Fatalf("%s must be an unknown flag, got success with stdout:\n%s", flag, stdout.String())
			}
			if !strings.Contains(err.Error(), "unknown flag") || !strings.Contains(err.Error(), flag) {
				t.Errorf("want an unknown-flag error naming %s, got: %v", flag, err)
			}
		})
	}
}

// TestTemplateRenderWithoutSkeletonSaysWhatToDo verifies that, with the JSON
// path gone and draft input not yet available, a plain render fails with a
// message naming the supported call rather than printing nothing.
func TestTemplateRenderWithoutSkeletonSaysWhatToDo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"template", "render", "impl-plan"})
	err := root.Execute()
	var cliErr *cli.CLIError
	if !errors.As(err, &cliErr) || cliErr.Code != 1 {
		t.Fatalf("want a CLIError with code 1, got %T: %v", err, err)
	}
	for _, w := range []string{"--skeleton", "--guidance", "impl-plan"} {
		if !strings.Contains(cliErr.Msg, w) {
			t.Errorf("message %q missing %q", cliErr.Msg, w)
		}
	}
	if stdout.Len() != 0 {
		t.Errorf("want no stdout, got %q", stdout.String())
	}
}

// TestTemplateRenderGuidancePrintsPlainText verifies --guidance prints the
// per-section guidance and the draft-writing rules.
func TestTemplateRenderGuidancePrintsPlainText(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"template", "render", "quick-task", "--guidance"})
	if err := root.Execute(); err != nil {
		t.Fatalf("--guidance returned error: %v\nstderr: %s", err, stderr.String())
	}
	for _, w := range []string{"schema: quick-task", "Problem Statement (required)", "problem_statement: a single | block", "no # comments"} {
		if !strings.Contains(stdout.String(), w) {
			t.Errorf("guidance output missing %q:\n%s", w, stdout.String())
		}
	}
}

// TestTemplateRenderSkeletonAndGuidanceAreExclusive verifies that passing
// both flags exits 1 and says to pick one.
func TestTemplateRenderSkeletonAndGuidanceAreExclusive(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"template", "render", "quick-task", "--skeleton", "--guidance"})
	err := root.Execute()
	var cliErr *cli.CLIError
	if !errors.As(err, &cliErr) || cliErr.Code != 1 {
		t.Fatalf("want a CLIError with code 1, got %T: %v", err, err)
	}
	for _, w := range []string{"--skeleton", "--guidance", "only one"} {
		if !strings.Contains(cliErr.Msg, w) {
			t.Errorf("message %q missing %q", cliErr.Msg, w)
		}
	}
	if stdout.Len() != 0 {
		t.Errorf("want no stdout, got %q", stdout.String())
	}
}

// TestTemplateRenderUnknownSchemaIsReportedBeforeFlagAdvice verifies that the
// schema is resolved first, so no message suggests a command that then fails.
func TestTemplateRenderUnknownSchemaIsReportedBeforeFlagAdvice(t *testing.T) {
	for _, args := range [][]string{
		{"template", "render", "nosuch"},
		{"template", "render", "nosuch", "--skeleton", "--guidance"},
	} {
		var stdout, stderr bytes.Buffer
		root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
		root.SetArgs(args)
		err := root.Execute()
		var cliErr *cli.CLIError
		if !errors.As(err, &cliErr) || cliErr.Code != 1 {
			t.Fatalf("%v: want a CLIError with code 1, got %T: %v", args, err, err)
		}
		if !strings.Contains(cliErr.Msg, `unknown schema "nosuch"`) || strings.Contains(cliErr.Msg, "--skeleton") {
			t.Errorf("%v: want the unknown-schema error alone, got %q", args, cliErr.Msg)
		}
	}
}
