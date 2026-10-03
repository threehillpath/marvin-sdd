package template

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// LoadSchema parses and validates schema bytes. origin names where the bytes
// came from ("built-in" or "project override: <path>") and prefixes every
// error.
func LoadSchema(origin string, data []byte) (*Schema, error) {
	var sc Schema
	if err := yaml.Unmarshal(data, &sc); err != nil {
		return nil, fmt.Errorf("%s: parsing schema: %w", origin, err)
	}
	return &sc, nil
}
