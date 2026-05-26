// Package findings provides plan-scoped JSON cache storage under .claude/cache/.
// Cache writes are validated as JSON before persisting. Clear removes only the
// target plan's directory, leaving sibling plans unaffected.
package findings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CachePath returns the canonical path for a cached findings file.
// It is relative to the working directory (no leading slash).
// Formula: .claude/cache/<planNumber>/<kind>/<name>.json
func CachePath(planNumber, kind, name string) string {
	return filepath.Join(".claude", "cache", planNumber, kind, name+".json")
}

// Cache validates payload as JSON, creates the directory tree, and writes the
// payload to CachePath(planNumber, kind, name). Returns an error if the payload
// is not valid JSON or if the file cannot be written.
func Cache(planNumber, kind, name string, payload []byte) error {
	if !json.Valid(payload) {
		return fmt.Errorf("findings cache: payload is not valid JSON")
	}

	path := CachePath(planNumber, kind, name)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("findings cache: create directories: %w", err)
	}

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("findings cache: write file: %w", err)
	}
	return nil
}

// Clear removes the .claude/cache/<planNumber>/ directory and all its contents.
// It is a no-op if the directory does not exist. Sibling plans are not affected.
func Clear(planNumber string) error {
	dir := filepath.Join(".claude", "cache", planNumber)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("findings clear: %w", err)
	}
	return nil
}
