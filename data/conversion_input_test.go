package data_test

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func ExampleNewFrameConverterBuilder() {
	inputData := struct {
		ColumnTypes []string
		ColumnNames []string
		Rows        [][]string
	}{
		[]string{
			"Stringz",
			"Floatz",
			"Timez",
		},
		[]string{
			"Animal",
			"Weight (lbs)",
			"Time",
		},
		[][]string{
			[]string{"sloth", "3.5", "1586014367"},
			[]string{"sloth", "5.5", "1586100767"},
			[]string{"sloth", "7", "1586187167"},
		},
	}

	stringzFieldConverter := data.FieldConverter{
		OutputFieldType: data.FieldTypeString,
		Converter: func(v interface{}) (interface{}, error) {
			return v, nil
		},
	}

	floatzFieldConverter := data.FieldConverter{
		OutputFieldType: data.FieldTypeFloat64,
		Converter: func(v interface{}) (interface{}, error) {
			val, ok := v.(string)
			if !ok { // or return some default value instead of erroring
				return nil, fmt.Errorf("expected string input but got type %T", val)
			}
			return strconv.ParseFloat(val, 64)
		},
	}

	timezFieldConverter := data.FieldConverter{
		OutputFieldType: data.FieldTypeTime,
		Converter: func(v interface{}) (interface{}, error) {
			val, ok := v.(string)
			if !ok { // or return some default value instead of erroring
				return nil, fmt.Errorf("expected string input but got type %T", val)
			}
			iV, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("could not parse epoch time into an int64")
			}
			return time.Unix(iV, 0).UTC(), nil
		},
	}

	converterMap := map[string]data.FieldConverter{
		"Stringz": stringzFieldConverter,
		"Floatz":  floatzFieldConverter,
		"Timez":   timezFieldConverter,
	}

	converters := make([]data.FieldConverter, len(inputData.ColumnTypes))
	for i, ct := range inputData.ColumnTypes {
		fc, ok := converterMap[ct]
		if !ok {
			fc = data.AsStringFieldConverter
		}
		converters[i] = fc
	}

	convBuilder, err := data.NewFrameInputConverter(converters, len(inputData.Rows))
	if err != nil {
		log.Fatal(err)
	}

	err = convBuilder.Frame.SetFieldNames(inputData.ColumnNames...)
	if err != nil {
		log.Fatal(err)
	}

	for rowIdx, row := range inputData.Rows {
		for fieldIdx, cell := range row {
			convBuilder.Set(fieldIdx, rowIdx, cell)
		}
	}
	convBuilder.Frame.Name = "Converted"

	fmt.Println(convBuilder.Frame.String())
	// Output:
	// Name: Converted
	// Dimensions: 3 Fields by 3 Rows
	// +----------------+--------------------+-------------------------------+
	// | Name: Animal   | Name: Weight (lbs) | Name: Time                    |
	// | Labels:        | Labels:            | Labels:                       |
	// | Type: []string | Type: []float64    | Type: []time.Time             |
	// +----------------+--------------------+-------------------------------+
	// | sloth          | 3.5                | 2020-04-04 15:32:47 +0000 UTC |
	// | sloth          | 5.5                | 2020-04-05 15:32:47 +0000 UTC |
	// | sloth          | 7                  | 2020-04-06 15:32:47 +0000 UTC |
	// +----------------+--------------------+-------------------------------+
}
