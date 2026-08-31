package template_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tmpl "threehillpath.com/marvin-sdd/tool/internal/template"
)

// findSchemaDir walks up from the test file's location to find the skills/SHARED/templates dir.
// Tests run from the package directory, so we look relative to the module root.
func schemaDir(t *testing.T) string {
	t.Helper()
	// Walk up from cwd to find skills/SHARED/templates
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		candidate := filepath.Join(dir, "skills", "SHARED", "templates")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not find skills/SHARED/templates directory")
	return ""
}

// TestImplPlanNumberedSections asserts that two component sections and one
// verification_steps section produce ## 1. / ## 2. / ## 3. headings
// with the metadata block above them.
func TestImplPlanNumberedSections(t *testing.T) {
	sd := schemaDir(t)
	schemaPath := filepath.Join(sd, "impl-plan.yml")

	meta := []tmpl.KV{
		{Key: "Objective", Value: "Build something"},
		{Key: "Architecture Plan", Value: "#14"},
		{Key: "Source Issue", Value: "#2"},
		{Key: "Author", Value: "Test"},
		{Key: "Status", Value: "Upcoming"},
		{Key: "Last Updated", Value: "2026-01-01"},
	}
	sections := map[string][]string{
		"scope":              {"**Includes:** stuff\n\n**Does NOT include:** nothing"},
		"component":         {"First Component\n\nFirst component content", "Second Component\n\nSecond component content"},
		"verification_steps": {"Verify Step\n\nVerify step body"},
		"design_notes":      {"Some design notes"},
		"success_criteria":  {"- [ ] Passes"},
	}

	out, err := tmpl.Render(schemaPath, meta, sections)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	// Metadata block
	if !strings.Contains(out, "**Objective:**") {
		t.Error("missing **Objective:** in metadata")
	}
	if !strings.Contains(out, "Build something") {
		t.Error("missing Objective value")
	}

	// Numbered component headings — assert full heading text, not just prefix.
	if !strings.Contains(out, "## 1. First Component") {
		t.Error("missing ## 1. First Component heading")
	}
	if !strings.Contains(out, "## 2. Second Component") {
		t.Error("missing ## 2. Second Component heading")
	}
	// Verification steps continues the ordinal from components (2 components → ## 3.)
	if !strings.Contains(out, "## 3. Verify Step") {
		t.Error("missing ## 3. Verify Step heading for verification_steps")
	}

	// Non-numbered sections use plain ## headings
	if !strings.Contains(out, "## Scope") {
		t.Error("missing ## Scope heading")
	}
	if !strings.Contains(out, "## Design Notes") {
		t.Error("missing ## Design Notes heading")
	}
	if !strings.Contains(out, "## Success Criteria") {
		t.Error("missing ## Success Criteria heading")
	}
}

// TestImplPhaseOptionalTDDEntryPoint verifies that omitting the optional
// tdd_entry_point section in impl-phase produces no heading for it and exits 0.
func TestImplPhaseOptionalTDDEntryPoint(t *testing.T) {
	sd := schemaDir(t)
	schemaPath := filepath.Join(sd, "impl-phase.yml")

	meta := []tmpl.KV{
		{Key: "Implementation Plan", Value: "#15"},
		{Key: "Plan Number", Value: "PLAN-00002"},
		{Key: "Status", Value: "Upcoming"},
	}
	sections := map[string][]string{
		"objective":        {"Deliver the foundation packages"},
		"scope":            {"**Includes:** stuff"},
		"components":       {"Component list here"},
		"verification":     {"Run go test ./..."},
		"success_criteria": {"- [ ] Tests pass"},
		// tdd_entry_point intentionally omitted
	}

	out, err := tmpl.Render(schemaPath, meta, sections)
	if err != nil {
		t.Fatalf("Render returned error (expected 0): %v", err)
	}

	if strings.Contains(out, "TDD Entry Point") {
		t.Error("output should not contain TDD Entry Point heading when section is omitted")
	}
	if !strings.Contains(out, "## Objective") {
		t.Error("missing ## Objective heading")
	}
}

// TestArchPlanMetadataKey verifies that rendering an arch-plan with a "Date" metadata
// entry emits "**Date:**" in the output.
func TestArchPlanMetadataKey(t *testing.T) {
	sd := schemaDir(t)
	schemaPath := filepath.Join(sd, "arch-plan.yml")

	meta := []tmpl.KV{
		{Key: "Source Issue", Value: "#2"},
		{Key: "Plan Number", Value: "PLAN-00042"},
		{Key: "Author", Value: "Test"},
		{Key: "Status", Value: "Upcoming"},
		{Key: "Date", Value: "2026-01-01"},
	}
	sections := map[string][]string{
		"problem_statement":      {"The problem"},
		"scope":                  {"**Includes:** x\n**Excludes:** y"},
		"domain_model_impacts":   {"Entities affected"},
		"integration_points":     {"Layers touched"},
		"cross_cutting_concerns": {"Auth"},
		"architectural_decisions": {"| Decision | Approach | Rationale |"},
		"adr_candidates":         {"- [ ] Some decision"},
		"constraints_and_tradeoffs": {"Trade-offs"},
		"tdd_strategy":           {"Start with a failing test"},
		"open_questions":         {"None"},
	}

	out, err := tmpl.Render(schemaPath, meta, sections)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if !strings.Contains(out, "**Date:**") {
		t.Errorf("expected **Date:** in output, got:\n%s", out)
	}
	if !strings.Contains(out, "2026-01-01") {
		t.Error("missing Date value in output")
	}
}

// TestSkeletonEmitsMetadataAndHeadings verifies that Skeleton emits bold metadata
// key placeholders before section headings, with no content validation.
func TestSkeletonEmitsMetadataAndHeadings(t *testing.T) {
	sd := schemaDir(t)
	schemaPath := filepath.Join(sd, "impl-plan.yml")

	out, err := tmpl.Skeleton(schemaPath)
	if err != nil {
		t.Fatalf("Skeleton returned error: %v", err)
	}

	// Metadata keys must appear as bold placeholders.
	for _, key := range []string{"Objective", "Architecture Plan", "Source Issue", "Author", "Status"} {
		want := "**" + key + ":**"
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in skeleton output, got:\n%s", want, out)
		}
	}

	// Section headings must be present.
	if !strings.Contains(out, "## Scope") {
		t.Errorf("expected '## Scope' in skeleton output, got:\n%s", out)
	}
}

// TestQuickTaskSkeletonHeadingsInOrder verifies that quick-task's skeleton
// output includes all six required section headings in the schema's order.
func TestQuickTaskSkeletonHeadingsInOrder(t *testing.T) {
	sd := schemaDir(t)
	schemaPath := filepath.Join(sd, "quick-task.yml")

	out, err := tmpl.Skeleton(schemaPath)
	if err != nil {
		t.Fatalf("Skeleton returned error: %v", err)
	}

	headings := []string{
		"## Problem Statement",
		"## Scope",
		"## Technical Analysis",
		"## TDD Entry Point",
		"## Implementation Notes",
		"## Success Criteria",
	}

	lastIdx := -1
	for _, h := range headings {
		idx := strings.Index(out, h)
		if idx == -1 {
			t.Fatalf("missing heading %q in skeleton output:\n%s", h, out)
		}
		if idx <= lastIdx {
			t.Fatalf("heading %q out of order in skeleton output:\n%s", h, out)
		}
		lastIdx = idx
	}
}

// TestQuickTaskRenderMissingTDDEntryPoint verifies that Render fails
// validation when the required tdd_entry_point section is omitted.
func TestQuickTaskRenderMissingTDDEntryPoint(t *testing.T) {
	sd := schemaDir(t)
	schemaPath := filepath.Join(sd, "quick-task.yml")

	meta := []tmpl.KV{
		{Key: "Source Issue", Value: "#91"},
		{Key: "Task Number", Value: "TASK-00091"},
		{Key: "Author", Value: "Test"},
		{Key: "Status", Value: "Upcoming"},
		{Key: "Date", Value: "2026-01-01"},
	}
	sections := map[string][]string{
		"problem_statement":    {"The problem"},
		"scope":                {"**Includes:** x"},
		"technical_analysis":   {"Analysis"},
		"implementation_notes": {"Notes"},
		"success_criteria":     {"- [ ] Done"},
		// tdd_entry_point intentionally omitted
	}

	_, err := tmpl.Render(schemaPath, meta, sections)
	if err == nil {
		t.Error("expected error for missing required tdd_entry_point section, got nil")
	}
}

// TestQuickTaskRenderMissingTechnicalAnalysis verifies that Render fails
// validation when the required technical_analysis section is omitted.
func TestQuickTaskRenderMissingTechnicalAnalysis(t *testing.T) {
	sd := schemaDir(t)
	schemaPath := filepath.Join(sd, "quick-task.yml")

	meta := []tmpl.KV{
		{Key: "Source Issue", Value: "#91"},
		{Key: "Task Number", Value: "TASK-00091"},
		{Key: "Author", Value: "Test"},
		{Key: "Status", Value: "Upcoming"},
		{Key: "Date", Value: "2026-01-01"},
	}
	sections := map[string][]string{
		"problem_statement":    {"The problem"},
		"scope":                {"**Includes:** x"},
		"tdd_entry_point":      {"What: ...\nWhere: ...\nPasses when: ..."},
		"implementation_notes": {"Notes"},
		"success_criteria":     {"- [ ] Done"},
		// technical_analysis intentionally omitted
	}

	_, err := tmpl.Render(schemaPath, meta, sections)
	if err == nil {
		t.Error("expected error for missing required technical_analysis section, got nil")
	}
}

// TestRenderRequiredSectionMissing verifies that omitting a required section returns an error.
func TestRenderRequiredSectionMissing(t *testing.T) {
	sd := schemaDir(t)
	schemaPath := filepath.Join(sd, "impl-phase.yml")

	meta := []tmpl.KV{
		{Key: "Implementation Plan", Value: "#15"},
		{Key: "Plan Number", Value: "PLAN-00002"},
		{Key: "Status", Value: "Upcoming"},
	}
	// Omit required "objective" section
	sections := map[string][]string{
		"scope":            {"stuff"},
		"components":       {"stuff"},
		"verification":     {"stuff"},
		"success_criteria": {"stuff"},
	}

	_, err := tmpl.Render(schemaPath, meta, sections)
	if err == nil {
		t.Error("expected error for missing required section, got nil")
	}
}
