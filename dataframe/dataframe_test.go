package dataframe_test

import (
	"database/sql"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
	"github.com/stretchr/testify/require"
)

func TestDataFrame(t *testing.T) {
	df := dataframe.New("http_requests_total",
		dataframe.NewField("timestamp", nil, []time.Time{time.Now(), time.Now(), time.Now()}).SetConfig(&dataframe.FieldConfig{
			Title: "A time Column.",
		}),
		dataframe.NewField("value", dataframe.Labels{"service": "auth"}, []float64{1.0, 2.0, 3.0}),
		dataframe.NewField("category", dataframe.Labels{"service": "auth"}, []string{"foo", "bar", "test"}),
		dataframe.NewField("valid", dataframe.Labels{"service": "auth"}, []bool{true, false, true}),
	)

	if df.Rows() != 3 {
		t.Fatal("unexpected length")
	}
}

func TestDataFrameWarnings(t *testing.T) {
	df := dataframe.New("warning_test")
	df.AppendWarning(&dataframe.Warning{Details: "details1", Message: "message1"})
	df.AppendWarning(&dataframe.Warning{Details: "details2", Message: "message2"})

	if len(df.Warnings) != 2 {
		t.Fatal("expected two warnings to be appended")
	}
}

func TestField(t *testing.T) {
	f := dataframe.NewField("value", nil, []float64{1.0, 2.0, 3.0})

	if f.Len() != 3 {
		t.Fatal("unexpected length")
	}
}

func TestField_Float64(t *testing.T) {
	f := dataframe.NewField("value", nil, make([]*float64, 3))

	want := 2.0
	f.Vector.Set(1, &want)

	if f.Len() != 3 {
		t.Fatal("unexpected length")
	}

	got := f.Vector.At(1).(*float64)

	if *got != want {
		t.Errorf("%+v", *got)
	}
}

func TestField_String(t *testing.T) {
	f := dataframe.NewField("value", nil, make([]*string, 3))

	want := "foo"
	f.Vector.Set(1, &want)

	if f.Len() != 3 {
		t.Fatal("unexpected length")
	}

	got := f.Vector.At(1).(*string)

	if *got != want {
		t.Errorf("%+v", *got)
	}
}

func TestTimeField(t *testing.T) {
	tests := []struct {
		Values []*time.Time
	}{
		{
			Values: []*time.Time{timePtr(time.Unix(111, 0))},
		},
		{
			Values: []*time.Time{nil, timePtr(time.Unix(111, 0))},
		},
		{
			Values: []*time.Time{nil, timePtr(time.Unix(111, 0)), nil},
		},
		{
			Values: make([]*time.Time, 10),
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			f := dataframe.NewField(t.Name(), nil, tt.Values)

			if f.Len() != len(tt.Values) {
				t.Error(f.Len())
			}

			for i := 0; i < f.Len(); i++ {
				got := reflect.ValueOf(f.Vector.At(i))
				want := reflect.ValueOf(tt.Values[i])

				if got != want {
					t.Error(got, want)
				}
			}

		})
	}
}

func TestAppendRowSafe(t *testing.T) {
	tests := []struct {
		name          string
		frame         *dataframe.Frame
		rowToAppend   []interface{}
		shouldErr     require.ErrorAssertionFunc
		errorContains []string
		newFrame      *dataframe.Frame
	}{
		{
			name:        "simple safe append",
			frame:       dataframe.New("test", dataframe.NewField("test", nil, []int64{})),
			rowToAppend: append(make([]interface{}, 0), int64(1)),
			shouldErr:   require.NoError,
			newFrame:    dataframe.New("test", dataframe.NewField("test", nil, []int64{1})),
		},
		{
			name:        "untyped nil append",
			frame:       dataframe.New("test", dataframe.NewField("test", nil, []*int64{})),
			rowToAppend: append(make([]interface{}, 0), nil),
			shouldErr:   require.NoError,
			newFrame:    dataframe.New("test", dataframe.NewField("test", nil, []*int64{nil})),
		},
		{
			name:          "untyped nil append to non-nullable should error",
			frame:         dataframe.New("test", dataframe.NewField("test", nil, []int64{})),
			rowToAppend:   append(make([]interface{}, 0), nil),
			shouldErr:     require.Error,
			errorContains: []string{"non-nullable", "underlying type []int64"},
		},
		{
			name:        "typed nil append",
			frame:       dataframe.New("test", dataframe.NewField("test", nil, []*int64{})),
			rowToAppend: append(make([]interface{}, 0), []*int64{nil}[0]),
			shouldErr:   require.NoError,
			newFrame:    dataframe.New("test", dataframe.NewField("test", nil, []*int64{nil})),
		},
		{
			name:        "wrong typed nil append should error",
			frame:       dataframe.New("test", dataframe.NewField("test", nil, []*int64{})),
			rowToAppend: append(make([]interface{}, 0), []*string{nil}[0]),
			shouldErr:   require.Error,
		},
		{
			name:          "append of wrong type should error",
			frame:         dataframe.New("test", dataframe.NewField("test", nil, []int64{})),
			rowToAppend:   append(make([]interface{}, 0), "1"),
			shouldErr:     require.Error,
			errorContains: []string{"string", "int64"},
		},
		{
			name:        "unsupported type should error",
			frame:       dataframe.New("test", dataframe.NewField("test", nil, []int64{})),
			rowToAppend: append(make([]interface{}, 0), dataframe.Frame{}),
			shouldErr:   require.Error,
		},
		{
			name:          "frame with no fields should error when appending a value",
			frame:         &dataframe.Frame{Name: "test"},
			rowToAppend:   append(make([]interface{}, 0), 1),
			shouldErr:     require.Error,
			errorContains: []string{"0 fields"},
		},
		{
			name:          "frame with uninitalized Field should error",
			frame:         &dataframe.Frame{Name: "test", Fields: []*dataframe.Field{nil}},
			rowToAppend:   append(make([]interface{}, 0), 1),
			shouldErr:     require.Error,
			errorContains: []string{"uninitalized Field at"},
		},
		{
			name:          "frame with uninitalized Field Vector should error",
			frame:         &dataframe.Frame{Name: "test", Fields: []*dataframe.Field{&dataframe.Field{}}},
			rowToAppend:   append(make([]interface{}, 0), 1),
			shouldErr:     require.Error,
			errorContains: []string{"uninitalized Field Vector at"},
		},
		{
			name:          "invalid vals type mixture",
			frame:         dataframe.New("test", dataframe.NewField("test", nil, []int64{}), dataframe.NewField("test-string", nil, []int64{})),
			rowToAppend:   append(append(make([]interface{}, 0), int64(1)), "foo"),
			shouldErr:     require.Error,
			errorContains: []string{"invalid type appending row at index 1, got string want int64"},
		},
		{
			name:        "valid vals type mixture",
			frame:       dataframe.New("test", dataframe.NewField("test", nil, []int64{}), dataframe.NewField("test-string", nil, []string{})),
			rowToAppend: append(append(make([]interface{}, 0), int64(1)), "foo"),
			shouldErr:   require.NoError,
			newFrame:    dataframe.New("test", dataframe.NewField("test", nil, []int64{1}), dataframe.NewField("test-string", nil, []string{"foo"})),
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			err := tt.frame.AppendRowSafe(tt.rowToAppend...)
			tt.shouldErr(t, err)
			for _, v := range tt.errorContains {
				require.Contains(t, err.Error(), v)
			}
			if err == nil {
				require.Equal(t, tt.frame, tt.newFrame)
			}
		})
	}

}

func timePtr(t time.Time) *time.Time {
	return &t
}

func float32Ptr(f float32) *float32 {
	return &f
}

func float64Ptr(f float64) *float64 {
	return &f
}

func int8Ptr(i int8) *int8 {
	return &i
}

func int16Ptr(i int16) *int16 {
	return &i
}

func int32Ptr(i int32) *int32 {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}

func uint8Ptr(ui uint8) *uint8 {
	return &ui
}

func uint16Ptr(ui uint16) *uint16 {
	return &ui
}

func uint32Ptr(ui uint32) *uint32 {
	return &ui
}

func uint64Ptr(ui uint64) *uint64 {
	return &ui
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func ExampleSQLStringConverter() {
	_ = dataframe.SQLStringConverter{
		Name:          "BIGINT to *int64",
		InputScanKind: reflect.Struct,
		InputTypeName: "BIGINT",
		Replacer: &dataframe.StringFieldReplacer{
			VectorType: []*int64{},
			ReplaceFunc: func(in *string) (interface{}, error) {
				if in == nil {
					return nil, nil
				}
				v, err := strconv.ParseInt(*in, 10, 64)
				if err != nil {
					return nil, err
				}
				return &v, nil
			},
		},
	}
}

func ExampleStringFieldReplacer() {
	_ = &dataframe.StringFieldReplacer{
		VectorType: []*int64{},
		ReplaceFunc: func(in *string) (interface{}, error) {
			if in == nil {
				return nil, nil
			}
			v, err := strconv.ParseInt(*in, 10, 64)
			if err != nil {
				return nil, err
			}
			return &v, nil
		},
	}
}

func ExampleNewFromSQLRows() {
	aQuery := "SELECT * FROM GoodData"
	db, err := sql.Open("fancySql", "fancysql://user:pass@localhost:1433")
	if err != nil {
		// return err
	}

	defer db.Close()

	rows, err := db.Query(aQuery)
	if err != nil {
		// return err
	}
	defer rows.Close()

	frame, mappings, err := dataframe.NewFromSQLRows(rows)
	if err != nil {
		// return err
	}
	_, _ = frame, mappings
}
