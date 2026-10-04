package template

// ParseMarkdown turns a markdown issue body and a caller-supplied title into
// a SectionMap for sc.
func ParseMarkdown(sc *Schema, title, body string) *SectionMap {
	return &SectionMap{Source: SourceMarkdown, Title: title}
}
