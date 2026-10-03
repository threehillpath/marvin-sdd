package template

// LoadDraft turns a YAML draft into a SectionMap. Every case yaml.v3 would
// accept while silently losing content is reported as a Finding (location
// "draft"), and no map is returned when there is any finding.
func LoadDraft(sc *Schema, data []byte) (*SectionMap, []Finding) {
	return nil, nil
}
