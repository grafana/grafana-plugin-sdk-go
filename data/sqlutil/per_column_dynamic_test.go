package sqlutil

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPerColumnDynamic verifies that DynamicPerColumn applies per-column, not per-frame.
func TestPerColumnDynamic(t *testing.T) {
	// Define converters: one DynamicPerColumn converter, multiple static converters
	convs := []Converter{
		{
			Name:             "DynamicPerColumn VARIANT converter",
			InputTypeName:    "VARIANT",
			DynamicPerColumn: true, // New field - only affects matched columns
		},
		{
			Name:          "INT converter",
			InputTypeName: "INT",
		},
		{
			Name:          "BIGINT converter",
			InputTypeName: "BIGINT",
		},
	}

	// Zero-value *sql.ColumnType has DatabaseTypeName()=="", so no match expected
	zero := &sql.ColumnType{}
	colTypes := []*sql.ColumnType{zero, zero, zero}

	dynamicIndices, filtered := findDynamicPerColumnConverters(colTypes, convs)
	assert.Equal(t, 0, len(dynamicIndices), "Should have no dynamic columns (no matching types)")
	assert.Equal(t, 2, len(filtered), "DynamicPerColumn converter should be filtered out")
	for _, conv := range filtered {
		assert.False(t, conv.DynamicPerColumn, "All filtered converters should be non-dynamic")
	}
}

// TestPerColumnDynamicMatchesCorrectColumns verifies that DynamicPerColumn actually matches columns
// by using a converter whose InputTypeName matches the column's DatabaseTypeName.
func TestPerColumnDynamicMatchesCorrectColumns(t *testing.T) {
	// Use InputTypeName "" which matches the zero-value ColumnType's DatabaseTypeName()
	convs := []Converter{
		{Name: "dynamic-empty-type", InputTypeName: "", DynamicPerColumn: true},
		{Name: "static", InputTypeName: "STATIC"},
	}

	zero := &sql.ColumnType{}
	colTypes := []*sql.ColumnType{zero, zero}

	dynamicIndices, filtered := findDynamicPerColumnConverters(colTypes, convs)
	assert.Equal(t, 2, len(dynamicIndices), "Both columns should be dynamic (DatabaseTypeName==\"\" matches InputTypeName==\"\")")
	assert.True(t, dynamicIndices[0], "Column 0 should be dynamic")
	assert.True(t, dynamicIndices[1], "Column 1 should be dynamic")
	assert.Equal(t, 1, len(filtered), "Only the non-dynamic converter should remain")
	assert.Equal(t, "static", filtered[0].Name)
}

// TestLegacyDynamicBackwardCompatibility verifies the old Dynamic field still works
func TestLegacyDynamicBackwardCompatibility(t *testing.T) {
	kind := &sql.ColumnType{}
	types := []*sql.ColumnType{kind, kind}
	
	// Use old Dynamic field
	converters := []Converter{
		{
			Name:          "Legacy dynamic converter",
			InputTypeName: "VARIANT",
			Dynamic:       true, // Old field - affects entire frame
		},
	}

	// Should use legacy removeDynamicConverter
	isDynamic, filtered := removeDynamicConverter(converters)
	assert.True(t, isDynamic, "Should detect legacy dynamic converter")
	assert.Equal(t, 0, len(filtered), "Legacy dynamic converter should be filtered out")

	// Mock data
	mockData := [][]interface{}{
		{"text1", float64(123)},
		{"text2", float64(456)},
	}
	
	mock := &MockRows{
		data:  mockData,
		index: -1,
	}
	rows := Rows{itr: mock}
	
	// Test legacy frameDynamic - all columns should use runtime inference
	frame, err := frameDynamic(rows, 100, types, []Converter{})
	require.NoError(t, err)
	require.NotNil(t, frame)
	
	// Both columns should be inferred from runtime data
	assert.Equal(t, 2, len(frame.Fields), "Should have 2 fields")
	assert.Equal(t, data.FieldTypeNullableString, frame.Fields[0].Type(), "First column should be string (legacy)")
	assert.Equal(t, data.FieldTypeNullableFloat64, frame.Fields[1].Type(), "Second column should be float64 (legacy)")
}

// TestDynamicPerColumnVsLegacyDynamic verifies the difference in behavior
func TestDynamicPerColumnVsLegacyDynamic(t *testing.T) {
	t.Run("Legacy Dynamic affects entire frame", func(t *testing.T) {
		converters := []Converter{
			{Name: "legacy", InputTypeName: "VARIANT", Dynamic: true},
		}
		
		isDynamic, _ := removeDynamicConverter(converters)
		assert.True(t, isDynamic, "Legacy Dynamic should trigger frame-wide inference")
	})

	t.Run("DynamicPerColumn affects only matched columns", func(t *testing.T) {
		// Use InputTypeName "" to match zero-value ColumnType, and "STATIC" to not match
		convs := []Converter{
			{Name: "dynamic", InputTypeName: "", DynamicPerColumn: true},  // matches column with empty db type
			{Name: "static", InputTypeName: "STATIC", DynamicPerColumn: true}, // does not match
		}

		zero := &sql.ColumnType{}
		colTypes := []*sql.ColumnType{zero}

		dynamicIndices, _ := findDynamicPerColumnConverters(colTypes, convs)
		// Only the empty-type converter matches the zero-value ColumnType
		assert.Equal(t, 1, len(dynamicIndices), "DynamicPerColumn only affects matched columns")
		assert.True(t, dynamicIndices[0], "Column 0 should be dynamic (matched by empty type name)")
	})
}

// TestFrameHybridStaticColumnInference verifies that static (non-dynamic) columns
// in frameHybrid get their types inferred from runtime data, same as frameDynamic.
func TestFrameHybridStaticColumnInference(t *testing.T) {
	zero := &sql.ColumnType{}
	types := []*sql.ColumnType{zero, zero}

	// Column 1 is dynamic, column 0 is static
	dynamicIndices := map[int]bool{1: true}

	mockData := [][]interface{}{
		{float64(42), "variant_value"},
	}
	mock := &MockRows{data: mockData, index: -1}
	rows := Rows{itr: mock}

	frame, err := frameHybrid(rows, 100, types, []Converter{}, dynamicIndices)
	require.NoError(t, err)
	require.NotNil(t, frame)
	require.Equal(t, 2, len(frame.Fields))

	// Both columns inferred from runtime data: float64→NullableFloat64, string→NullableString
	assert.Equal(t, data.FieldTypeNullableFloat64, frame.Fields[0].Type(), "Static column inferred as float64")
	assert.Equal(t, data.FieldTypeNullableString, frame.Fields[1].Type(), "Dynamic column inferred as string")
}

// TestDynamicConverterWithRegex verifies that DynamicPerColumn converters matched by regex work
func TestDynamicConverterWithRegex(t *testing.T) {
	converter := Converter{
		Name:             "DynamicPerColumn VARIANT converter",
		InputTypeRegex:   regexp.MustCompile("^VARIANT"),
		DynamicPerColumn: true,
	}

	assert.True(t, converter.DynamicPerColumn, "Converter should be per-column dynamic")
	assert.NotNil(t, converter.InputTypeRegex, "Converter should have regex")
	
	// Test regex matching
	assert.True(t, converter.InputTypeRegex.MatchString("VARIANT"), "Should match VARIANT")
	assert.True(t, converter.InputTypeRegex.MatchString("VARIANT_JSON"), "Should match VARIANT_JSON")
	assert.False(t, converter.InputTypeRegex.MatchString("INT"), "Should not match INT")
}

// TestAllStaticConverters verifies behavior with no dynamic converters
func TestAllStaticConverters(t *testing.T) {
	converters := []Converter{
		NullInt32Converter,
		NullStringConverter,
		NullBoolConverter,
	}

	// Legacy check
	isDynamic, filtered := removeDynamicConverter(converters)
	assert.False(t, isDynamic, "Should not detect any legacy dynamic converters")
	assert.Equal(t, len(converters), len(filtered), "All converters should be preserved")

	// New check
	kind := &sql.ColumnType{}
	colTypes := []*sql.ColumnType{kind}
	dynamicIndices, filtered2 := findDynamicPerColumnConverters(colTypes, converters)
	assert.Equal(t, 0, len(dynamicIndices), "Should not detect any per-column dynamic converters")
	assert.Equal(t, len(converters), len(filtered2), "All converters should be preserved")
}

// TestBackwardCompatibility_OldDynamicBehavior tests that the deprecated removeDynamicConverter still works
func TestBackwardCompatibility_OldDynamicBehavior(t *testing.T) {
	converters := []Converter{
		{Name: "static1", Dynamic: false},
		{Name: "dynamic1", Dynamic: true},
		{Name: "static2", Dynamic: false},
		{Name: "dynamic2", Dynamic: true},
	}

	isDynamic, filtered := removeDynamicConverter(converters)
	
	assert.True(t, isDynamic, "Should detect legacy dynamic converter")
	assert.Equal(t, 2, len(filtered), "Should filter out all legacy dynamic converters")
	assert.Equal(t, "static1", filtered[0].Name)
	assert.Equal(t, "static2", filtered[1].Name)
}

// TestFrameDynamicWithMixedTypes tests the full dynamic frame flow
func TestFrameDynamicWithMixedTypes(t *testing.T) {
	kind := &sql.ColumnType{}
	types := []*sql.ColumnType{kind, kind}
	
	// Mock data: one string column, one numeric column
	mockData := [][]interface{}{
		{"text1", float64(123)},
		{"text2", float64(456)},
	}
	
	mock := &MockRows{
		data:  mockData,
		index: -1,
	}
	rows := Rows{itr: mock}
	
	converters := []Converter{}
	
	// Test frameDynamic (legacy behavior)
	frame, err := frameDynamic(rows, 100, types, converters)
	require.NoError(t, err)
	require.NotNil(t, frame)
	
	// Verify frame structure
	assert.Equal(t, 2, len(frame.Fields), "Should have 2 fields")
	assert.Equal(t, 2, frame.Rows(), "Should have 2 rows")
	
	// Verify types were inferred dynamically
	assert.Equal(t, data.FieldTypeNullableString, frame.Fields[0].Type(), "First column should be string")
	assert.Equal(t, data.FieldTypeNullableFloat64, frame.Fields[1].Type(), "Second column should be float64")
}

// TestFrameHybridBasic tests the hybrid frame approach with mixed dynamic/static columns
func TestFrameHybridBasic(t *testing.T) {
	kind := &sql.ColumnType{}
	types := []*sql.ColumnType{kind, kind}
	
	// Column 0 is static (but empty converters means it will fallback to default),
	// column 1 is dynamic
	dynamicIndices := map[int]bool{
		1: true, // Only column 1 is dynamic
	}
	
	// Mock data: column 0 has int32, column 1 (dynamic) has string
	mockData := [][]interface{}{
		{int32(123), "dynamic_value_1"},
		{int32(456), "dynamic_value_2"},
	}
	
	mock := &MockRows{
		data:  mockData,
		index: -1,
	}
	rows := Rows{itr: mock}
	
	// No converters for simplicity - column 0 will use fallback, column 1 is dynamic
	converters := []Converter{}
	
	// Test frameHybrid (new per-column behavior)
	frame, err := frameHybrid(rows, 100, types, converters, dynamicIndices)
	require.NoError(t, err)
	require.NotNil(t, frame)
	
	// Verify frame structure
	assert.Equal(t, 2, len(frame.Fields), "Should have 2 fields")
	assert.Equal(t, 2, frame.Rows(), "Should have 2 rows")
	
	// int32 infers as float64 (runtime inference), string infers as string
	assert.Equal(t, data.FieldTypeNullableFloat64, frame.Fields[0].Type(), "First column inferred as float64 from int32")
	assert.Equal(t, data.FieldTypeNullableString, frame.Fields[1].Type(), "Second (dynamic) column inferred as string")
}
