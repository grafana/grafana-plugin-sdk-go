package data_test

import (
	"database/sql"
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

func ExampleStringFieldReplacer() {
	_ = &data.StringFieldReplacer{
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

	frame, mappings, err := data.NewFromSQLRows(rows)
	if err != nil {
		// return err
	}
	_, _ = frame, mappings
}
