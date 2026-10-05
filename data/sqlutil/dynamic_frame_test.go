package sqlutil

import (
	"database/sql"
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func TestDynamicFrame(t *testing.T) {
	kind := &sql.ColumnType{}
	types := []*sql.ColumnType{}
	types = append(types, kind)
	converters := []Converter{}
	data := [][]any{}
	mockRow := []any{}
	val := string("foo")
	mockRow = append(mockRow, val)
	mockRow2 := []any{}
	mockRow2 = append(mockRow2, "bar")
	data = append(data, mockRow)
	data = append(data, mockRow2)
	mock := &MockRows{
		data:  data,
		index: -1,
	}
	rows := Rows{
		itr: mock,
	}

	_, converters = removeDynamicConverter(converters)
	frame, err := frameDynamic(rows, 100, types, converters)
	assert.Nil(t, err)
	assert.NotNil(t, frame)

	assert.Equal(t, 2, frame.Rows())

	actual := frame.Fields[0].At(0).(*string)
	assert.Equal(t, val, *actual)

	actual = frame.Fields[0].At(1).(*string)
	assert.Equal(t, "bar", *actual)
}

type MockRows struct {
	data  [][]any
	index int
}

func (rs *MockRows) Next() bool {
	rs.index++
	return rs.index < len(rs.data)
}

func (rs *MockRows) Scan(dest ...any) error {
	data := rs.data[rs.index]
	for i, d := range dest {
		foo := d.(*any)
		val := reflect.ValueOf(foo)
		if val.Kind() != reflect.Ptr { //nolint:govet // inline analyzer false positive on reflect.Ptr alias
			panic("val must be a pointer")
		}
		val.Elem().Set(reflect.ValueOf(data[i]))
	}
	return nil
}

func TestDynamicFrameShouldNotPanic(t *testing.T) {
	kind := &sql.ColumnType{}
	types := []*sql.ColumnType{}
	types = append(types, kind)
	converters := []Converter{dynamic()}
	data := [][]any{}
	mockRow := []any{}
	val := string("foo")
	mockRow = append(mockRow, val)
	data = append(data, mockRow)
	mock := &MockRows{
		data:  data,
		index: -1,
	}
	rows := Rows{
		itr: mock,
	}

	_, converters = removeDynamicConverter(converters)
	frame, err := frameDynamic(rows, 100, types, converters)
	assert.Nil(t, err)
	assert.NotNil(t, frame)

	assert.Equal(t, 1, frame.Rows())

	actual := frame.Fields[0].At(0).(*string)
	assert.Equal(t, val, *actual)
}

// dynamic is the converter that uses the results to determine data types
func dynamic() Converter {
	kind := "dynamic"
	return Converter{
		Name:          kind,
		InputTypeName: kind,
		Dynamic:       true,
	}
}

func TestDynamicFrameRowLimit(t *testing.T) {
	input := []string{"a", "b", "c", "d", "e"}
	tests := []struct {
		name          string
		rowLimit      int64
		wantRows      int
		wantNextCalls int
		wantNotice    bool
	}{
		{name: "zero limit reads no rows", rowLimit: 0, wantRows: 0, wantNextCalls: 0, wantNotice: false},
		{name: "limit below row count truncates and warns", rowLimit: 2, wantRows: 2, wantNextCalls: 2, wantNotice: true},
		{name: "limit equal to row count warns", rowLimit: 5, wantRows: 5, wantNextCalls: 5, wantNotice: true},
		{name: "limit above row count keeps all rows", rowLimit: 10, wantRows: 5, wantNextCalls: 6, wantNotice: false},
		{name: "no limit keeps all rows", rowLimit: math.MaxInt64, wantRows: 5, wantNextCalls: 6, wantNotice: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var raw [][]any
			for _, v := range input {
				raw = append(raw, []any{v})
			}
			mock := &MockRows{data: raw, index: -1}

			frame, err := frameDynamic(Rows{itr: mock}, tc.rowLimit, []*sql.ColumnType{{}}, nil)
			require.NoError(t, err)

			require.Equal(t, tc.wantRows, frame.Rows())
			assert.Equal(t, tc.wantNextCalls, mock.index+1, "rows.Next() calls")
			for j := 0; j < tc.wantRows; j++ {
				assert.Equal(t, input[j], *frame.Fields[0].At(j).(*string))
			}
			if tc.wantNotice {
				require.NotNil(t, frame.Meta)
				assert.Equal(t, []data.Notice{rowLimitNotice(tc.rowLimit)}, frame.Meta.Notices)
			} else {
				assert.Nil(t, frame.Meta)
			}
		})
	}
}
