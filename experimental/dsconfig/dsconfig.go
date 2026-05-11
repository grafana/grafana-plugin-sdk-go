package dsconfig

import "fmt"

// DatasourceConfigSchema is the top-level schema definition.
// It acts as the single source of truth for datasource configuration.
type DatasourceConfigSchema struct {
	// SchemaVersion defines the version of the schema spec.
	SchemaVersion string `json:"schemaVersion"`

	// PluginType uniquely identifies the datasource plugin.
	PluginType string `json:"pluginType"`

	// PluginName is a human-readable name.
	PluginName string `json:"pluginName"`

	// Optional documentation URL.
	DocURL string `json:"docURL,omitempty"`

	// Fields defines all configuration fields.
	Fields []ConfigField `json:"fields"`

	// Optional UI grouping
	Groups []ConfigGroup `json:"groups,omitempty"`
}

func (s *DatasourceConfigSchema) Validate() error {
	if s.SchemaVersion == "" {
		return fmt.Errorf("schemaVersion is required")
	}
	if s.PluginType == "" {
		return fmt.Errorf("pluginType is required")
	}
	if s.PluginName == "" {
		return fmt.Errorf("pluginName is required")
	}
	if len(s.Fields) == 0 {
		return fmt.Errorf("fields is required")
	}

	for i := range s.Fields {
		if err := s.Fields[i].Validate(); err != nil {
			return err
		}
	}

	fieldIDs, err := s.FieldIDs()
	if err != nil {
		return err
	}

	if err := s.ValidateRefs(fieldIDs); err != nil {
		return err
	}

	return nil
}

// ValidateRefs checks that all group field references
// point to existing field IDs.
func (s *DatasourceConfigSchema) ValidateRefs(fieldIDs map[string]bool) error {
	for _, g := range s.Groups {
		for _, ref := range g.FieldRefs {
			if !fieldIDs[ref] {
				return fmt.Errorf("group %s references unknown field id: %s", g.ID, ref)
			}
		}
	}

	// Validate effect set keys reference known field IDs
	var visitEffects func(fields []ConfigField) error
	visitEffects = func(fields []ConfigField) error {
		for _, f := range fields {
			for i, eff := range f.Effects {
				for ref := range eff.Set {
					if !fieldIDs[ref] {
						return fmt.Errorf("field %s: effect[%d].set references unknown field id: %s", f.ID, i, ref)
					}
				}
			}
			if f.Item != nil {
				if err := visitEffects(f.Item.Fields); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visitEffects(s.Fields); err != nil {
		return err
	}

	return nil
}

// FieldIDs returns a set of all field IDs in the schema, checking for duplicates.
func (s *DatasourceConfigSchema) FieldIDs() (map[string]bool, error) {
	seen := map[string]bool{}

	var visit func(fields []ConfigField) error
	visit = func(fields []ConfigField) error {
		for i := range fields {
			f := fields[i]

			if f.ID == "" {
				return fmt.Errorf("field id is required")
			}

			if seen[f.ID] {
				return fmt.Errorf("duplicate field id: %s", f.ID)
			}
			seen[f.ID] = true

			if f.Item != nil {
				if err := visit(f.Item.Fields); err != nil {
					return err
				}
			}
		}

		return nil
	}

	if err := visit(s.Fields); err != nil {
		return nil, err
	}

	return seen, nil
}
