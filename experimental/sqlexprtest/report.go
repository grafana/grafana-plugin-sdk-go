package sqlexprtest

import (
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Severity says how a finding affects a SQL Expression.
type Severity int

const (
	// SeverityInfo is advice. Nothing is lost.
	SeverityInfo Severity = iota
	// SeverityWarning means SQL runs, but data is lost or changed without an error.
	SeverityWarning
	// SeverityError means the SQL Expression fails on this input.
	SeverityError
)

func (s Severity) String() string {
	switch s {
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
	// CodeUnexpectedRefID: the response has an entry for a RefID no query asked for.
	CodeUnexpectedRefID Code = "unexpected-refid"
	// CodePanic: the QueryData handler panicked.
	CodePanic Code = "panic"
	// CodeNoFrames: the query returned no frames, so SQL sees a table with no columns.
	CodeNoFrames Code = "no-frames"
	// CodeNoFields: the query returned one frame with no fields.
	CodeNoFields Code = "no-fields"
	// CodeMissingType: a frame has no Meta.Type.
	CodeMissingType Code = "missing-type"
	// CodeUnsupportedType: Grafana can't convert this Meta.Type, and the
	// frames can't pass through as one table.
	CodeUnsupportedType Code = "unsupported-type"
	// CodeMixedTypes: frames of one query have different Meta.Type values.
	CodeMixedTypes Code = "mixed-types"
	// CodeInvalidShape: the frames don't match the shape their Meta.Type
	// declares, so Grafana's conversion fails.
	CodeInvalidShape Code = "invalid-shape"
	// CodeNullableTime: a time series frame's only time field is nullable.
	CodeNullableTime Code = "nullable-time"
	// CodeDroppedField: Grafana drops this field when it converts the frames for SQL.
	CodeDroppedField Code = "dropped-field"
	// CodeIntegerAsFloat: Grafana stores this integer value field as float64,
	// and some values are too large to keep exactly.
	CodeIntegerAsFloat Code = "integer-as-float"
	// CodeLargeIntegerInFloat: a float field holds whole numbers at or above
	// 2^53, which suggests integers already lost precision in the plugin.
	CodeLargeIntegerInFloat Code = "large-integer-in-float"
	// CodeEnumField: SQL reads an enum field as a JSON number that holds the
	// enum index, not the text.
	CodeEnumField Code = "enum-field"
	// CodeDuplicateName: two columns share a name, ignoring case.
	CodeDuplicateName Code = "duplicate-column-name"
	// CodeGeneratedNameClash: a label key matches a column Grafana generates.
	CodeGeneratedNameClash Code = "generated-name-clash"
)

// Finding is one problem or piece of advice about a query's frames.
type Finding struct {
	Code     Code
	Severity Severity
	Owner    Owner
	// Frame is the index of the frame the finding is about, or -1 when it is
	// about the whole query.
	Frame int
	// Field is the field name the finding is about, if any.
	Field   string
	Message string
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
	return fmt.Sprintf("%s [%s, %s]%s %s", f.Severity, f.Code, f.Owner, loc, f.Message)
}

// Mode is how Grafana turns a query's frames into a SQL table.
type Mode string

const (
	// ModeNone: Grafana builds no table; the SQL Expression fails.
	ModeNone Mode = "none"
	// ModeTable: the frame passes through as a table, one column per field.
	ModeTable Mode = "table"
	// ModeFullLong: Grafana converts numeric or time series frames into one
	// long table with a row per value.
	ModeFullLong Mode = "full-long"
)

// Column is a column of the SQL table Grafana builds.
type Column struct {
	Name string
	// Type is the field type Grafana builds the column from.
	Type data.FieldType
	// SQLType is the column's SQL type, for example BIGINT or TEXT.
	SQLType string
}

// Report is the result of checking one query's frames.
type Report struct {
	RefID string
	// InputType is the Meta.Type of the first frame, which Grafana uses for
	// every frame of the query.
	InputType data.FrameType
	Mode      Mode
	// Columns lists the columns SQL will see. It is empty when Mode is ModeNone.
	Columns  []Column
	Findings []Finding
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

// Worst returns the highest severity in the report, or -1 when it has no findings.
func (r *Report) Worst() Severity {
	worst := Severity(-1)
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
