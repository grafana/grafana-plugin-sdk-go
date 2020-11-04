package sqlutil_test

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
)

func ExampleStringConverter() {
	_ = sqlutil.StringConverter{
		Name:          "BIGINT to *int64",
		InputScanKind: reflect.Struct,
		InputTypeName: "BIGINT",
		Replacer: &sqlutil.StringFieldReplacer{
			OutputFieldType: data.FieldTypeNullableInt64,
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

func ExampleReplace() {
	i := 0
	getString := func() *string {
		i++
		s := strconv.Itoa(i)
		return &s
	}

	frame := data.NewFrame("Before",
		data.NewField("string", nil, []*string{getString(), getString()}))

	st, _ := frame.StringTable(-1, -1)
	fmt.Println(st)

	intReplacer := &sqlutil.StringFieldReplacer{
		OutputFieldType: data.FieldTypeNullableInt64,
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

	err := sqlutil.Replace(frame, 0, intReplacer)
	if err != nil {
		// return err
	}

	frame.Name = "After"
	st, _ = frame.StringTable(-1, -1)
	fmt.Println(st) // After
	// Output:
	// Name: Before
	// Dimensions: 1 Fields by 2 Rows
	// +-----------------+
	// | Name: string    |
	// | Labels:         |
	// | Type: []*string |
	// +-----------------+
	// | 1               |
	// | 2               |
	// +-----------------+
	//
	// Name: After
	// Dimensions: 1 Fields by 2 Rows
	// +----------------+
	// | Name: string   |
	// | Labels:        |
	// | Type: []*int64 |
	// +----------------+
	// | 1              |
	// | 2              |
	// +----------------+
}

func ExampleFrameFromRows() {
	aQuery := "SELECT * FROM GoodData"
	db, err := sql.Open("fancySql", "fancysql://user:pass@localhost:1433")
	if err != nil {
		return
	}

	defer func() {
		_ = db.Close()
	}()

	// For some reason the rowserrcheck linter doesn't catch that rows.Err is checked
	// nolint: rowserrcheck
	rows, err := db.Query(aQuery)
	if err != nil {
		return
	}
	if rows.Err() != nil {
		return
	}
	defer func() {
		_ = rows.Close()
	}()

	frame, mappings, err := sqlutil.FrameFromRows(rows, 1000)
	if err != nil {
		return
	}
	_, _ = frame, mappings
}
