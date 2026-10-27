package sqlexprtest

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Columns Grafana generates when it converts numeric and time series frames.
const (
	timeColumn    = "time"
	valueColumn   = "__value__"
	metricColumn  = "__metric_name__"
	displayColumn = "__display_name__"
)

// maxExactInteger is 2^53, the largest power of two below which float64
// holds every integer exactly.
const maxExactInteger = 1 << 53

// Check checks the frames that one query returned, as SQL Expressions read
// them. It uses only the [MinGrafana] option. It never returns nil.
func Check(frames data.Frames, opts ...Option) *Report {
	r := &Report{Mode: ModeTable, minGrafana: newOptions(opts).minGrafana}
	if len(frames) == 0 {
		r.add(Finding{
			Code: CodeNoFrames, Severity: SeverityWarning, Owner: OwnerPlugin, Frame: -1,
			Message: "the query returned no frames, so SQL sees a table with no columns, and a query on its columns fails. " +
				"Return a frame with typed fields and zero rows, so SQL keeps the schema.",
		})
		return r
	}

	if frames[0] == nil {
		r.Mode = ModeNone
		r.add(Finding{Code: CodeNilFrame, Severity: SeverityError, Owner: OwnerPlugin, Frame: 0,
			Message: "the first frame is nil, so Grafana panics when it reads the SQL input. Return a frame, not nil."})
		return r
	}
	r.RefID = frames[0].RefID
	r.InputType = frameType(frames[0])
	if len(frames) == 1 && len(frames[0].Fields) == 0 {
		r.add(Finding{
			Code: CodeNoFields, Severity: SeverityWarning, Owner: OwnerPlugin, Frame: 0,
			Message: "the frame has no fields, so SQL sees a table with no columns, and a query on its columns fails. " +
				"Return the fields with zero rows, so SQL keeps the schema.",
		})
		return r
	}

	if convertible(r.InputType) {
		checkFullLong(r, frames)
	} else {
		checkTable(r, frames)
	}
	if r.Mode == ModeNone {
		r.Rows = 0
		r.Columns = nil
	}
	if r.Worst() < SeverityError {
		checkValues(r, frames)
	}
	return r
}

func frameType(f *data.Frame) data.FrameType {
	if f.Meta == nil {
		return data.FrameTypeUnknown
	}
	return f.Meta.Type
}

// convertible reports whether Grafana converts frames of this type into a
// full-long table, instead of passing them through as a table.
func convertible(t data.FrameType) bool {
	switch t {
	case data.FrameTypeNumericMulti, data.FrameTypeNumericWide,
		data.FrameTypeTimeSeriesMulti, data.FrameTypeTimeSeriesWide:
		return true
	default:
		return false
	}
}

func checkTable(r *Report, frames data.Frames) {
	if len(frames) > 1 {
		r.Mode = ModeNone
		r.add(Finding{Code: CodeMultipleTableFrames, Severity: SeverityError, Owner: OwnerPlugin, Frame: -1,
			Message: fmt.Sprintf("the query returned %d frames, but Grafana passes only one frame through as a table, so the SQL Expression fails. "+
				"Merge them into one table. If they are series, set Meta.Type to timeseries-multi or numeric-multi.", len(frames))})
		return
	}
	frame := frames[0]
	for j, f := range frame.Fields {
		if f == nil {
			r.Mode = ModeNone
			addNilField(r, 0, j)
			return
		}
		if len(f.Labels) > 0 {
			r.Mode = ModeNone
			r.add(Finding{Code: CodeLabelsOnTable, Severity: SeverityError, Owner: OwnerPlugin, Frame: 0, Field: f.Name,
				Message: fmt.Sprintf("field %d has labels, and a table can't hold labels, so the SQL Expression fails. Move the labels into string fields. "+
					"If the frame matches a supported series shape, set Meta.Type to timeseries-wide, timeseries-multi, numeric-wide, or numeric-multi. Grafana converts those labels into columns.", j)})
			return
		}
	}
	r.Rows = frame.Rows()
	enumFixedIn, enumFixed := r.fixedIn(CodeEnumField)
	for j, f := range frame.Fields {
		if !initializedField(f) {
			r.Columns = append(r.Columns, Column{Name: f.Name, Type: data.FieldTypeUnknown})
			r.add(Finding{Code: CodeInvalidField, Severity: SeverityError, Owner: OwnerPlugin, Frame: 0, Field: f.Name,
				Message: fmt.Sprintf("field %d has no values array, so Grafana panics when it reads the SQL schema. Create the field with data.NewField.", j)})
			continue
		}
		column := Column{Name: f.Name, Type: f.Type(), SQLType: sqlType(f.Type())}
		isEnum := f.Type() == data.FieldTypeEnum || f.Type() == data.FieldTypeNullableEnum
		if isEnum && enumFixed {
			column.SQLType = "TEXT"
		}
		r.Columns = append(r.Columns, column)
		if f.Len() != r.Rows {
			severity, consequence := SeverityWarning, "SQL reads only as many rows as field 0 has, so this field's last values are lost."
			if f.Len() < r.Rows {
				severity, consequence = SeverityError, "SQL panics when it reads past the end of this field."
			}
			r.add(Finding{Code: CodeUnequalFieldLengths, Severity: severity, Owner: OwnerPlugin, Frame: 0, Field: f.Name,
				Message: fmt.Sprintf("field %d has %d values, and field 0 has %d. %s Return fields of equal length.", j, f.Len(), r.Rows, consequence)})
		}
		if isEnum && !enumFixed {
			msg := fmt.Sprintf("SQL reads field %d's enum values as JSON numbers, the enum index, not the text. Use a string field.", j)
			if enumFixedIn != "" {
				msg += fmt.Sprintf(" Grafana %s and later read the enum text as TEXT, but the plugin supports older versions.", enumFixedIn)
			}
			r.add(Finding{Code: CodeEnumField, Severity: SeverityWarning, Owner: OwnerGrafana, Frame: 0, Field: f.Name, FixedIn: enumFixedIn,
				Message: msg})
		}
	}
	if r.InputType == data.FrameTypeUnknown {
		r.add(Finding{Code: CodeMissingType, Severity: SeverityInfo, Owner: OwnerPlugin, Frame: 0,
			Message: "the frame has no Meta.Type, so it passes through as a table. " +
				"Set Meta.Type, for example to table, timeseries-multi or numeric-wide, so Grafana knows the shape."})
	}
	checkDuplicateNames(r, r.Columns, 0)
}

// RowLen can inspect private field storage without calling Type, which panics
// on a zero-value Field. A one-field frame cannot have unequal field lengths.
func initializedField(f *data.Field) bool {
	_, err := data.NewFrame("", f).RowLen()
	return err == nil
}

func addNilField(r *Report, frame, field int) {
	r.add(Finding{Code: CodeNilField, Severity: SeverityError, Owner: OwnerPlugin, Frame: frame,
		Message: fmt.Sprintf("field %d is nil, so Grafana panics when it converts the SQL input. Return a field, not nil.", field)})
}

func checkFullLongFrame(r *Report, i int, frame *data.Frame) bool {
	if frame != nil {
		return true
	}
	r.add(Finding{Code: CodeNilFrame, Severity: SeverityError, Owner: OwnerPlugin, Frame: i,
		Message: "the frame is nil, so Grafana panics when it converts the SQL input. Return a frame, not nil."})
	return false
}

func checkFullLongField(r *Report, i, j int, f *data.Field) bool {
	if f == nil {
		addNilField(r, i, j)
		return false
	}
	if !initializedField(f) {
		r.add(Finding{Code: CodeInvalidField, Severity: SeverityError, Owner: OwnerPlugin, Frame: i, Field: f.Name,
			Message: fmt.Sprintf("field %d has no values array, so Grafana panics when it converts the SQL input. Create the field with data.NewField.", j)})
		return false
	}
	return true
}

func checkFullLongPointers(r *Report, frames data.Frames) bool {
	for i, frame := range frames {
		if !checkFullLongFrame(r, i, frame) {
			return false
		}
		for j, f := range frame.Fields {
			if !checkFullLongField(r, i, j, f) {
				return false
			}
		}
	}
	return true
}

// checkDuplicateNames reports columns whose names match, ignoring case.
// frame is the frame the columns come from, or -1 for generated tables.
func checkDuplicateNames(r *Report, cols []Column, frame int) {
	seen := map[string]string{}
	for _, c := range cols {
		key := strings.ToLower(c.Name)
		if prev, ok := seen[key]; ok {
			r.add(Finding{
				Code: CodeDuplicateName, Severity: SeverityWarning, Owner: OwnerGrafana, Frame: frame, Field: c.Name,
				Message: fmt.Sprintf("columns %q and %q have the same name, ignoring case, "+
					"so a SQL reference to either fails with an ambiguous column error. Give each field a unique name.", prev, c.Name),
			})
			continue
		}
		seen[key] = c.Name
	}
}

type fullLong struct {
	labelKeys map[string]struct{}
	// emptyLabelKeys are label keys of timeseries-multi series with no rows.
	// Grafana takes label columns from rows, so these keys get no column.
	emptyLabelKeys map[string]struct{}
	hasDisplay     bool
	rows           int
}

func checkFullLong(r *Report, frames data.Frames) {
	r.Mode = ModeFullLong
	if !checkFullLongShape(r, frames) {
		r.Mode = ModeNone
		return
	}

	acc := fullLong{labelKeys: map[string]struct{}{}, emptyLabelKeys: map[string]struct{}{}}
	timeSeries := r.InputType == data.FrameTypeTimeSeriesMulti || r.InputType == data.FrameTypeTimeSeriesWide
	for i, frame := range frames {
		if !checkFullLongFrame(r, i, frame) {
			r.Mode = ModeNone
			return
		}
		// Grafana ignores the Meta of later frames, so a frame without a type is fine.
		if t := frameType(frame); i > 0 && t != data.FrameTypeUnknown && t != r.InputType {
			r.add(Finding{
				Code: CodeMixedTypes, Severity: SeverityWarning, Owner: OwnerPlugin, Frame: i,
				Message: fmt.Sprintf("the frame has Meta.Type %q, but Grafana reads every frame of a query as the first frame's type, %q, "+
					"so this frame's data can be lost or misread without an error. Return one frame type per query.", t, r.InputType),
			})
		}

		var timeField *data.Field
		if timeSeries {
			// Match the converter's search: malformed fields after the selected
			// time field are visited only by its subsequent numeric-field loop.
			for j, f := range frame.Fields {
				if !checkFullLongField(r, i, j, f) {
					r.Mode = ModeNone
					return
				}
				if f.Type() == data.FieldTypeTime {
					timeField = f
					break
				}
			}
			if timeField == nil {
				addMissingTime(r, i, frame)
				r.Mode = ModeNone
				return
			}
		}
		// Wide time series collects label metadata from all fields before rows.
		if r.InputType == data.FrameTypeTimeSeriesWide && !checkFullLongPointers(r, frames) {
			r.Mode = ModeNone
			return
		}
		if !checkFullLongFields(r, i, frame, timeField, &acc) {
			r.Mode = ModeNone
			return
		}
	}

	r.Rows = acc.rows
	r.Columns = fullLongColumns(timeSeries, acc)
	// Generated columns come first, then one column per label key.
	split := len(r.Columns) - len(acc.labelKeys)
	generated, labels := r.Columns[:split], r.Columns[split:]
	renames := checkGeneratedNameClash(r, generated, labels)
	// Label keys that clash with generated columns are reported above, so
	// only compare the label columns with each other.
	for i, name := range renames {
		labels[i].Name = name
	}
	checkDuplicateNames(r, labels, -1)
	checkEmptySeriesLabels(r, acc)
}

// checkEmptySeriesLabels reports label keys that only empty series have, so
// SQL gets no column for them.
func checkEmptySeriesLabels(r *Report, acc fullLong) {
	keys := make([]string, 0, len(acc.emptyLabelKeys))
	for k := range acc.emptyLabelKeys {
		if _, ok := acc.labelKeys[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		r.add(Finding{
			Code: CodeMissingLabelColumn, Severity: SeverityWarning, Owner: OwnerGrafana, Frame: -1, Field: k,
			Message: fmt.Sprintf("only series with no rows have the label %q, and Grafana takes label columns from rows, "+
				"so SQL has no %q column, and a query on it fails. Use timeseries-wide, or a long frame, "+
				"to keep the columns when there are no rows.", k, k),
		})
	}
}

// checkFullLongShape checks the frame and row counts the declared type allows.
func checkFullLongShape(r *Report, frames data.Frames) bool {
	wide := r.InputType == data.FrameTypeNumericWide || r.InputType == data.FrameTypeTimeSeriesWide
	if wide && len(frames) != 1 {
		r.add(Finding{
			Code: CodeWideFrameCount, Severity: SeverityError, Owner: OwnerPlugin, Frame: -1,
			Message: fmt.Sprintf("Meta.Type %s allows one frame, but the query returned %d, so Grafana's conversion fails. "+
				"Return one frame, or use the matching multi type.", r.InputType, len(frames)),
		})
		return false
	}
	if r.InputType == data.FrameTypeNumericWide && frames[0].Rows() > 1 {
		r.add(Finding{
			Code: CodeNumericRowCount, Severity: SeverityError, Owner: OwnerPlugin, Frame: 0,
			Message: fmt.Sprintf("Meta.Type numeric-wide allows at most one row, but the frame has %d, so Grafana's conversion fails. "+
				"Use a time series type, or a table.", frames[0].Rows()),
		})
		return false
	}
	if r.InputType == data.FrameTypeNumericMulti {
		// Numeric-multi visits every input field while merging to numeric-wide.
		return checkFullLongPointers(r, frames) && checkNumericMultiFirstField(r, frames)
	}
	return true
}

// checkNumericMultiFirstField checks the first numeric field of all frames.
// Grafana merges numeric-multi frames into one frame of their numeric fields,
// then allows at most one row, counted on that first field.
func checkNumericMultiFirstField(r *Report, frames data.Frames) bool {
	for i, frame := range frames {
		for j, f := range frame.Fields {
			if !f.Type().Numeric() {
				continue
			}
			if f.Len() > 1 {
				r.add(Finding{
					Code: CodeNumericRowCount, Severity: SeverityError, Owner: OwnerPlugin, Frame: i, Field: f.Name,
					Message: fmt.Sprintf("Meta.Type numeric-multi allows one value per field, but field %d, the first numeric field, "+
						"has %d, so Grafana's conversion fails. Use a time series type, or a table.", j, f.Len()),
				})
				return false
			}
			return true
		}
	}
	return true
}

func addMissingTime(r *Report, i int, frame *data.Frame) {
	if f := firstFieldOfType(frame, data.FieldTypeNullableTime); f != nil {
		r.add(Finding{
			Code: CodeNullableTime, Severity: SeverityError, Owner: OwnerGrafana, Frame: i, Field: f.Name,
			Message: "the frame's time fields are all nullable ([]*time.Time). Grafana's time series conversion " +
				"needs a non-nullable time field, so the SQL Expression fails. Use a []time.Time field.",
		})
		return
	}
	r.add(Finding{
		Code: CodeMissingTimeField, Severity: SeverityError, Owner: OwnerPlugin, Frame: i,
		Message: fmt.Sprintf("Meta.Type %s needs a time field, but the frame has none, so Grafana's conversion fails.", r.InputType),
	})
}

// checkFullLongFields checks one frame's fields for the full-long
// conversion. It returns false when the conversion fails.
func checkFullLongFields(r *Report, i int, frame *data.Frame, timeField *data.Field, acc *fullLong) bool {
	for j, f := range frame.Fields {
		if !checkFullLongField(r, i, j, f) {
			return false
		}
		switch {
		case f == timeField:
			if len(f.Labels) > 0 {
				keys := make([]string, 0, len(f.Labels))
				for k := range f.Labels {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				r.add(Finding{Code: CodeDroppedLabel, Severity: SeverityWarning, Owner: OwnerGrafana, Frame: i, Field: f.Name,
					Message: fmt.Sprintf("Grafana drops the labels %q on time field %d. "+
						"Put series labels on the numeric value fields, or return them as string columns in a table or long frame.", keys, j)})
			}
		case f.Type().Numeric():
			rows := 1
			if timeField == nil {
				if !checkNumericRows(r, i, j, f) {
					return false
				}
			} else {
				rows = f.Len()
				if r.InputType == data.FrameTypeTimeSeriesWide {
					rows = timeField.Len()
				}
				if f.Len() != timeField.Len() {
					severity := SeverityWarning
					consequence := "Grafana drops this field's last values."
					if r.InputType == data.FrameTypeTimeSeriesMulti {
						consequence = "Grafana ignores the timestamps after this field's last value."
					}
					if rows > f.Len() || rows > timeField.Len() {
						severity, consequence = SeverityError, "Grafana panics when it reads past the end of the shorter field."
					}
					r.add(Finding{Code: CodeUnequalFieldLengths, Severity: severity, Owner: OwnerPlugin, Frame: i, Field: f.Name,
						Message: fmt.Sprintf("numeric field %d has %d values, and the time field has %d. %s Return fields of equal length.", j, f.Len(), timeField.Len(), consequence)})
					if severity == SeverityError {
						return false
					}
				}
			}
			acc.rows += rows
			keys := acc.labelKeys
			if r.InputType == data.FrameTypeTimeSeriesMulti && rows == 0 {
				keys = acc.emptyLabelKeys
			}
			for k := range f.Labels {
				keys[k] = struct{}{}
			}
			if f.Config != nil && f.Config.DisplayNameFromDS != "" {
				acc.hasDisplay = true
			}
		default:
			r.add(Finding{Code: CodeDroppedField, Severity: SeverityWarning, Owner: OwnerGrafana, Frame: i, Field: f.Name,
				Message: fmt.Sprintf("Grafana keeps only numeric value fields when it converts %s for SQL, so field %d (%s) is dropped without an error. Return a table or long frame to keep it.", r.InputType, j, f.Type().ItemTypeString())})
		}
	}
	return true
}

// checkNumericRows checks that a numeric frame's value field has exactly one
// row, because Grafana reads only the first value. It returns false when the
// conversion fails.
func checkNumericRows(r *Report, i, fieldIndex int, f *data.Field) bool {
	switch {
	case f.Len() == 0:
		r.add(Finding{
			Code: CodeEmptyNumericField, Severity: SeverityError, Owner: OwnerGrafana, Frame: i, Field: f.Name,
			Message: fmt.Sprintf("field %d has no values, and Grafana fails the whole request when it converts an empty %s field for SQL. "+
				"Return a table frame with zero rows instead.", fieldIndex, r.InputType),
		})
		return false
	case f.Len() > 1:
		r.add(Finding{
			Code: CodeNumericValuesTruncated, Severity: SeverityWarning, Owner: OwnerPlugin, Frame: i, Field: f.Name,
			Message: fmt.Sprintf("field %d has %d values, but %s holds one value per field, and Grafana keeps only the first, without an error. "+
				"Use a time series type for several values.", fieldIndex, f.Len(), r.InputType),
		})
	}
	return true
}

func fullLongColumns(timeSeries bool, acc fullLong) []Column {
	value := Column{Name: valueColumn, Type: data.FieldTypeNullableFloat64, SQLType: sqlType(data.FieldTypeNullableFloat64)}
	metric := Column{Name: metricColumn, Type: data.FieldTypeString, SQLType: sqlType(data.FieldTypeString)}
	var cols []Column
	if timeSeries {
		cols = []Column{{Name: timeColumn, Type: data.FieldTypeTime, SQLType: sqlType(data.FieldTypeTime)}, value, metric}
	} else {
		cols = []Column{metric, value}
	}
	if acc.hasDisplay {
		cols = append(cols, Column{Name: displayColumn, Type: data.FieldTypeNullableString, SQLType: sqlType(data.FieldTypeNullableString)})
	}

	keys := make([]string, 0, len(acc.labelKeys))
	for k := range acc.labelKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		cols = append(cols, Column{Name: k, Type: data.FieldTypeNullableString, SQLType: sqlType(data.FieldTypeNullableString)})
	}
	return cols
}

// checkGeneratedNameClash reports label keys that match, ignoring case, a
// column Grafana generates. Grafana versions with the fix in grafanaFixes add
// a "label_" prefix to such a label column until its name is unique. When
// every supported version has the fix, it returns the new names, by index into
// labels.
func checkGeneratedNameClash(r *Report, generated, labels []Column) map[int]string {
	fixedIn, fixed := r.fixedIn(CodeGeneratedNameClash)
	isGenerated, taken := map[string]bool{}, map[string]bool{}
	for _, c := range generated {
		isGenerated[strings.ToLower(c.Name)] = true
		taken[strings.ToLower(c.Name)] = true
	}
	for _, c := range labels {
		taken[strings.ToLower(c.Name)] = true
	}

	renames := map[int]string{}
	for i, c := range labels {
		if !isGenerated[strings.ToLower(c.Name)] {
			continue
		}
		name := "label_" + c.Name
		for taken[strings.ToLower(name)] {
			name = "label_" + name
		}
		taken[strings.ToLower(name)] = true

		if fixed {
			renames[i] = name
			r.add(Finding{
				Code: CodeGeneratedNameClash, Severity: SeverityInfo, Owner: OwnerGrafana, Frame: -1, Field: c.Name, FixedIn: fixedIn,
				Message: fmt.Sprintf("the label key %q matches a column Grafana generates, so Grafana names its column %q. "+
					"Use that name in SQL, or rename the label.", c.Name, name),
			})
			continue
		}
		msg := fmt.Sprintf("the label key %q matches a column Grafana generates, "+
			"so a SQL reference to it fails with an ambiguous column error. Rename the label.", c.Name)
		if fixedIn != "" {
			msg += fmt.Sprintf(" Grafana %s and later name the column %q, "+
				"but the plugin supports older versions.", fixedIn, name)
		}
		r.add(Finding{
			Code: CodeGeneratedNameClash, Severity: SeverityWarning, Owner: OwnerGrafana, Frame: -1, Field: c.Name, FixedIn: fixedIn,
			Message: msg,
		})
	}
	return renames
}

func checkIntegerAsFloat(r *Report, i, fieldIndex int, f *data.Field, rows int) {
	switch f.Type() {
	case data.FieldTypeInt64, data.FieldTypeNullableInt64, data.FieldTypeUint64, data.FieldTypeNullableUint64:
	default:
		return
	}
	n, example := 0, ""
	for j := range rows {
		v, ok := f.ConcreteAt(j)
		if !ok {
			continue
		}
		large := false
		switch x := v.(type) {
		case int64:
			large = x > maxExactInteger || x < -maxExactInteger
		case uint64:
			large = x > maxExactInteger
		}
		if large {
			if n == 0 {
				example = fmt.Sprint(v)
			}
			n++
		}
	}
	if n > 0 {
		r.add(Finding{
			Code: CodeIntegerAsFloat, Severity: SeverityWarning, Owner: OwnerGrafana, Frame: i, Field: f.Name,
			Message: fmt.Sprintf("Grafana stores numeric and time series values as float64. %d values of field %d are beyond 2^53 "+
				"and may lose precision, for example %s. Return a table or long frame to keep exact integers.", n, fieldIndex, example),
		})
	}
}

// checkValues runs only when conversion and SQL reading succeed. It checks
// only the values SQL reads, not values Grafana drops.
func checkValues(r *Report, frames data.Frames) {
	for i, frame := range frames {
		for j, f := range frame.Fields {
			rows := r.Rows
			if r.Mode == ModeFullLong {
				if !f.Type().Numeric() {
					continue
				}
				switch r.InputType {
				case data.FrameTypeNumericMulti, data.FrameTypeNumericWide:
					rows = 1
				case data.FrameTypeTimeSeriesMulti:
					rows = f.Len()
				case data.FrameTypeTimeSeriesWide:
					rows = firstFieldOfType(frame, data.FieldTypeTime).Len()
				}
				checkIntegerAsFloat(r, i, j, f, rows)
			}
			checkFieldValues(r, i, j, f, rows)
		}
	}
}

func checkFieldValues(r *Report, frameIndex, fieldIndex int, f *data.Field, rows int) {
	t := f.Type().NonNullableType()
	if t != data.FieldTypeFloat32 && t != data.FieldTypeFloat64 && t != data.FieldTypeJSON {
		return
	}
	large, nonfinite, invalidJSON := 0, 0, 0
	jsonOutOfRange, jsonRounded := 0, 0
	bits, threshold := 53, float64(maxExactInteger)
	if t == data.FieldTypeFloat32 {
		bits, threshold = 24, 1<<24
	}
	for row := range rows {
		v, ok := f.ConcreteAt(row)
		if !ok {
			continue
		}
		var x float64
		switch v := v.(type) {
		case float64:
			x = v
		case float32:
			x = float64(v)
		case json.RawMessage:
			if v != nil && !json.Valid(v) {
				invalidJSON++
			} else if v != nil {
				outOfRange, rounded := checkJSONNumbers(v)
				jsonOutOfRange += outOfRange
				jsonRounded += rounded
			}
			continue
		}
		if math.IsInf(x, 0) || math.IsNaN(x) {
			nonfinite++
			continue
		}
		if math.Abs(x) >= threshold {
			large++
		}
	}
	if large > 0 {
		workaround := "Use an int64 field for integer columns."
		if r.Mode == ModeFullLong {
			workaround = "Return int64 fields in a table or long frame. Wide and multi conversion stores every value as float64."
		}
		r.add(Finding{Code: CodeLargeIntegerInFloat, Severity: SeverityInfo, Owner: OwnerPlugin, Frame: frameIndex, Field: f.Name,
			Message: fmt.Sprintf("%d values of field %d are at or above 2^%d, where %s can't hold every integer. "+
				"If the source column is an integer, the values may have lost precision before Grafana got them. %s", large, fieldIndex, bits, t.ItemTypeString(), workaround)})
	}
	if nonfinite > 0 {
		r.add(Finding{Code: CodeNonfiniteAsNull, Severity: SeverityInfo, Owner: OwnerGrafana, Frame: frameIndex, Field: f.Name,
			Message: fmt.Sprintf("SQL reads %d NaN or infinite values in field %d as NULL. "+
				"That changes COUNT(column), filters, and anything else that treats NULL differently.", nonfinite, fieldIndex)})
	}
	if jsonOutOfRange > 0 {
		r.add(Finding{Code: CodeJSONNumberOutOfRange, Severity: SeverityError, Owner: OwnerGrafana, Frame: frameIndex, Field: f.Name,
			Message: fmt.Sprintf("%d JSON values in field %d contain a number outside float64's finite range. Grafana decodes JSON numbers as float64, so SQL row reading fails. "+
				"Encode those numbers as JSON strings to preserve them.", jsonOutOfRange, fieldIndex)})
	}
	if jsonRounded > 0 {
		r.add(Finding{Code: CodeJSONIntegerAsFloat, Severity: SeverityWarning, Owner: OwnerGrafana, Frame: frameIndex, Field: f.Name,
			Message: fmt.Sprintf("%d integer JSON numbers in field %d change when Grafana decodes them as float64. "+
				"Encode exact large integers as JSON strings, or expose them as int64 or uint64 table columns when they fit.", jsonRounded, fieldIndex)})
	}
	if invalidJSON > 0 {
		r.add(Finding{Code: CodeInvalidJSON, Severity: SeverityWarning, Owner: OwnerPlugin, Frame: frameIndex, Field: f.Name,
			Message: fmt.Sprintf("field %d has %d values that aren't valid JSON. Grafana's SQL reader turns them into NULL without an error. "+
				"Return valid JSON, and use the literal null for a JSON null.", fieldIndex, invalidJSON)})
	}
}

func firstFieldOfType(frame *data.Frame, t data.FieldType) *data.Field {
	for _, f := range frame.Fields {
		if f.Type() == t {
			return f
		}
	}
	return nil
}

// sqlType returns the SQL column type Grafana uses for a field type.
func sqlType(t data.FieldType) string {
	switch t.NonNullableType() {
	case data.FieldTypeInt8:
		return "TINYINT"
	case data.FieldTypeUint8:
		return "TINYINT UNSIGNED"
	case data.FieldTypeInt16:
		return "SMALLINT"
	case data.FieldTypeUint16:
		return "SMALLINT UNSIGNED"
	case data.FieldTypeInt32:
		return "INT"
	case data.FieldTypeUint32:
		return "INT UNSIGNED"
	case data.FieldTypeInt64:
		return "BIGINT"
	case data.FieldTypeUint64:
		return "BIGINT UNSIGNED"
	case data.FieldTypeFloat32:
		return "FLOAT"
	case data.FieldTypeFloat64:
		return "DOUBLE"
	case data.FieldTypeString:
		return "TEXT"
	case data.FieldTypeBool:
		return "TINYINT(1)"
	case data.FieldTypeTime:
		return "TIMESTAMP"
	default:
		// JSON, enum, and any other type become JSON. Grafana 13.3.0 and
		// later read enums as TEXT; checkTable handles that.
		return "JSON"
	}
}
