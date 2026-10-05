package template

// ParseMarkdown exposes the markdown parser to the external tests, which
// inspect the SectionMap it builds. Production callers use CheckMarkdown.
func ParseMarkdown(sc *Schema, title, body string) *SectionMap {
	m, _, _, _ := parseMarkdown(sc, title, body)
	return m
}
