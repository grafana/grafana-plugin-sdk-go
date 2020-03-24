package data_test

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func ExampleSQLStringConverter() {
	_ = data.SQLStringConverter{
		Name:          "BIGINT to *int64",
		InputScanKind: reflect.Struct,
		InputTypeName: "BIGINT",
		Replacer: &data.StringFieldReplacer{
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

func ExampleReplace() {
	i := 0
	getString := func() *string {
		i++
		s := strconv.Itoa(i)
		return &s
	}

	frame := data.NewFrame("Before",
		data.NewField("string", nil, []*string{getString(), getString()}))

	fmt.Println(frame.String()) // Before

	intReplacer := &data.StringFieldReplacer{
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

	err := data.Replace(frame, 0, intReplacer)
	if err != nil {
		// return err
	}

	frame.Name = "After"
	fmt.Println(frame.String()) // After
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

	frame, mappings, err := data.NewFromSQLRows(rows)
	if err != nil {
		// return err
	}
	_, _ = frame, mappings
}
