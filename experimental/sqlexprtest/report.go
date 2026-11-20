package sqlexprtest

import (
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Severity says how a finding affects a SQL Expression.
type Severity int

const (
	// SeverityNone is what Report.Worst returns for a report with no findings.
	SeverityNone Severity = iota - 1
	// SeverityInfo is advice or expected normalization; it does not fail assertions by default.
	SeverityInfo
	// SeverityWarning means data may be lost or changed, or some SQL references may fail.
	SeverityWarning
	// SeverityError means input conversion, schema inspection, or SQL row reading fails.
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityNone:
		return "none"
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return fmt.Sprintf("severity(%d)", int(s))
	}
}

// MarshalText saves a severity by name, such as "warning", so JSON reports
// stay readable if the constants change.
func (s Severity) MarshalText() ([]byte, error) {
	switch s {
	case SeverityNone, SeverityInfo, SeverityWarning, SeverityError:
		return []byte(s.String()), nil
	default:
		return nil, fmt.Errorf("unknown severity %d", int(s))
	}
}

// UnmarshalText reads a severity name that MarshalText wrote.
func (s *Severity) UnmarshalText(b []byte) error {
	for _, v := range []Severity{SeverityNone, SeverityInfo, SeverityWarning, SeverityError} {
		if string(b) == v.String() {
			*s = v
			return nil
		}
	}
	return fmt.Errorf("unknown severity %q", b)
}

// Owner says where a finding is fixed.
type Owner string

const (
	// OwnerPlugin findings are fixed in the plugin.
	OwnerPlugin Owner = "plugin"
	// OwnerGrafana findings describe how Grafana handles the input today.
	// The plugin can often work around them.
	OwnerGrafana Owner = "grafana"
)

// Code identifies a check.
type Code string

const (
	// CodeQueryError: the query's DataResponse has an error, so SQL gets that error.
	CodeQueryError Code = "query-error"
	// CodeMissingRefID: the response has no entry for the query's RefID.
	CodeMissingRefID Code = "missing-refid"
	// CodeUnexpectedRefID: the response has an entry no query asked for.
	CodeUnexpectedRefID Code = "unexpected-refid"
	// CodeNilResponse: the handler returned a nil QueryDataResponse.
	CodeNilResponse Code = "nil-response"
	// CodePanic: the QueryData handler panicked.
	CodePanic Code = "panic"
	// CodeNoFrames: SQL receives a table with no columns.
	CodeNoFrames Code = "no-frames"
	// CodeNoFields: the query returned one frame with no fields.
	CodeNoFields Code = "no-fields"
	// CodeMissingType: an unlabeled frame passes through without Meta.Type.
	CodeMissingType Code = "missing-type"
	// CodeMultipleTableFrames: several frames cannot pass through as one table.
	CodeMultipleTableFrames Code = "multiple-table-frames"
	// CodeLabelsOnTable: field labels prevent a frame from passing through as a table.
	CodeLabelsOnTable Code = "labels-on-table"
	// CodeMixedTypes: later frames declare a different type from the first frame.
	CodeMixedTypes Code = "mixed-types"
	// CodeWideFrameCount: wide conversion requires exactly one frame.
	CodeWideFrameCount Code = "wide-frame-count"
	// CodeMissingTimeField: time series conversion requires a time field.
	CodeMissingTimeField Code = "missing-time-field"
	// CodeNullableTime: time series conversion has no non-nullable time field.
	CodeNullableTime Code = "nullable-time"
	// CodeNumericRowCount: the first field used to count numeric rows has more than one row.
	CodeNumericRowCount Code = "numeric-row-count"
	// CodeNumericValuesTruncated: numeric conversion keeps only the first value of this field.
	CodeNumericValuesTruncated Code = "numeric-values-truncated"
	// CodeEmptyNumericField: numeric conversion panics on an empty value field.
	CodeEmptyNumericField Code = "empty-numeric-field"
	// CodeNilFrame: an input frame pointer is nil.
	CodeNilFrame Code = "nil-frame"
	// CodeNilField: an input field pointer is nil.
	CodeNilField Code = "nil-field"
	// CodeInvalidField: a field has no values array, as in a zero-value data.Field.
	CodeInvalidField Code = "invalid-field"
	// CodeUnequalFieldLengths: unequal lengths cause truncation or an input/SQL failure.
	CodeUnequalFieldLengths Code = "unequal-field-lengths"
	// CodeDroppedField: conversion drops this non-numeric value field.
	CodeDroppedField Code = "dropped-field"
	// CodeDroppedLabel: conversion drops labels on the selected time field.
	CodeDroppedLabel Code = "dropped-label"
	// CodeIntegerAsFloat: integer values converted to float64 may lose precision.
	CodeIntegerAsFloat Code = "integer-as-float"
	// CodeLargeIntegerInFloat: a float value may already have lost integer precision.
	CodeLargeIntegerInFloat Code = "large-integer-in-float"
	// CodeNonfiniteAsNull: SQL reads NaN and infinities as NULL.
	CodeNonfiniteAsNull Code = "nonfinite-as-null"
	// CodeInvalidJSON: a JSON field contains malformed JSON values.
	CodeInvalidJSON Code = "invalid-json"
	// CodeJSONNumberOutOfRange: a valid JSON number exceeds the SQL reader's float64 range.
	CodeJSONNumberOutOfRange Code = "json-number-out-of-range"
	// CodeJSONIntegerAsFloat: SQL rounds an integer JSON number when decoding it as float64.
	CodeJSONIntegerAsFloat Code = "json-integer-as-float"
	// CodeEnumField: SQL reads enum indexes as JSON numbers rather than enum text.
	CodeEnumField Code = "enum-field"
	// CodeDuplicateName: two columns share a name, ignoring case.
	CodeDuplicateName Code = "duplicate-column-name"
	// CodeGeneratedNameClash: a label key matches a generated column.
	CodeGeneratedNameClash Code = "generated-name-clash"
	// CodeMissingLabelColumn: only empty multi-series values supply a label, so its column is lost.
	CodeMissingLabelColumn Code = "missing-label-column"
)

// Codes returns a copy of the finding-code catalog, in declaration order.
func Codes() []Code {
	return []Code{
		CodeQueryError,
		CodeMissingRefID,
		CodeUnexpectedRefID,
		CodeNilResponse,
		CodePanic,
		CodeNoFrames,
		CodeNoFields,
		CodeMissingType,
		CodeMultipleTableFrames,
		CodeLabelsOnTable,
		CodeMixedTypes,
		CodeWideFrameCount,
		CodeMissingTimeField,
		CodeNullableTime,
		CodeNumericRowCount,
		CodeNumericValuesTruncated,
		CodeEmptyNumericField,
		CodeNilFrame,
		CodeNilField,
		CodeInvalidField,
		CodeUnequalFieldLengths,
		CodeDroppedField,
		CodeDroppedLabel,
		CodeIntegerAsFloat,
		CodeLargeIntegerInFloat,
		CodeNonfiniteAsNull,
		CodeInvalidJSON,
		CodeJSONNumberOutOfRange,
		CodeJSONIntegerAsFloat,
		CodeEnumField,
		CodeDuplicateName,
		CodeGeneratedNameClash,
		CodeMissingLabelColumn,
	}
}

// Finding is one problem or piece of advice about a query's frames.
type Finding struct {
	Code     Code     `json:"code"`
	Severity Severity `json:"severity"`
	Owner    Owner    `json:"owner"`
	// Frame is the index of the frame the finding is about, or -1 when it is
	// about the whole query.
	Frame int `json:"frame"`
	// Field is the source field name, or a label/column name for generated tables.
	// Messages include source field indexes when malformed inputs need an unambiguous location.
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	// FixedIn is the first Grafana version without the behavior an
	// OwnerGrafana finding reports. It is empty when no release fixes it yet.
	FixedIn string `json:"fixedIn,omitempty"`
}

func (f Finding) String() string {
	loc := ""
	switch {
	case f.Frame >= 0 && f.Field != "":
		loc = fmt.Sprintf(" frame %d field %q:", f.Frame, f.Field)
	case f.Frame >= 0:
		loc = fmt.Sprintf(" frame %d:", f.Frame)
	case f.Field != "":
		loc = fmt.Sprintf(" field %q:", f.Field)
	}
	fixed := ""
	if f.FixedIn != "" {
		fixed = ", fixed in Grafana " + f.FixedIn
	}
	return fmt.Sprintf("%s [%s, %s%s]%s %s", f.Severity, f.Code, f.Owner, fixed, loc, f.Message)
}

// Mode is how Grafana turns a query's frames into a SQL table.
type Mode string

const (
	// ModeNone: input conversion fails or panics without producing a frame.
	ModeNone Mode = "none"
	// ModeTable: the frame passes through as a table, one column per field.
	// Schema inspection or SQL row reading can still fail; see error findings.
	ModeTable Mode = "table"
	// ModeFullLong: Grafana converts numeric or time series frames into one
	// long table with a row per value.
	ModeFullLong Mode = "full-long"
)

// Column is a column of the SQL table Grafana builds.
type Column struct {
	Name string `json:"name"`
	// Type is the field type Grafana builds the column from.
	Type data.FieldType `json:"type"`
	// SQLType is the column's SQL type, for example BIGINT or TEXT.
	SQLType string `json:"sqlType"`
}

// Report is the result of checking one query's frames.
type Report struct {
	RefID string `json:"refId"`
	// InputType is the Meta.Type of the first frame, which Grafana uses for
	// every frame of the query.
	InputType data.FrameType `json:"inputType"`
	Mode      Mode           `json:"mode"`
	// Rows is the converted frame row count. It is zero when Mode is ModeNone.
	Rows int `json:"rows"`
	// Columns describes the schema at the selected minimum version. A finding
	// explains known schema changes in newer versions. It is empty for ModeNone.
	// An uninitialized field has TypeUnknown and an empty SQLType because
	// Grafana cannot inspect its schema; see the corresponding error finding.
	Columns  []Column  `json:"columns"`
	Findings []Finding `json:"findings"`

	// minGrafana is the oldest Grafana version the plugin supports, or nil
	// when it is not known.
	minGrafana *version
}

// Has reports whether the report has a finding with the code.
func (r *Report) Has(code Code) bool {
	for _, f := range r.Findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

// Worst returns the highest severity in the report, or SeverityNone when it
// has no findings.
func (r *Report) Worst() Severity {
	worst := SeverityNone
	for _, f := range r.Findings {
		worst = max(worst, f.Severity)
	}
	return worst
}

func (r *Report) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "query %q: mode %s", r.RefID, r.Mode)
	if r.InputType != "" {
		fmt.Fprintf(&b, ", input type %s", r.InputType)
	}
	if r.Mode != ModeNone {
		fmt.Fprintf(&b, ", rows %d", r.Rows)
	}
	b.WriteString("\n")
	if len(r.Columns) > 0 {
		cols := make([]string, len(r.Columns))
		for i, c := range r.Columns {
			cols[i] = c.Name + " " + c.SQLType
		}
		fmt.Fprintf(&b, "  columns: %s\n", strings.Join(cols, ", "))
	}
	for _, f := range r.Findings {
		fmt.Fprintf(&b, "  %s\n", f)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func (r *Report) add(f Finding) {
	r.Findings = append(r.Findings, f)
}
