package dsconfig_test

import (
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/dsconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// DatasourceConfigSchema.Validate
// ============================================================

// TestSchemaValidate_MinimalValid verifies that the simplest possible
// schema (one storage field) passes validation without errors.
func TestSchemaValidate_MinimalValid(t *testing.T) {
	s := minimalSchema(validStorageField("url", "url"))
	require.NoError(t, s.Validate())
}

// TestSchemaValidate_EmptyFields confirms that a schema with zero
// fields is valid — plugins may start with no config.
func TestSchemaValidate_EmptyFields(t *testing.T) {
	s := minimalSchema()
	require.NoError(t, s.Validate())
}

// TestSchemaValidate_PropagatesFieldError ensures that a validation
// error on an individual field bubbles up through the root Validate().
func TestSchemaValidate_PropagatesFieldError(t *testing.T) {
	s := minimalSchema(dsconfig.ConfigField{ID: "", Key: "x", ValueType: dsconfig.StringType})
	assert.ErrorContains(t, s.Validate(), "field id is required")
}

// ============================================================
// DatasourceConfigSchema.FieldIDs
// ============================================================

// TestFieldIDs_CollectsTopLevel verifies that FieldIDs returns all
// top-level field IDs as a set.
func TestFieldIDs_CollectsTopLevel(t *testing.T) {
	s := minimalSchema(
		validStorageField("a", "a"),
		validStorageField("b", "b"),
	)
	ids, err := s.FieldIDs()
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"a": true, "b": true}, ids)
}

// TestFieldIDs_CollectsItemFields verifies that FieldIDs recursively
// collects IDs from nested array item fields.
func TestFieldIDs_CollectsItemFields(t *testing.T) {
	s := minimalSchema(dsconfig.ConfigField{
		ID:        "headers",
		Key:       "headers",
		ValueType: dsconfig.ArrayType,
		Target:    dsconfig.JSONDataTarget,
		Item: &dsconfig.FieldItemSchema{
			ValueType: dsconfig.ObjectType,
			Fields: []dsconfig.ConfigField{
				{ID: "headers.item.key", Key: "key", ValueType: dsconfig.StringType, IsItemField: ptr(true)},
				{ID: "headers.item.value", Key: "value", ValueType: dsconfig.StringType, IsItemField: ptr(true)},
			},
		},
	})
	ids, err := s.FieldIDs()
	require.NoError(t, err)
	assert.True(t, ids["headers"])
	assert.True(t, ids["headers.item.key"])
	assert.True(t, ids["headers.item.value"])
}

// TestFieldIDs_DuplicateID ensures that two top-level fields sharing
// the same ID are rejected.
func TestFieldIDs_DuplicateID(t *testing.T) {
	s := minimalSchema(
		validStorageField("dup", "a"),
		validStorageField("dup", "b"),
	)
	_, err := s.FieldIDs()
	assert.ErrorContains(t, err, "duplicate field id: dup")
}

// TestFieldIDs_EmptyID ensures that a field with an empty ID is caught.
func TestFieldIDs_EmptyID(t *testing.T) {
	s := minimalSchema(dsconfig.ConfigField{Key: "x", ValueType: dsconfig.StringType})
	_, err := s.FieldIDs()
	assert.ErrorContains(t, err, "field id is required")
}

// TestFieldIDs_DuplicateBetweenTopAndItem verifies that an item field
// cannot reuse a top-level field's ID (global uniqueness).
func TestFieldIDs_DuplicateBetweenTopAndItem(t *testing.T) {
	s := minimalSchema(
		validStorageField("conflict", "x"),
		dsconfig.ConfigField{
			ID: "arr", Key: "arr", ValueType: dsconfig.ArrayType, Target: dsconfig.JSONDataTarget,
			Item: &dsconfig.FieldItemSchema{
				ValueType: dsconfig.ObjectType,
				Fields: []dsconfig.ConfigField{
					{ID: "conflict", Key: "k", ValueType: dsconfig.StringType, IsItemField: ptr(true)},
				},
			},
		},
	)
	_, err := s.FieldIDs()
	assert.ErrorContains(t, err, "duplicate field id: conflict")
}

// ============================================================
// DatasourceConfigSchema.ValidateRefs
// ============================================================

// TestValidateRefs_ValidGroupRefs ensures groups referencing existing
// field IDs pass validation.
func TestValidateRefs_ValidGroupRefs(t *testing.T) {
	s := minimalSchema(
		validStorageField("a", "a"),
		validStorageField("b", "b"),
	)
	s.Groups = []dsconfig.ConfigGroup{{ID: "g1", Title: "G", FieldRefs: []string{"a", "b"}}}
	require.NoError(t, s.Validate())
}

// TestValidateRefs_InvalidGroupRef ensures a group referencing a
// non-existent field ID is rejected.
func TestValidateRefs_InvalidGroupRef(t *testing.T) {
	s := minimalSchema(validStorageField("a", "a"))
	s.Groups = []dsconfig.ConfigGroup{{ID: "g1", Title: "G", FieldRefs: []string{"missing"}}}
	assert.ErrorContains(t, s.Validate(), "group g1 references unknown field id: missing")
}

// TestValidateRefs_GroupRefToItemField verifies that groups can reference
// nested item field IDs (not just top-level).
func TestValidateRefs_GroupRefToItemField(t *testing.T) {
	s := &dsconfig.DatasourceConfigSchema{
		SchemaVersion: "v1",
		PluginType:    "test",
		PluginName:    "Test",
		Fields: []dsconfig.ConfigField{
			{
				ID: "arr", Key: "arr", ValueType: dsconfig.ArrayType, Target: dsconfig.JSONDataTarget,
				Item: &dsconfig.FieldItemSchema{
					ValueType: dsconfig.ObjectType,
					Fields: []dsconfig.ConfigField{
						{ID: "arr.item.name", Key: "name", ValueType: dsconfig.StringType, IsItemField: ptr(true)},
					},
				},
			},
		},
		Groups: []dsconfig.ConfigGroup{
			{ID: "g1", Title: "G", FieldRefs: []string{"arr.item.name"}},
		},
	}
	require.NoError(t, s.Validate())
}

// ============================================================
// ConfigField.Validate — identity fields
// ============================================================

// TestFieldValidate_EmptyID ensures that a field without an ID is
// rejected, since ID is the primary reference key.
func TestFieldValidate_EmptyID(t *testing.T) {
	f := dsconfig.ConfigField{Key: "x", ValueType: dsconfig.StringType, Target: dsconfig.JSONDataTarget}
	assert.ErrorContains(t, f.Validate(), "field id is required")
}

// TestFieldValidate_EmptyKey ensures that a field without a key is
// rejected, since key is required for storage mapping.
func TestFieldValidate_EmptyKey(t *testing.T) {
	f := dsconfig.ConfigField{ID: "x", ValueType: dsconfig.StringType, Target: dsconfig.JSONDataTarget}
	assert.ErrorContains(t, f.Validate(), "key is required")
}

// ============================================================
// ConfigField.Validate — valueType
// ============================================================

// TestFieldValidate_InvalidValueType ensures that an unrecognized
// valueType string is rejected.
func TestFieldValidate_InvalidValueType(t *testing.T) {
	f := dsconfig.ConfigField{ID: "x", Key: "x", ValueType: "blob", Target: dsconfig.JSONDataTarget}
	assert.ErrorContains(t, f.Validate(), "invalid valueType")
}

// TestFieldValidate_AllValueTypes verifies that every valid ValueType
// constant passes field validation.
func TestFieldValidate_AllValueTypes(t *testing.T) {
	for _, vt := range []dsconfig.ValueType{
		dsconfig.StringType, dsconfig.NumberType, dsconfig.BooleanType,
		dsconfig.ArrayType, dsconfig.ObjectType, dsconfig.MapType, dsconfig.AnyType,
	} {
		f := validStorageField("x", "x")
		f.ValueType = vt
		if vt == dsconfig.ArrayType || vt == dsconfig.MapType {
			f.Item = &dsconfig.FieldItemSchema{ValueType: dsconfig.StringType}
		}
		assert.NoError(t, f.Validate(), "valueType %s should be valid", vt)
	}
}

// ============================================================
// ConfigField.Validate — target requirement
// ============================================================

// TestFieldValidate_StorageFieldRequiresTarget verifies that a storage
// field (default kind) without a target is rejected.
func TestFieldValidate_StorageFieldRequiresTarget(t *testing.T) {
	f := dsconfig.ConfigField{ID: "x", Key: "x", ValueType: dsconfig.StringType}
	assert.ErrorContains(t, f.Validate(), "target is required for storage fields")
}

// TestFieldValidate_VirtualFieldOmitsTarget confirms that virtual
// fields do not require a target.
func TestFieldValidate_VirtualFieldOmitsTarget(t *testing.T) {
	f := dsconfig.ConfigField{ID: "x", Key: "x", ValueType: dsconfig.StringType, Kind: dsconfig.VirtualField}
	require.NoError(t, f.Validate())
}

// TestFieldValidate_ItemFieldOmitsTarget confirms that item fields
// (isItemField=true) do not require a target.
func TestFieldValidate_ItemFieldOmitsTarget(t *testing.T) {
	f := dsconfig.ConfigField{ID: "x", Key: "x", ValueType: dsconfig.StringType, IsItemField: ptr(true)}
	require.NoError(t, f.Validate())
}

// TestFieldValidate_SectionOnItemFieldRejected ensures that section
// is not allowed on item fields (they inherit storage from the parent).
func TestFieldValidate_SectionOnItemFieldRejected(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.StringType,
		IsItemField: ptr(true), Section: "nested",
	}
	assert.ErrorContains(t, f.Validate(), "section is not allowed on item fields")
}

// TestFieldValidate_SectionOnVirtualFieldRejected ensures that section
// is not allowed on virtual fields (they have no storage target).
func TestFieldValidate_SectionOnVirtualFieldRejected(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.StringType,
		Kind: dsconfig.VirtualField, Section: "nested",
	}
	assert.ErrorContains(t, f.Validate(), "section is not allowed on virtual fields")
}

// TestFieldValidate_SectionOnStorageFieldAllowed confirms that section
// is valid on a normal storage field with a target.
func TestFieldValidate_SectionOnStorageFieldAllowed(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.StringType,
		Target: dsconfig.JSONDataTarget, Section: "oauth2.endpoints",
	}
	require.NoError(t, f.Validate())
}

// TestFieldValidate_InvalidTarget ensures that an unrecognized target
// location string is rejected.
func TestFieldValidate_InvalidTarget(t *testing.T) {
	f := dsconfig.ConfigField{ID: "x", Key: "x", ValueType: dsconfig.StringType, Target: dsconfig.TargetLocation("badTarget")}
	assert.ErrorContains(t, f.Validate(), "invalid target")
}

// TestFieldValidate_AllTargets verifies that every valid TargetLocation
// constant passes field validation.
func TestFieldValidate_AllTargets(t *testing.T) {
	for _, tgt := range []dsconfig.TargetLocation{
		dsconfig.RootTarget, dsconfig.JSONDataTarget, dsconfig.SecureJSONTarget,
	} {
		f := dsconfig.ConfigField{ID: "x", Key: "x", ValueType: dsconfig.StringType, Target: tgt}
		assert.NoError(t, f.Validate(), "target %s should be valid", tgt)
	}
}

// ============================================================
// ConfigField.Validate — kind
// ============================================================

// TestFieldValidate_InvalidKind ensures that an unrecognized kind
// string is rejected even if target is provided.
func TestFieldValidate_InvalidKind(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.StringType,
		Kind: "unknown", Target: dsconfig.JSONDataTarget,
	}
	assert.ErrorContains(t, f.Validate(), "invalid kind")
}

// TestFieldValidate_ValidKinds verifies that both storage and virtual
// field kinds pass validation when properly configured.
func TestFieldValidate_ValidKinds(t *testing.T) {
	for _, k := range []dsconfig.FieldKind{dsconfig.StorageField, dsconfig.VirtualField} {
		f := dsconfig.ConfigField{ID: "x", Key: "x", ValueType: dsconfig.StringType, Kind: k}
		if k == dsconfig.StorageField {
			f.Target = dsconfig.JSONDataTarget
		}
		assert.NoError(t, f.Validate(), "kind %s should be valid", k)
	}
}

// ============================================================
// ConfigField.Validate — array / item
// ============================================================

// TestFieldValidate_ArrayRequiresItem ensures that an array field
// without an item schema is rejected.
func TestFieldValidate_ArrayRequiresItem(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.ArrayType, Target: dsconfig.JSONDataTarget,
	}
	assert.ErrorContains(t, f.Validate(), "item is required for array and map fields")
}

// TestFieldValidate_MapRequiresItem ensures that a map field
// without an item schema is rejected.
func TestFieldValidate_MapRequiresItem(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.MapType, Target: dsconfig.JSONDataTarget,
	}
	assert.ErrorContains(t, f.Validate(), "item is required for array and map fields")
}

// TestFieldValidate_MapWithStringItem confirms that a map field
// with a string item schema passes (Record<string, string>).
func TestFieldValidate_MapWithStringItem(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.MapType, Target: dsconfig.JSONDataTarget,
		Item: &dsconfig.FieldItemSchema{ValueType: dsconfig.StringType},
	}
	require.NoError(t, f.Validate())
}

// TestFieldValidate_MapWithObjectItem confirms that a map field
// with an object item schema passes (Record<string, SomeObj>).
func TestFieldValidate_MapWithObjectItem(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "routes", Key: "routes", ValueType: dsconfig.MapType, Target: dsconfig.JSONDataTarget,
		Item: &dsconfig.FieldItemSchema{
			ValueType: dsconfig.ObjectType,
			Fields: []dsconfig.ConfigField{
				{ID: "routes.item.url", Key: "url", ValueType: dsconfig.StringType, IsItemField: ptr(true)},
			},
		},
	}
	require.NoError(t, f.Validate())
}

// TestFieldValidate_AnyFieldValid confirms that an any-typed field
// passes validation without item.
func TestFieldValidate_AnyFieldValid(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.AnyType, Target: dsconfig.JSONDataTarget,
	}
	require.NoError(t, f.Validate())
}

// TestFieldValidate_AnyItemFieldValid confirms that any-typed item
// fields pass validation.
func TestFieldValidate_AnyItemFieldValid(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.AnyType, IsItemField: ptr(true),
	}
	require.NoError(t, f.Validate())
}

// TestFieldValidate_ArrayWithItem confirms that an array field
// with a valid item schema passes.
func TestFieldValidate_ArrayWithItem(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.ArrayType, Target: dsconfig.JSONDataTarget,
		Item: &dsconfig.FieldItemSchema{ValueType: dsconfig.StringType},
	}
	require.NoError(t, f.Validate())
}

// TestFieldValidate_ItemInvalidValueType ensures that an item schema
// with an unrecognized valueType is rejected.
func TestFieldValidate_ItemInvalidValueType(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.ArrayType, Target: dsconfig.JSONDataTarget,
		Item: &dsconfig.FieldItemSchema{ValueType: "invalid"},
	}
	assert.ErrorContains(t, f.Validate(), "invalid item valueType")
}

// TestFieldValidate_ItemFieldsOnlyForObject ensures that item.fields
// are only allowed when item.valueType is "object". A string array
// with nested fields should be rejected.
func TestFieldValidate_ItemFieldsOnlyForObject(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.ArrayType, Target: dsconfig.JSONDataTarget,
		Item: &dsconfig.FieldItemSchema{
			ValueType: dsconfig.StringType,
			Fields: []dsconfig.ConfigField{
				{ID: "sub", Key: "sub", ValueType: dsconfig.StringType, IsItemField: ptr(true)},
			},
		},
	}
	assert.ErrorContains(t, f.Validate(), "item fields are only allowed when item valueType is object")
}

// TestFieldValidate_ItemFieldMustHaveIsItemField ensures that every
// field inside item.fields must have isItemField=true.
func TestFieldValidate_ItemFieldMustHaveIsItemField(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.ArrayType, Target: dsconfig.JSONDataTarget,
		Item: &dsconfig.FieldItemSchema{
			ValueType: dsconfig.ObjectType,
			Fields: []dsconfig.ConfigField{
				{ID: "sub", Key: "sub", ValueType: dsconfig.StringType},
			},
		},
	}
	assert.ErrorContains(t, f.Validate(), "must have isItemField=true")
}

// TestFieldValidate_ItemFieldValidationPropagates ensures that
// validation errors in nested item fields bubble up through the
// parent field's Validate().
func TestFieldValidate_ItemFieldValidationPropagates(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.ArrayType, Target: dsconfig.JSONDataTarget,
		Item: &dsconfig.FieldItemSchema{
			ValueType: dsconfig.ObjectType,
			Fields: []dsconfig.ConfigField{
				{ID: "sub", Key: "", ValueType: dsconfig.StringType, IsItemField: ptr(true)},
			},
		},
	}
	assert.ErrorContains(t, f.Validate(), "key is required")
}

// TestFieldValidate_ObjectItemWithValidFields confirms that a
// well-formed array-of-objects field passes validation.
func TestFieldValidate_ObjectItemWithValidFields(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "headers", Key: "headers", ValueType: dsconfig.ArrayType, Target: dsconfig.JSONDataTarget,
		Item: &dsconfig.FieldItemSchema{
			ValueType: dsconfig.ObjectType,
			Fields: []dsconfig.ConfigField{
				{ID: "headers.key", Key: "key", ValueType: dsconfig.StringType, IsItemField: ptr(true)},
				{ID: "headers.val", Key: "val", ValueType: dsconfig.StringType, IsItemField: ptr(true)},
			},
		},
	}
	require.NoError(t, f.Validate())
}

// ============================================================
// ConfigField.Validate — validation rules
// ============================================================

// TestFieldValidate_ValidValidationRules confirms that a field with
// multiple well-formed validation rules passes.
func TestFieldValidate_ValidValidationRules(t *testing.T) {
	f := validStorageField("x", "x")
	f.Validations = []dsconfig.FieldValidationRule{
		{Type: dsconfig.PatternValidation, Pattern: "^https?://"},
		{Type: dsconfig.RangeValidation, Min: ptr(0.0), Max: ptr(100.0)},
	}
	require.NoError(t, f.Validate())
}

// TestFieldValidate_InvalidValidationRule ensures that an invalid
// validation rule (missing required fields) causes the parent field
// validation to fail.
func TestFieldValidate_InvalidValidationRule(t *testing.T) {
	f := validStorageField("x", "x")
	f.Validations = []dsconfig.FieldValidationRule{
		{Type: dsconfig.PatternValidation}, // missing pattern
	}
	assert.ErrorContains(t, f.Validate(), "pattern validation requires pattern")
}

// TestFieldValidate_OverrideValidationRulePropagates ensures that
// invalid validation rules inside overrides are caught.
func TestFieldValidate_OverrideValidationRulePropagates(t *testing.T) {
	f := validStorageField("x", "x")
	f.Overrides = []dsconfig.FieldOverride{
		{
			When: "authType == 'basic'",
			Validations: []dsconfig.FieldValidationRule{
				{Type: dsconfig.CustomValidation}, // missing expression
			},
		},
	}
	assert.ErrorContains(t, f.Validate(), "custom validation requires expression")
}

// ============================================================
// ConfigField.Validate — storage mapping integration
// ============================================================

// TestFieldValidate_DirectStorageMapping confirms that a field with
// a valid direct storage mapping passes.
func TestFieldValidate_DirectStorageMapping(t *testing.T) {
	f := validStorageField("x", "x")
	f.Storage = &dsconfig.StorageMapping{Type: dsconfig.DirectMapping}
	require.NoError(t, f.Validate())
}

// TestFieldValidate_InvalidStorageMapping ensures that a field with
// an invalid storage mapping (e.g. computed with no read/write)
// causes validation to fail.
func TestFieldValidate_InvalidStorageMapping(t *testing.T) {
	f := validStorageField("x", "x")
	f.Storage = &dsconfig.StorageMapping{Type: dsconfig.ComputedMapping} // missing read/write
	assert.ErrorContains(t, f.Validate(), "computed mapping requires read or write")
}

// TestFieldValidate_ComputedStorageMappingOnVirtual confirms that
// a virtual field with a computed storage mapping passes.
func TestFieldValidate_ComputedStorageMappingOnVirtual(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "derived", Key: "derived", ValueType: dsconfig.StringType, Kind: dsconfig.VirtualField,
		Storage: &dsconfig.StorageMapping{Type: dsconfig.ComputedMapping, Read: "jsonData.a + jsonData.b"},
	}
	require.NoError(t, f.Validate())
}

// ============================================================
// ConfigField.Path
// ============================================================

// TestFieldPath_WithTarget verifies that Path() returns
// "target.key" when a target is set.
func TestFieldPath_WithTarget(t *testing.T) {
	f := dsconfig.ConfigField{Target: dsconfig.JSONDataTarget, Key: "timeout"}
	assert.Equal(t, "jsonData.timeout", f.Path())
}

// TestFieldPath_WithoutTarget verifies that Path() returns just
// the key when no target is set (e.g. virtual fields).
func TestFieldPath_WithoutTarget(t *testing.T) {
	f := dsconfig.ConfigField{Key: "url"}
	assert.Equal(t, "url", f.Path())
}

// TestFieldPath_RootTarget verifies the path for root-level fields.
func TestFieldPath_RootTarget(t *testing.T) {
	f := dsconfig.ConfigField{Target: dsconfig.RootTarget, Key: "url"}
	assert.Equal(t, "root.url", f.Path())
}

// TestFieldPath_SecureTarget verifies the path for secure fields.
func TestFieldPath_SecureTarget(t *testing.T) {
	f := dsconfig.ConfigField{Target: dsconfig.SecureJSONTarget, Key: "password"}
	assert.Equal(t, "secureJsonData.password", f.Path())
}

// ============================================================
// ValueType.IsValid
// ============================================================

// TestValueType_Valid verifies that all defined ValueType constants
// are recognized as valid.
func TestValueType_Valid(t *testing.T) {
	for _, v := range []dsconfig.ValueType{
		dsconfig.StringType, dsconfig.NumberType, dsconfig.BooleanType,
		dsconfig.ArrayType, dsconfig.ObjectType, dsconfig.MapType, dsconfig.AnyType,
	} {
		assert.True(t, v.IsValid(), "%s should be valid", v)
	}
}

// TestValueType_Invalid verifies that empty strings and unknown
// type names are rejected.
func TestValueType_Invalid(t *testing.T) {
	assert.False(t, dsconfig.ValueType("").IsValid())
	assert.False(t, dsconfig.ValueType("int").IsValid())
	assert.False(t, dsconfig.ValueType("union").IsValid())
}

// ============================================================
// FieldKind.IsValid
// ============================================================

// TestFieldKind_Valid verifies that storage and virtual are
// accepted as valid kinds.
func TestFieldKind_Valid(t *testing.T) {
	assert.True(t, dsconfig.StorageField.IsValid())
	assert.True(t, dsconfig.VirtualField.IsValid())
}

// TestFieldKind_Invalid verifies that empty strings and unknown
// kind names are rejected.
func TestFieldKind_Invalid(t *testing.T) {
	assert.False(t, dsconfig.FieldKind("").IsValid())
	assert.False(t, dsconfig.FieldKind("computed").IsValid())
	assert.False(t, dsconfig.FieldKind("derived").IsValid())
}

// ============================================================
// TargetLocation.IsValid
// ============================================================

// TestTargetLocation_Valid verifies that all defined target
// location constants are recognized as valid.
func TestTargetLocation_Valid(t *testing.T) {
	for _, tgt := range []dsconfig.TargetLocation{
		dsconfig.RootTarget, dsconfig.JSONDataTarget, dsconfig.SecureJSONTarget,
	} {
		assert.True(t, tgt.IsValid(), "%s should be valid", tgt)
	}
}

// TestTargetLocation_Invalid verifies that empty strings and unknown
// target names are rejected.
func TestTargetLocation_Invalid(t *testing.T) {
	assert.False(t, dsconfig.TargetLocation("").IsValid())
	assert.False(t, dsconfig.TargetLocation("metadata").IsValid())
}

// ============================================================
// FieldValidationRule.Validate — pattern
// ============================================================

// TestValidationRule_Pattern_Valid confirms that a pattern rule
// with a non-empty regex string passes.
func TestValidationRule_Pattern_Valid(t *testing.T) {
	r := dsconfig.FieldValidationRule{Type: dsconfig.PatternValidation, Pattern: "^[a-z]+$"}
	require.NoError(t, r.Validate())
}

// TestValidationRule_Pattern_MissingPattern ensures that a pattern
// rule without a pattern string is rejected.
func TestValidationRule_Pattern_MissingPattern(t *testing.T) {
	r := dsconfig.FieldValidationRule{Type: dsconfig.PatternValidation}
	assert.ErrorContains(t, r.Validate(), "pattern validation requires pattern")
}

// ============================================================
// FieldValidationRule.Validate — range
// ============================================================

// TestValidationRule_Range_MinOnly verifies a range rule with
// only a minimum bound.
func TestValidationRule_Range_MinOnly(t *testing.T) {
	r := dsconfig.FieldValidationRule{Type: dsconfig.RangeValidation, Min: ptr(1.0)}
	require.NoError(t, r.Validate())
}

// TestValidationRule_Range_MaxOnly verifies a range rule with
// only a maximum bound.
func TestValidationRule_Range_MaxOnly(t *testing.T) {
	r := dsconfig.FieldValidationRule{Type: dsconfig.RangeValidation, Max: ptr(100.0)}
	require.NoError(t, r.Validate())
}

// TestValidationRule_Range_BothBounds verifies a range rule with
// both min and max.
func TestValidationRule_Range_BothBounds(t *testing.T) {
	r := dsconfig.FieldValidationRule{Type: dsconfig.RangeValidation, Min: ptr(1.0), Max: ptr(300.0)}
	require.NoError(t, r.Validate())
}

// TestValidationRule_Range_NeitherMinNorMax ensures that a range
// rule with no bounds is rejected.
func TestValidationRule_Range_NeitherMinNorMax(t *testing.T) {
	r := dsconfig.FieldValidationRule{Type: dsconfig.RangeValidation}
	assert.ErrorContains(t, r.Validate(), "range validation requires min or max")
}

// ============================================================
// FieldValidationRule.Validate — length
// ============================================================

// TestValidationRule_Length_Valid verifies a length rule with both
// min and max bounds.
func TestValidationRule_Length_Valid(t *testing.T) {
	r := dsconfig.FieldValidationRule{Type: dsconfig.LengthValidation, Min: ptr(1.0), Max: ptr(255.0)}
	require.NoError(t, r.Validate())
}

// TestValidationRule_Length_NeitherMinNorMax ensures that a length
// rule with no bounds is rejected.
func TestValidationRule_Length_NeitherMinNorMax(t *testing.T) {
	r := dsconfig.FieldValidationRule{Type: dsconfig.LengthValidation}
	assert.ErrorContains(t, r.Validate(), "length validation requires min or max")
}

// ============================================================
// FieldValidationRule.Validate — itemCount
// ============================================================

// TestValidationRule_ItemCount_Valid verifies an itemCount rule
// with a maximum.
func TestValidationRule_ItemCount_Valid(t *testing.T) {
	r := dsconfig.FieldValidationRule{Type: dsconfig.ItemCountValidation, Max: ptr(10.0)}
	require.NoError(t, r.Validate())
}

// TestValidationRule_ItemCount_NeitherMinNorMax ensures that an
// itemCount rule with no bounds is rejected.
func TestValidationRule_ItemCount_NeitherMinNorMax(t *testing.T) {
	r := dsconfig.FieldValidationRule{Type: dsconfig.ItemCountValidation}
	assert.ErrorContains(t, r.Validate(), "itemCount validation requires min or max")
}

// ============================================================
// FieldValidationRule.Validate — allowedValues
// ============================================================

// TestValidationRule_AllowedValues_Valid confirms that a rule
// with a non-empty values list passes.
func TestValidationRule_AllowedValues_Valid(t *testing.T) {
	r := dsconfig.FieldValidationRule{
		Type: dsconfig.AllowedValuesValidation, Values: []any{"GET", "POST"},
	}
	require.NoError(t, r.Validate())
}

// TestValidationRule_AllowedValues_Empty ensures that an empty
// values slice is rejected.
func TestValidationRule_AllowedValues_Empty(t *testing.T) {
	r := dsconfig.FieldValidationRule{Type: dsconfig.AllowedValuesValidation, Values: []any{}}
	assert.ErrorContains(t, r.Validate(), "allowedValues validation requires values")
}

// TestValidationRule_AllowedValues_Nil ensures that a nil values
// field is rejected (same as empty).
func TestValidationRule_AllowedValues_Nil(t *testing.T) {
	r := dsconfig.FieldValidationRule{Type: dsconfig.AllowedValuesValidation}
	assert.ErrorContains(t, r.Validate(), "allowedValues validation requires values")
}

// ============================================================
// FieldValidationRule.Validate — custom
// ============================================================

// TestValidationRule_Custom_Valid confirms that a custom rule
// with a non-empty CEL expression passes.
func TestValidationRule_Custom_Valid(t *testing.T) {
	r := dsconfig.FieldValidationRule{
		Type: dsconfig.CustomValidation, Expression: "self.startsWith('http')",
	}
	require.NoError(t, r.Validate())
}

// TestValidationRule_Custom_MissingExpression ensures that a
// custom rule without an expression is rejected.
func TestValidationRule_Custom_MissingExpression(t *testing.T) {
	r := dsconfig.FieldValidationRule{Type: dsconfig.CustomValidation}
	assert.ErrorContains(t, r.Validate(), "custom validation requires expression")
}

// ============================================================
// FieldValidationRule.Validate — unknown type & optional fields
// ============================================================

// TestValidationRule_UnknownType ensures that an unrecognized
// validation rule type is rejected.
func TestValidationRule_UnknownType(t *testing.T) {
	r := dsconfig.FieldValidationRule{Type: "banana"}
	assert.ErrorContains(t, r.Validate(), "unknown validation rule type: banana")
}

// TestValidationRule_WithOptionalIDAndMessage confirms that the
// optional id and message fields do not interfere with validation.
func TestValidationRule_WithOptionalIDAndMessage(t *testing.T) {
	r := dsconfig.FieldValidationRule{
		Type:    dsconfig.PatternValidation,
		ID:      "url-format",
		Message: "Must be a valid URL",
		Pattern: "^https?://",
	}
	require.NoError(t, r.Validate())
}

// ============================================================
// StorageMapping.Validate — direct
// ============================================================

// TestStorageMapping_Direct_Valid confirms that a bare direct
// mapping with no extra fields passes.
func TestStorageMapping_Direct_Valid(t *testing.T) {
	m := dsconfig.StorageMapping{Type: dsconfig.DirectMapping}
	require.NoError(t, m.Validate())
}

// TestStorageMapping_Direct_WithRead ensures that a direct mapping
// with unexpected read/write fields is rejected.
func TestStorageMapping_Direct_WithRead(t *testing.T) {
	m := dsconfig.StorageMapping{Type: dsconfig.DirectMapping, Read: "something"}
	assert.ErrorContains(t, m.Validate(), "direct mapping must not have")
}

// TestStorageMapping_Direct_WithKey ensures that a direct mapping
// with unexpected key field is rejected.
func TestStorageMapping_Direct_WithKey(t *testing.T) {
	m := dsconfig.StorageMapping{
		Type: dsconfig.DirectMapping,
		Key:  &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "x{index}"},
	}
	assert.ErrorContains(t, m.Validate(), "direct mapping must not have")
}

// TestStorageMapping_Direct_WithStartIndex ensures that a direct
// mapping with an unexpected startIndex is rejected.
func TestStorageMapping_Direct_WithStartIndex(t *testing.T) {
	m := dsconfig.StorageMapping{Type: dsconfig.DirectMapping, StartIndex: ptr(1)}
	assert.ErrorContains(t, m.Validate(), "direct mapping must not have")
}

// ============================================================
// StorageMapping.Validate — indexedPair
// ============================================================

// TestStorageMapping_IndexedPair_Valid confirms that a properly
// configured indexed pair mapping passes.
func TestStorageMapping_IndexedPair_Valid(t *testing.T) {
	m := dsconfig.StorageMapping{
		Type:  dsconfig.IndexedPairMapping,
		Key:   &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "httpHeaderName{index}"},
		Value: &dsconfig.MappingField{Target: dsconfig.SecureJSONTarget, Pattern: "httpHeaderValue{index}"},
	}
	require.NoError(t, m.Validate())
}

// TestStorageMapping_IndexedPair_WithStartIndex confirms that
// startIndex is allowed on indexed pair mappings.
func TestStorageMapping_IndexedPair_WithStartIndex(t *testing.T) {
	m := dsconfig.StorageMapping{
		Type:       dsconfig.IndexedPairMapping,
		Key:        &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "k{index}"},
		Value:      &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "v{index}"},
		StartIndex: ptr(1),
	}
	require.NoError(t, m.Validate())
}

// TestStorageMapping_IndexedPair_MissingKey ensures that an indexed
// pair mapping without a key field is rejected.
func TestStorageMapping_IndexedPair_MissingKey(t *testing.T) {
	m := dsconfig.StorageMapping{
		Type:  dsconfig.IndexedPairMapping,
		Value: &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "x{index}"},
	}
	assert.ErrorContains(t, m.Validate(), "indexedPair requires key and value")
}

// TestStorageMapping_IndexedPair_MissingValue ensures that an indexed
// pair mapping without a value field is rejected.
func TestStorageMapping_IndexedPair_MissingValue(t *testing.T) {
	m := dsconfig.StorageMapping{
		Type: dsconfig.IndexedPairMapping,
		Key:  &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "x{index}"},
	}
	assert.ErrorContains(t, m.Validate(), "indexedPair requires key and value")
}

// TestStorageMapping_IndexedPair_WithRead ensures that indexed pair
// mappings with read/write (computed fields) are rejected.
func TestStorageMapping_IndexedPair_WithRead(t *testing.T) {
	m := dsconfig.StorageMapping{
		Type:  dsconfig.IndexedPairMapping,
		Key:   &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "k{i}"},
		Value: &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "v{i}"},
		Read:  "expr",
	}
	assert.ErrorContains(t, m.Validate(), "indexedPair must not have read/write")
}

// TestStorageMapping_IndexedPair_InvalidKeyTarget ensures that an
// invalid target on the key mapping field is caught.
func TestStorageMapping_IndexedPair_InvalidKeyTarget(t *testing.T) {
	m := dsconfig.StorageMapping{
		Type:  dsconfig.IndexedPairMapping,
		Key:   &dsconfig.MappingField{Target: "bad", Pattern: "k{i}"},
		Value: &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "v{i}"},
	}
	assert.ErrorContains(t, m.Validate(), "indexedPair key")
}

// TestStorageMapping_IndexedPair_EmptyValuePattern ensures that an
// empty pattern on the value mapping field is caught.
func TestStorageMapping_IndexedPair_EmptyValuePattern(t *testing.T) {
	m := dsconfig.StorageMapping{
		Type:  dsconfig.IndexedPairMapping,
		Key:   &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "k{i}"},
		Value: &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: ""},
	}
	assert.ErrorContains(t, m.Validate(), "indexedPair value")
}

// ============================================================
// StorageMapping.Validate — computed
// ============================================================

// TestStorageMapping_Computed_ReadOnly confirms a computed mapping
// with only a read expression.
func TestStorageMapping_Computed_ReadOnly(t *testing.T) {
	m := dsconfig.StorageMapping{Type: dsconfig.ComputedMapping, Read: "jsonData.x + jsonData.y"}
	require.NoError(t, m.Validate())
}

// TestStorageMapping_Computed_WriteOnly confirms a computed mapping
// with only a write expression.
func TestStorageMapping_Computed_WriteOnly(t *testing.T) {
	m := dsconfig.StorageMapping{Type: dsconfig.ComputedMapping, Write: "split(value)"}
	require.NoError(t, m.Validate())
}

// TestStorageMapping_Computed_Both confirms a computed mapping
// with both read and write expressions.
func TestStorageMapping_Computed_Both(t *testing.T) {
	m := dsconfig.StorageMapping{Type: dsconfig.ComputedMapping, Read: "r", Write: "w"}
	require.NoError(t, m.Validate())
}

// TestStorageMapping_Computed_Neither ensures that a computed
// mapping with neither read nor write is rejected.
func TestStorageMapping_Computed_Neither(t *testing.T) {
	m := dsconfig.StorageMapping{Type: dsconfig.ComputedMapping}
	assert.ErrorContains(t, m.Validate(), "computed mapping requires read or write")
}

// TestStorageMapping_Computed_WithKey ensures that computed mappings
// reject key/value/startIndex fields meant for indexedPair.
func TestStorageMapping_Computed_WithKey(t *testing.T) {
	m := dsconfig.StorageMapping{
		Type: dsconfig.ComputedMapping,
		Read: "expr",
		Key:  &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "x"},
	}
	assert.ErrorContains(t, m.Validate(), "computed mapping must not have key/value/startIndex")
}

// ============================================================
// StorageMapping.Validate — unknown type
// ============================================================

// TestStorageMapping_UnknownType ensures that an unrecognized
// mapping type string is rejected.
func TestStorageMapping_UnknownType(t *testing.T) {
	m := dsconfig.StorageMapping{Type: "magic"}
	assert.ErrorContains(t, m.Validate(), "unknown mapping type: magic")
}

// ============================================================
// MappingField.Validate
// ============================================================

// TestMappingField_Valid confirms that a mapping field with a
// valid target and non-empty pattern passes.
func TestMappingField_Valid(t *testing.T) {
	m := dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "httpHeaderName{index}"}
	require.NoError(t, m.Validate())
}

// TestMappingField_InvalidTarget ensures that an unrecognized
// target on a mapping field is rejected.
func TestMappingField_InvalidTarget(t *testing.T) {
	m := dsconfig.MappingField{Target: "bad", Pattern: "x{i}"}
	assert.ErrorContains(t, m.Validate(), "invalid target")
}

// TestMappingField_EmptyPattern ensures that a mapping field
// with an empty pattern is rejected.
func TestMappingField_EmptyPattern(t *testing.T) {
	m := dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: ""}
	assert.ErrorContains(t, m.Validate(), "pattern is required")
}

// ============================================================
// Integration: full schema validation
// ============================================================

// TestFullSchemaValidation_Prometheus exercises a realistic
// Prometheus-like schema with multiple field types, groups,
// relationships, array items, storage mappings, and validation
// rules — validating end-to-end correctness.
func TestFullSchemaValidation_Prometheus(t *testing.T) {
	s := &dsconfig.DatasourceConfigSchema{
		SchemaVersion: "v1",
		PluginType:    "prometheus",
		PluginName:    "Prometheus",
		Fields: []dsconfig.ConfigField{
			{
				ID: "url", Key: "url", ValueType: dsconfig.StringType,
				Target: dsconfig.RootTarget, Required: true,
				Validations: []dsconfig.FieldValidationRule{
					{Type: dsconfig.PatternValidation, Pattern: "^https?://", Message: "Must be HTTP(S) URL"},
				},
			},
			{
				ID: "auth.basicAuth", Key: "basicAuth",
				ValueType: dsconfig.BooleanType, Target: dsconfig.RootTarget,
			},
			{
				ID: "auth.basicAuthUser", Key: "basicAuthUser",
				ValueType: dsconfig.StringType, Target: dsconfig.RootTarget,
				RequiredWhen: "auth.basicAuth == true",
			},
			{
				ID: "auth.basicAuthPassword", Key: "basicAuthPassword",
				ValueType: dsconfig.StringType, Target: dsconfig.SecureJSONTarget,
				SemanticType: dsconfig.PasswordType,
			},
			{
				ID: "jsonData.httpMethod", Key: "httpMethod",
				ValueType: dsconfig.StringType, Target: dsconfig.JSONDataTarget,
				Validations: []dsconfig.FieldValidationRule{
					{Type: dsconfig.AllowedValuesValidation, Values: []any{"GET", "POST"}},
				},
				UI: &dsconfig.FieldUI{
					Component: dsconfig.UISelect,
					Options: []dsconfig.FieldOption{
						{Label: "GET", Value: "GET"},
						{Label: "POST", Value: "POST"},
					},
				},
			},
			{
				ID: "jsonData.timeout", Key: "timeout",
				ValueType: dsconfig.NumberType, Target: dsconfig.JSONDataTarget,
				Validations: []dsconfig.FieldValidationRule{
					{Type: dsconfig.RangeValidation, Min: ptr(1.0), Max: ptr(300.0)},
				},
			},
			{
				ID: "httpHeaders", Key: "httpHeaders",
				ValueType: dsconfig.ArrayType, Target: dsconfig.JSONDataTarget,
				Item: &dsconfig.FieldItemSchema{
					ValueType: dsconfig.ObjectType,
					Fields: []dsconfig.ConfigField{
						{ID: "httpHeaders.item.key", Key: "key", ValueType: dsconfig.StringType, IsItemField: ptr(true)},
						{ID: "httpHeaders.item.value", Key: "value", ValueType: dsconfig.StringType, IsItemField: ptr(true)},
					},
				},
				Storage: &dsconfig.StorageMapping{
					Type:  dsconfig.IndexedPairMapping,
					Key:   &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "httpHeaderName{index}"},
					Value: &dsconfig.MappingField{Target: dsconfig.SecureJSONTarget, Pattern: "httpHeaderValue{index}"},
				},
			},
			{
				ID: "derived.hasAuth", Key: "hasAuth",
				ValueType: dsconfig.BooleanType, Kind: dsconfig.VirtualField,
				DependsOn: "auth.basicAuth == true",
			},
		},
		Groups: []dsconfig.ConfigGroup{
			{ID: "connection", Title: "Connection", FieldRefs: []string{"url", "jsonData.httpMethod", "jsonData.timeout"}},
			{ID: "auth", Title: "Authentication", FieldRefs: []string{"auth.basicAuth", "auth.basicAuthUser", "auth.basicAuthPassword"}},
		},
	}

	require.NoError(t, s.Validate())

	// Verify all 10 fields (8 top-level + 2 item fields) are collected
	ids, err := s.FieldIDs()
	require.NoError(t, err)
	assert.Len(t, ids, 10)
}

// ============================================================
// SemanticType.IsValid
// ============================================================

// TestSemanticType_Valid verifies all defined SemanticType constants
// are recognized as valid.
func TestSemanticType_Valid(t *testing.T) {
	for _, st := range []dsconfig.SemanticType{
		dsconfig.URLType, dsconfig.PasswordType, dsconfig.TokenType,
		dsconfig.HostnameType, dsconfig.DurationType,
	} {
		assert.True(t, st.IsValid(), "%s should be valid", st)
	}
}

// TestSemanticType_Invalid verifies that empty and unknown semantic
// types are rejected.
func TestSemanticType_Invalid(t *testing.T) {
	assert.False(t, dsconfig.SemanticType("").IsValid())
	assert.False(t, dsconfig.SemanticType("email").IsValid())
}

// TestFieldValidate_InvalidSemanticType ensures that a field with an
// unrecognized semanticType is rejected during validation.
func TestFieldValidate_InvalidSemanticType(t *testing.T) {
	f := validStorageField("x", "x")
	f.SemanticType = "email"
	assert.ErrorContains(t, f.Validate(), "invalid semanticType")
}

// TestFieldValidate_ValidSemanticType confirms that known semantic
// types pass validation.
func TestFieldValidate_ValidSemanticType(t *testing.T) {
	f := validStorageField("x", "x")
	f.SemanticType = dsconfig.PasswordType
	require.NoError(t, f.Validate())
}

// ============================================================
// Lifecycle.IsValid
// ============================================================

// TestLifecycle_Valid verifies all defined Lifecycle constants
// are recognized as valid.
func TestLifecycle_Valid(t *testing.T) {
	for _, l := range []dsconfig.Lifecycle{
		dsconfig.StableLifecycle, dsconfig.DeprecatedLifecycle, dsconfig.ExperimentalLifecycle,
	} {
		assert.True(t, l.IsValid(), "%s should be valid", l)
	}
}

// TestLifecycle_Invalid verifies that empty and unknown lifecycle
// values are rejected.
func TestLifecycle_Invalid(t *testing.T) {
	assert.False(t, dsconfig.Lifecycle("").IsValid())
	assert.False(t, dsconfig.Lifecycle("beta").IsValid())
}

// TestFieldValidate_InvalidLifecycle ensures that a field with an
// unrecognized lifecycle is rejected during validation.
func TestFieldValidate_InvalidLifecycle(t *testing.T) {
	f := validStorageField("x", "x")
	f.Lifecycle = "beta"
	assert.ErrorContains(t, f.Validate(), "invalid lifecycle")
}

// TestFieldValidate_ValidLifecycle confirms that known lifecycle
// values pass validation.
func TestFieldValidate_ValidLifecycle(t *testing.T) {
	f := validStorageField("x", "x")
	f.Lifecycle = dsconfig.DeprecatedLifecycle
	require.NoError(t, f.Validate())
}

// ============================================================
// UIComponent.IsValid
// ============================================================

// TestUIComponent_Valid verifies all defined UIComponent constants
// are recognized as valid.
func TestUIComponent_Valid(t *testing.T) {
	for _, c := range []dsconfig.UIComponent{
		dsconfig.UIInput, dsconfig.UITextarea, dsconfig.UISelect, dsconfig.UIMultiselect,
		dsconfig.UIRadio, dsconfig.UICheckbox, dsconfig.UISwitch, dsconfig.UICode,
		dsconfig.UIKeyValue, dsconfig.UIList,
	} {
		assert.True(t, c.IsValid(), "%s should be valid", c)
	}
}

// TestUIComponent_Invalid verifies that empty and unknown component
// names are rejected.
func TestUIComponent_Invalid(t *testing.T) {
	assert.False(t, dsconfig.UIComponent("").IsValid())
	assert.False(t, dsconfig.UIComponent("datepicker").IsValid())
}

// TestFieldValidate_InvalidUIComponent ensures that a field with
// an unrecognized UI component is rejected during validation.
func TestFieldValidate_InvalidUIComponent(t *testing.T) {
	f := validStorageField("x", "x")
	f.UI = &dsconfig.FieldUI{Component: "datepicker"}
	assert.ErrorContains(t, f.Validate(), "invalid ui component")
}

// TestFieldValidate_ValidUIComponent confirms that a field with
// a known UI component passes validation.
func TestFieldValidate_ValidUIComponent(t *testing.T) {
	f := validStorageField("x", "x")
	f.UI = &dsconfig.FieldUI{Component: dsconfig.UIInput}
	require.NoError(t, f.Validate())
}

// ============================================================
// UIWidth.IsValid
// ============================================================

// TestUIWidth_Valid verifies that full and half are accepted.
func TestUIWidth_Valid(t *testing.T) {
	assert.True(t, dsconfig.FullWidth.IsValid())
	assert.True(t, dsconfig.HalfWidth.IsValid())
}

// TestUIWidth_Invalid verifies that empty and unknown widths
// are rejected.
func TestUIWidth_Invalid(t *testing.T) {
	assert.False(t, dsconfig.UIWidth("").IsValid())
	assert.False(t, dsconfig.UIWidth("third").IsValid())
}

// TestFieldValidate_InvalidUIWidth ensures that a field with an
// unrecognized UI width is rejected during validation.
func TestFieldValidate_InvalidUIWidth(t *testing.T) {
	f := validStorageField("x", "x")
	f.UI = &dsconfig.FieldUI{Component: dsconfig.UIInput, Width: "third"}
	assert.ErrorContains(t, f.Validate(), "invalid ui width")
}

// TestFieldValidate_ValidUIWidth confirms that a known UI width
// passes validation.
func TestFieldValidate_ValidUIWidth(t *testing.T) {
	f := validStorageField("x", "x")
	f.UI = &dsconfig.FieldUI{Component: dsconfig.UIInput, Width: dsconfig.HalfWidth}
	require.NoError(t, f.Validate())
}

// ============================================================
// ValidateOptionValue — option type checking
// ============================================================

// TestValidateOptionValue_StringMatch confirms that string options
// are accepted for string fields.
func TestValidateOptionValue_StringMatch(t *testing.T) {
	assert.True(t, dsconfig.ValidateOptionValue("hello", dsconfig.StringType))
}

// TestValidateOptionValue_StringMismatch ensures that a numeric
// option is rejected for a string field.
func TestValidateOptionValue_StringMismatch(t *testing.T) {
	assert.False(t, dsconfig.ValidateOptionValue(42, dsconfig.StringType))
}

// TestValidateOptionValue_NumberInt confirms that int values are
// accepted for number fields.
func TestValidateOptionValue_NumberInt(t *testing.T) {
	assert.True(t, dsconfig.ValidateOptionValue(42, dsconfig.NumberType))
}

// TestValidateOptionValue_NumberFloat confirms that float64 values
// are accepted for number fields.
func TestValidateOptionValue_NumberFloat(t *testing.T) {
	assert.True(t, dsconfig.ValidateOptionValue(3.14, dsconfig.NumberType))
}

// TestValidateOptionValue_NumberMismatch ensures that a string
// value is rejected for a number field.
func TestValidateOptionValue_NumberMismatch(t *testing.T) {
	assert.False(t, dsconfig.ValidateOptionValue("not-a-number", dsconfig.NumberType))
}

// TestValidateOptionValue_BoolMatch confirms that bool values are
// accepted for boolean fields.
func TestValidateOptionValue_BoolMatch(t *testing.T) {
	assert.True(t, dsconfig.ValidateOptionValue(true, dsconfig.BooleanType))
}

// TestValidateOptionValue_BoolMismatch ensures that a string value
// is rejected for a boolean field.
func TestValidateOptionValue_BoolMismatch(t *testing.T) {
	assert.False(t, dsconfig.ValidateOptionValue("true", dsconfig.BooleanType))
}

// TestValidateOptionValue_NilRejected confirms that nil values are
// rejected for all field types, matching JSON Schema's "value is required".
func TestValidateOptionValue_NilRejected(t *testing.T) {
	assert.False(t, dsconfig.ValidateOptionValue(nil, dsconfig.StringType))
	assert.False(t, dsconfig.ValidateOptionValue(nil, dsconfig.NumberType))
	assert.False(t, dsconfig.ValidateOptionValue(nil, dsconfig.BooleanType))
	assert.False(t, dsconfig.ValidateOptionValue(nil, dsconfig.ArrayType))
	assert.False(t, dsconfig.ValidateOptionValue(nil, dsconfig.ObjectType))
}

// TestValidateOptionValue_ArrayObjectSkipped confirms that array
// and object fields skip type checking on option values.
func TestValidateOptionValue_ArrayObjectSkipped(t *testing.T) {
	assert.True(t, dsconfig.ValidateOptionValue("anything", dsconfig.ArrayType))
	assert.True(t, dsconfig.ValidateOptionValue(42, dsconfig.ObjectType))
}

// TestFieldValidate_OptionTypeMismatch ensures that a select field
// with an option value that doesn't match the field's valueType is
// rejected during validation.
func TestFieldValidate_OptionTypeMismatch(t *testing.T) {
	f := validStorageField("x", "x")
	f.ValueType = dsconfig.StringType
	f.UI = &dsconfig.FieldUI{
		Component: dsconfig.UISelect,
		Options: []dsconfig.FieldOption{
			{Label: "Good", Value: "good"},
			{Label: "Bad", Value: 42}, // mismatch: number in string field
		},
	}
	assert.ErrorContains(t, f.Validate(), "option[1] value type mismatch")
}

// TestFieldValidate_OptionTypeValid confirms that a select field
// with correctly-typed option values passes validation.
func TestFieldValidate_OptionTypeValid(t *testing.T) {
	f := validStorageField("x", "x")
	f.ValueType = dsconfig.StringType
	f.UI = &dsconfig.FieldUI{
		Component: dsconfig.UISelect,
		Options: []dsconfig.FieldOption{
			{Label: "GET", Value: "GET"},
			{Label: "POST", Value: "POST"},
		},
	}
	require.NoError(t, f.Validate())
}

// TestFieldValidate_NumberOptionTypeValid confirms that number
// field options with numeric values pass validation.
func TestFieldValidate_NumberOptionTypeValid(t *testing.T) {
	f := validStorageField("x", "x")
	f.ValueType = dsconfig.NumberType
	f.UI = &dsconfig.FieldUI{
		Component: dsconfig.UISelect,
		Options: []dsconfig.FieldOption{
			{Label: "Low", Value: 1},
			{Label: "High", Value: 100},
		},
	}
	require.NoError(t, f.Validate())
}

// ============================================================
// JSON round-trip compatibility
// ============================================================

// TestJSONRoundTrip_MinimalSchema verifies that a minimal schema
// survives JSON marshal/unmarshal and still validates.
func TestJSONRoundTrip_MinimalSchema(t *testing.T) {
	s := minimalSchema(validStorageField("url", "url"))
	require.NoError(t, s.Validate())

	data, err := json.Marshal(s)
	require.NoError(t, err)

	var decoded dsconfig.DatasourceConfigSchema
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.NoError(t, decoded.Validate())

	assert.Equal(t, s.SchemaVersion, decoded.SchemaVersion)
	assert.Equal(t, s.PluginType, decoded.PluginType)
	assert.Len(t, decoded.Fields, 1)
	assert.Equal(t, "url", decoded.Fields[0].ID)
}

// TestJSONRoundTrip_FullSchema verifies that a complex schema with
// all feature areas (groups, relationships, validations, overrides,
// storage mappings, item fields) survives JSON round-trip.
func TestJSONRoundTrip_FullSchema(t *testing.T) {
	s := &dsconfig.DatasourceConfigSchema{
		SchemaVersion: "v1",
		PluginType:    "test",
		PluginName:    "Test Plugin",
		DocURL:        "https://example.com/docs",
		Fields: []dsconfig.ConfigField{
			{
				ID: "url", Key: "url", ValueType: dsconfig.StringType,
				Target: dsconfig.RootTarget, Required: true,
				SemanticType: dsconfig.URLType,
				Lifecycle:    dsconfig.StableLifecycle,
				Validations: []dsconfig.FieldValidationRule{
					{Type: dsconfig.PatternValidation, Pattern: "^https?://", ID: "url-check", Message: "Must be URL"},
				},
				UI: &dsconfig.FieldUI{Component: dsconfig.UIInput, Width: dsconfig.FullWidth, Placeholder: "https://..."},
			},
			{
				ID: "method", Key: "httpMethod", ValueType: dsconfig.StringType,
				Target: dsconfig.JSONDataTarget,
				Validations: []dsconfig.FieldValidationRule{
					{Type: dsconfig.AllowedValuesValidation, Values: []any{"GET", "POST"}},
				},
				UI: &dsconfig.FieldUI{
					Component: dsconfig.UISelect,
					Options: []dsconfig.FieldOption{
						{Label: "GET", Value: "GET"},
						{Label: "POST", Value: "POST"},
					},
				},
				Overrides: []dsconfig.FieldOverride{
					{When: "version == 'v2'", DefaultValue: "POST"},
				},
			},
			{
				ID: "headers", Key: "headers", ValueType: dsconfig.ArrayType,
				Target: dsconfig.JSONDataTarget,
				Item: &dsconfig.FieldItemSchema{
					ValueType: dsconfig.ObjectType,
					Fields: []dsconfig.ConfigField{
						{ID: "headers.item.k", Key: "key", ValueType: dsconfig.StringType, IsItemField: ptr(true)},
						{ID: "headers.item.v", Key: "value", ValueType: dsconfig.StringType, IsItemField: ptr(true)},
					},
				},
				Storage: &dsconfig.StorageMapping{
					Type:  dsconfig.IndexedPairMapping,
					Key:   &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "headerName{index}"},
					Value: &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "headerValue{index}"},
				},
			},
		},
		Groups: []dsconfig.ConfigGroup{
			{ID: "conn", Title: "Connection", FieldRefs: []string{"url", "method"}},
		},
	}

	require.NoError(t, s.Validate())

	data, err := json.Marshal(s)
	require.NoError(t, err)

	var decoded dsconfig.DatasourceConfigSchema
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.NoError(t, decoded.Validate())

	assert.Equal(t, s.PluginType, decoded.PluginType)
	assert.Len(t, decoded.Fields, 3)
	assert.Len(t, decoded.Groups, 1)
	assert.Equal(t, dsconfig.IndexedPairMapping, decoded.Fields[2].Storage.Type)
}

// TestJSONRoundTrip_ValidationRules verifies that all validation
// rule types survive JSON serialization with correct discriminators.
func TestJSONRoundTrip_ValidationRules(t *testing.T) {
	rules := []dsconfig.FieldValidationRule{
		{Type: dsconfig.PatternValidation, Pattern: "^[a-z]+$", Message: "lowercase only"},
		{Type: dsconfig.RangeValidation, Min: ptr(0.0), Max: ptr(100.0)},
		{Type: dsconfig.LengthValidation, Min: ptr(1.0)},
		{Type: dsconfig.ItemCountValidation, Max: ptr(10.0)},
		{Type: dsconfig.AllowedValuesValidation, Values: []any{"a", "b"}},
		{Type: dsconfig.CustomValidation, Expression: "self.size() > 0"},
	}

	data, err := json.Marshal(rules)
	require.NoError(t, err)

	var decoded []dsconfig.FieldValidationRule
	require.NoError(t, json.Unmarshal(data, &decoded))

	require.Len(t, decoded, 6)
	for i := range decoded {
		assert.Equal(t, rules[i].Type, decoded[i].Type)
		require.NoError(t, decoded[i].Validate())
	}
}

// TestJSONRoundTrip_StorageMappingTypes verifies that all three
// storage mapping types survive JSON serialization.
func TestJSONRoundTrip_StorageMappingTypes(t *testing.T) {
	mappings := []dsconfig.StorageMapping{
		{Type: dsconfig.DirectMapping},
		{
			Type:  dsconfig.IndexedPairMapping,
			Key:   &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "k{i}"},
			Value: &dsconfig.MappingField{Target: dsconfig.JSONDataTarget, Pattern: "v{i}"},
		},
		{Type: dsconfig.ComputedMapping, Read: "expr"},
	}

	for _, m := range mappings {
		data, err := json.Marshal(m)
		require.NoError(t, err)

		var decoded dsconfig.StorageMapping
		require.NoError(t, json.Unmarshal(data, &decoded))
		assert.Equal(t, m.Type, decoded.Type)
		require.NoError(t, decoded.Validate())
	}
}

// ============================================================
// Example schemas — Loki & Tempo
// ============================================================

// TestExampleSchema_Loki validates a Loki-like datasource schema
// with derived fields (array of objects), basic auth, and groups.
func TestExampleSchema_Loki(t *testing.T) {
	s := &dsconfig.DatasourceConfigSchema{
		SchemaVersion: "v1",
		PluginType:    "loki",
		PluginName:    "Loki",
		Fields: []dsconfig.ConfigField{
			{
				ID: "url", Key: "url", ValueType: dsconfig.StringType,
				Target: dsconfig.RootTarget, Required: true,
				SemanticType: dsconfig.URLType,
			},
			{
				ID: "jsonData.maxLines", Key: "maxLines", ValueType: dsconfig.StringType,
				Target: dsconfig.JSONDataTarget,
			},
			{
				ID: "jsonData.derivedFields", Key: "derivedFields",
				ValueType: dsconfig.ArrayType, Target: dsconfig.JSONDataTarget,
				Item: &dsconfig.FieldItemSchema{
					ValueType: dsconfig.ObjectType,
					Fields: []dsconfig.ConfigField{
						{ID: "derivedFields.item.name", Key: "name", ValueType: dsconfig.StringType, IsItemField: ptr(true)},
						{ID: "derivedFields.item.matcherRegex", Key: "matcherRegex", ValueType: dsconfig.StringType, IsItemField: ptr(true)},
						{ID: "derivedFields.item.url", Key: "url", ValueType: dsconfig.StringType, IsItemField: ptr(true),
							SemanticType: dsconfig.URLType},
					},
				},
			},
			{
				ID: "jsonData.timeout", Key: "timeout", ValueType: dsconfig.NumberType,
				Target: dsconfig.JSONDataTarget,
				Validations: []dsconfig.FieldValidationRule{
					{Type: dsconfig.RangeValidation, Min: ptr(1.0), Max: ptr(600.0)},
				},
			},
		},
		Groups: []dsconfig.ConfigGroup{
			{ID: "connection", Title: "Connection", FieldRefs: []string{"url", "jsonData.timeout"}},
			{ID: "derived", Title: "Derived Fields", FieldRefs: []string{"jsonData.derivedFields"}},
		},
	}
	require.NoError(t, s.Validate())

	ids, err := s.FieldIDs()
	require.NoError(t, err)
	assert.Len(t, ids, 7) // 4 top-level + 3 item fields
}

// TestExampleSchema_Tempo validates a Tempo-like datasource schema
// with nested config (service map), virtual fields, and custom
// validation rules.
func TestExampleSchema_Tempo(t *testing.T) {
	s := &dsconfig.DatasourceConfigSchema{
		SchemaVersion: "v1",
		PluginType:    "tempo",
		PluginName:    "Tempo",
		Fields: []dsconfig.ConfigField{
			{
				ID: "url", Key: "url", ValueType: dsconfig.StringType,
				Target: dsconfig.RootTarget, Required: true,
				SemanticType: dsconfig.URLType,
				Lifecycle:    dsconfig.StableLifecycle,
			},
			{
				ID: "jsonData.serviceMap.datasourceUid", Key: "serviceMap.datasourceUid",
				ValueType: dsconfig.StringType, Target: dsconfig.JSONDataTarget,
			},
			{
				ID: "jsonData.search.hide", Key: "search.hide",
				ValueType: dsconfig.BooleanType, Target: dsconfig.JSONDataTarget,
			},
			{
				ID: "jsonData.nodeGraph.enabled", Key: "nodeGraph.enabled",
				ValueType: dsconfig.BooleanType, Target: dsconfig.JSONDataTarget,
			},
			{
				ID: "jsonData.streamingEnabled.search", Key: "streamingEnabled.search",
				ValueType: dsconfig.BooleanType, Target: dsconfig.JSONDataTarget,
			},
			{
				ID: "jsonData.streamingEnabled.metrics", Key: "streamingEnabled.metrics",
				ValueType: dsconfig.BooleanType, Target: dsconfig.JSONDataTarget,
			},
			{
				ID: "derived.hasServiceMap", Key: "hasServiceMap",
				ValueType: dsconfig.BooleanType, Kind: dsconfig.VirtualField,
				Lifecycle: dsconfig.ExperimentalLifecycle,
				DependsOn: "jsonData.serviceMap.datasourceUid != ''",
			},
		},
		Groups: []dsconfig.ConfigGroup{
			{ID: "connection", Title: "Connection", FieldRefs: []string{"url"}},
			{ID: "features", Title: "Features", FieldRefs: []string{
				"jsonData.nodeGraph.enabled",
				"jsonData.streamingEnabled.search",
				"jsonData.streamingEnabled.metrics",
			}},
		},
	}
	require.NoError(t, s.Validate())
}

// TestExampleSchema_MySQL validates a MySQL-like datasource schema
// with secure fields, conditional requirements, and legacy patterns.
func TestExampleSchema_MySQL(t *testing.T) {
	s := &dsconfig.DatasourceConfigSchema{
		SchemaVersion: "v1",
		PluginType:    "mysql",
		PluginName:    "MySQL",
		Fields: []dsconfig.ConfigField{
			{
				ID: "url", Key: "url", ValueType: dsconfig.StringType,
				Target: dsconfig.RootTarget, Required: true,
				SemanticType: dsconfig.URLType,
				Validations: []dsconfig.FieldValidationRule{
					{Type: dsconfig.PatternValidation, Pattern: ".+:\\d+", Message: "Must include host:port"},
				},
			},
			{
				ID: "root.database", Key: "database", ValueType: dsconfig.StringType,
				Target: dsconfig.RootTarget,
			},
			{
				ID: "root.user", Key: "user", ValueType: dsconfig.StringType,
				Target: dsconfig.RootTarget,
			},
			{
				ID: "secureJsonData.password", Key: "password", ValueType: dsconfig.StringType,
				Target: dsconfig.SecureJSONTarget, SemanticType: dsconfig.PasswordType,
				RequiredWhen: "root.user != ''",
			},
			{
				ID: "jsonData.maxOpenConns", Key: "maxOpenConns", ValueType: dsconfig.NumberType,
				Target: dsconfig.JSONDataTarget,
				Validations: []dsconfig.FieldValidationRule{
					{Type: dsconfig.RangeValidation, Min: ptr(0.0), Max: ptr(100.0)},
				},
			},
			{
				ID: "jsonData.connMaxLifetime", Key: "connMaxLifetime", ValueType: dsconfig.NumberType,
				Target:       dsconfig.JSONDataTarget,
				SemanticType: dsconfig.DurationType,
			},
			{
				ID: "jsonData.tlsAuth", Key: "tlsAuth", ValueType: dsconfig.BooleanType,
				Target: dsconfig.JSONDataTarget,
			},
			{
				ID: "secureJsonData.tlsCACert", Key: "tlsCACert", ValueType: dsconfig.StringType,
				Target:    dsconfig.SecureJSONTarget,
				DependsOn: "jsonData.tlsAuth == true",
				UI:        &dsconfig.FieldUI{Component: dsconfig.UITextarea, Rows: 5},
			},
		},
		Groups: []dsconfig.ConfigGroup{
			{ID: "connection", Title: "Connection", FieldRefs: []string{"url", "root.database"}},
			{ID: "auth", Title: "Authentication", FieldRefs: []string{"root.user", "secureJsonData.password"}},
			{ID: "tls", Title: "TLS / SSL", FieldRefs: []string{"jsonData.tlsAuth", "secureJsonData.tlsCACert"}},
		},
	}
	require.NoError(t, s.Validate())

	ids, err := s.FieldIDs()
	require.NoError(t, err)
	assert.Len(t, ids, 8)
}

// ============================================================
// FieldEffect.Validate
// ============================================================

// TestFieldEffect_Valid confirms that a well-formed effect passes.
func TestFieldEffect_Valid(t *testing.T) {
	e := dsconfig.FieldEffect{When: "value == 'basic-auth'", Set: map[string]any{"auth.basicAuth": true}}
	require.NoError(t, e.Validate())
}

// TestFieldEffect_EmptyWhen ensures an effect without a when is rejected.
func TestFieldEffect_EmptyWhen(t *testing.T) {
	e := dsconfig.FieldEffect{Set: map[string]any{"a": true}}
	assert.ErrorContains(t, e.Validate(), "effect when is required")
}

// TestFieldEffect_EmptySet ensures an effect with no set entries is rejected.
func TestFieldEffect_EmptySet(t *testing.T) {
	e := dsconfig.FieldEffect{When: "value == 'x'", Set: map[string]any{}}
	assert.ErrorContains(t, e.Validate(), "effect set must not be empty")
}

// TestFieldEffect_NilSet ensures an effect with nil set is rejected.
func TestFieldEffect_NilSet(t *testing.T) {
	e := dsconfig.FieldEffect{When: "value == 'x'"}
	assert.ErrorContains(t, e.Validate(), "effect set must not be empty")
}

// TestFieldValidate_PropagatesEffectError ensures that invalid effects
// on a field bubble up through field validation.
func TestFieldValidate_PropagatesEffectError(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.StringType, Kind: dsconfig.VirtualField,
		Effects: []dsconfig.FieldEffect{{When: "", Set: map[string]any{"a": true}}},
	}
	assert.ErrorContains(t, f.Validate(), "invalid effect[0]")
}

// TestFieldValidate_ValidEffects confirms a field with well-formed effects passes.
func TestFieldValidate_ValidEffects(t *testing.T) {
	f := dsconfig.ConfigField{
		ID: "x", Key: "x", ValueType: dsconfig.StringType, Kind: dsconfig.VirtualField,
		Effects: []dsconfig.FieldEffect{
			{When: "value == 'a'", Set: map[string]any{"y": true}},
			{When: "value == 'b'", Set: map[string]any{"y": false}},
		},
	}
	require.NoError(t, f.Validate())
}

// TestValidateRefs_EffectSetRefsValid ensures effect set keys that
// reference existing field IDs pass validation.
func TestValidateRefs_EffectSetRefsValid(t *testing.T) {
	s := &dsconfig.DatasourceConfigSchema{
		SchemaVersion: "v1", PluginType: "test", PluginName: "Test",
		Fields: []dsconfig.ConfigField{
			{
				ID: "selector", Key: "selector", ValueType: dsconfig.StringType,
				Kind: dsconfig.VirtualField,
				Effects: []dsconfig.FieldEffect{
					{When: "value == 'on'", Set: map[string]any{"target": true}},
				},
			},
			{ID: "target", Key: "target", ValueType: dsconfig.BooleanType, Target: dsconfig.JSONDataTarget},
		},
	}
	require.NoError(t, s.Validate())
}

// TestValidateRefs_EffectSetRefsUnknown ensures effect set keys that
// reference non-existent field IDs are rejected.
func TestValidateRefs_EffectSetRefsUnknown(t *testing.T) {
	s := &dsconfig.DatasourceConfigSchema{
		SchemaVersion: "v1", PluginType: "test", PluginName: "Test",
		Fields: []dsconfig.ConfigField{
			{
				ID: "selector", Key: "selector", ValueType: dsconfig.StringType,
				Kind: dsconfig.VirtualField,
				Effects: []dsconfig.FieldEffect{
					{When: "value == 'on'", Set: map[string]any{"ghost": true}},
				},
			},
		},
	}
	assert.ErrorContains(t, s.Validate(), "effect[0].set references unknown field id: ghost")
}

// TestExampleSchema_AuthSelector validates the full auth-selector
// pattern: virtual dropdown + effects + dependent storage fields.
func TestExampleSchema_AuthSelector(t *testing.T) {
	s := &dsconfig.DatasourceConfigSchema{
		SchemaVersion: "v1", PluginType: "test-auth", PluginName: "Auth Test",
		Fields: []dsconfig.ConfigField{
			{
				ID: "url", Key: "url", ValueType: dsconfig.StringType,
				Target: dsconfig.RootTarget, Required: true,
			},
			{
				ID: "auth.method", Key: "authMethod", Label: "Authentication method",
				ValueType: dsconfig.StringType, Kind: dsconfig.VirtualField,
				DefaultValue: "no-auth",
				Validations: []dsconfig.FieldValidationRule{
					{Type: dsconfig.AllowedValuesValidation, Values: []any{"no-auth", "basic-auth", "forward-oauth"}},
				},
				UI: &dsconfig.FieldUI{
					Component: dsconfig.UISelect,
					Options: []dsconfig.FieldOption{
						{Label: "No Authentication", Value: "no-auth"},
						{Label: "Basic authentication", Value: "basic-auth"},
						{Label: "Forward OAuth Identity", Value: "forward-oauth"},
					},
				},
				Storage: &dsconfig.StorageMapping{
					Type: dsconfig.ComputedMapping,
					Read: "root.basicAuth == true ? 'basic-auth' : (jsonData.oauthPassThru == true ? 'forward-oauth' : 'no-auth')",
				},
				Effects: []dsconfig.FieldEffect{
					{When: "value == 'no-auth'", Set: map[string]any{"auth.basicAuth": false, "auth.oauthPassThru": false}},
					{When: "value == 'basic-auth'", Set: map[string]any{"auth.basicAuth": true, "auth.oauthPassThru": false}},
					{When: "value == 'forward-oauth'", Set: map[string]any{"auth.basicAuth": false, "auth.oauthPassThru": true}},
				},
			},
			{
				ID: "auth.basicAuth", Key: "basicAuth", ValueType: dsconfig.BooleanType,
				Target: dsconfig.RootTarget, DefaultValue: false,
			},
			{
				ID: "auth.oauthPassThru", Key: "oauthPassThru", ValueType: dsconfig.BooleanType,
				Target: dsconfig.JSONDataTarget, DefaultValue: false,
			},
			{
				ID: "auth.basicAuthUser", Key: "basicAuthUser", ValueType: dsconfig.StringType,
				Target:    dsconfig.RootTarget,
				DependsOn: "auth.method == 'basic-auth'", RequiredWhen: "auth.method == 'basic-auth'",
			},
			{
				ID: "auth.basicAuthPassword", Key: "basicAuthPassword", ValueType: dsconfig.StringType,
				Target: dsconfig.SecureJSONTarget, SemanticType: dsconfig.PasswordType,
				DependsOn: "auth.method == 'basic-auth'",
			},
		},
		Groups: []dsconfig.ConfigGroup{
			{ID: "connection", Title: "Connection", FieldRefs: []string{"url"}},
			{ID: "auth", Title: "Authentication", FieldRefs: []string{"auth.method", "auth.basicAuthUser", "auth.basicAuthPassword"}},
		},
	}
	require.NoError(t, s.Validate())

	ids, err := s.FieldIDs()
	require.NoError(t, err)
	assert.Len(t, ids, 6)
}
