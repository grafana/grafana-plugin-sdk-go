package data_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

func TestSafeFrameAppendRow(t *testing.T) {
	tests := []struct {
		name          string
		frame         *data.Frame
		rowToAppend   []interface{}
		shouldErr     require.ErrorAssertionFunc
		errorContains []string
		newFrame      *data.Frame
	}{
		{
			name:        "simple safe append",
			frame:       data.NewFrame("test", data.NewField("test", nil, []int64{})),
			rowToAppend: append(make([]interface{}, 0), int64(1)),
			shouldErr:   require.NoError,
			newFrame:    data.NewFrame("test", data.NewField("test", nil, []int64{1})),
		},
		{
			name:        "untyped nil append",
			frame:       data.NewFrame("test", data.NewField("test", nil, []*int64{})),
			rowToAppend: append(make([]interface{}, 0), nil),
			shouldErr:   require.NoError,
			newFrame:    data.NewFrame("test", data.NewField("test", nil, []*int64{nil})),
		},
		{
			name:          "untyped nil append to non-nullable should error",
			frame:         data.NewFrame("test", data.NewField("test", nil, []int64{})),
			rowToAppend:   append(make([]interface{}, 0), nil),
			shouldErr:     require.Error,
			errorContains: []string{"non-nullable", "underlying type []int64"},
		},
		{
			name:        "typed nil append",
			frame:       data.NewFrame("test", data.NewField("test", nil, []*int64{})),
			rowToAppend: append(make([]interface{}, 0), []*int64{nil}[0]),
			shouldErr:   require.NoError,
			newFrame:    data.NewFrame("test", data.NewField("test", nil, []*int64{nil})),
		},
		{
			name:        "wrong typed nil append should error",
			frame:       data.NewFrame("test", data.NewField("test", nil, []*int64{})),
			rowToAppend: append(make([]interface{}, 0), []*string{nil}[0]),
			shouldErr:   require.Error,
		},
		{
			name:          "append of wrong type should error",
			frame:         data.NewFrame("test", data.NewField("test", nil, []int64{})),
			rowToAppend:   append(make([]interface{}, 0), "1"),
			shouldErr:     require.Error,
			errorContains: []string{"string", "int64"},
		},
		{
			name:        "unsupported type should error",
			frame:       data.NewFrame("test", data.NewField("test", nil, []int64{})),
			rowToAppend: append(make([]interface{}, 0), data.Frame{}),
			shouldErr:   require.Error,
		},
		{
			name:          "frame with no fields should error when appending a value",
			frame:         &data.Frame{Name: "test"},
			rowToAppend:   append(make([]interface{}, 0), 1),
			shouldErr:     require.Error,
			errorContains: []string{"0 fields"},
		},
		{
			name:          "frame with uninitalized Field should error",
			frame:         &data.Frame{Name: "test", Fields: []*data.Field{nil}},
			rowToAppend:   append(make([]interface{}, 0), 1),
			shouldErr:     require.Error,
			errorContains: []string{"uninitalized Field at"},
		},
		{
			name:          "frame with uninitalized Field Vector should error",
			frame:         &data.Frame{Name: "test", Fields: []*data.Field{&data.Field{}}},
			rowToAppend:   append(make([]interface{}, 0), 1),
			shouldErr:     require.Error,
			errorContains: []string{"uninitalized Field at"},
		},
		{
			name:          "invalid vals type mixture",
			frame:         data.NewFrame("test", data.NewField("test", nil, []int64{}), data.NewField("test-string", nil, []int64{})),
			rowToAppend:   append(append(make([]interface{}, 0), int64(1)), "foo"),
			shouldErr:     require.Error,
			errorContains: []string{"invalid type appending row at index 1, got string want int64"},
		},
		{
			name:        "valid vals type mixture",
			frame:       data.NewFrame("test", data.NewField("test", nil, []int64{}), data.NewField("test-string", nil, []string{})),
			rowToAppend: append(append(make([]interface{}, 0), int64(1)), "foo"),
			shouldErr:   require.NoError,
			newFrame:    data.NewFrame("test", data.NewField("test", nil, []int64{1}), data.NewField("test-string", nil, []string{"foo"})),
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			// TODO: FIX kludge assertion, use Safe instead
			err := (*data.SafeFrame)(tt.frame).AppendRow(tt.rowToAppend...)
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
