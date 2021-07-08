package sqlutil_test

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConverter(t *testing.T) {
	type Suite struct {
		Name     string
		Type     reflect.Type
		Nullable bool
		Expected sqlutil.Converter
	}

	suite := []Suite{
		{
			Name:     "non-nullable type",
			Type:     reflect.TypeOf(int64(0)),
			Nullable: false,
			Expected: sqlutil.Converter{
				InputScanType: reflect.TypeOf(int64(0)),
				FrameConverter: sqlutil.FrameConverter{
					FieldType: data.FieldTypeInt64,
				},
			},
		},
		{
			Name:     "nullable int64",
			Type:     reflect.TypeOf(int64(0)),
			Nullable: true,
			Expected: sqlutil.Converter{
				InputScanType: reflect.TypeOf(sql.NullInt64{}),
				FrameConverter: sqlutil.FrameConverter{
					FieldType: data.FieldTypeInt64.NullableType(),
				},
			},
		},
		{
			Name:     "string",
			Type:     reflect.TypeOf(""),
			Nullable: false,
			Expected: sqlutil.Converter{
				InputScanType: reflect.TypeOf(""),
				FrameConverter: sqlutil.FrameConverter{
					FieldType: data.FieldTypeString,
				},
			},
		},
		{
			Name:     "nullable string",
			Type:     reflect.TypeOf(""),
			Nullable: true,
			Expected: sqlutil.Converter{
				InputScanType: reflect.TypeOf(sql.NullString{}),
				FrameConverter: sqlutil.FrameConverter{
					FieldType: data.FieldTypeString.NullableType(),
				},
			},
		},
		{
			Name:     "string",
			Type:     reflect.TypeOf(time.Time{}),
			Nullable: false,
			Expected: sqlutil.Converter{
				InputScanType: reflect.TypeOf(time.Time{}),
				FrameConverter: sqlutil.FrameConverter{
					FieldType: data.FieldTypeTime,
				},
			},
		},
		{
			Name:     "nullable time",
			Type:     reflect.TypeOf(time.Time{}),
			Nullable: true,
			Expected: sqlutil.Converter{
				InputScanType: reflect.TypeOf(sql.NullTime{}),
				FrameConverter: sqlutil.FrameConverter{
					FieldType: data.FieldTypeTime.NullableType(),
				},
			},
		},
		{
			Name:     "nullable bool",
			Type:     reflect.TypeOf(false),
			Nullable: true,
			Expected: sqlutil.Converter{
				InputScanType: reflect.TypeOf(sql.NullBool{}),
				FrameConverter: sqlutil.FrameConverter{
					FieldType: data.FieldTypeBool.NullableType(),
				},
			},
		},
	}

	for i, v := range suite {
		t.Run(fmt.Sprintf("[%d/%d] %s", i+1, len(suite), v.Name), func(t *testing.T) {
			c := sqlutil.NewDefaultConverter(v.Name, v.Nullable, v.Type)
			assert.Equal(t, c.FrameConverter.FieldType, v.Expected.FrameConverter.FieldType)
			assert.Equal(t, c.InputScanType.String(), v.Expected.InputScanType.String())

			t.Run("When the converter is called, the expected type should be returned", func(t *testing.T) {
				n := reflect.New(v.Expected.InputScanType).Interface()
				value, err := c.FrameConverter.ConverterFunc(n)
				assert.NoError(t, err)

				if !v.Nullable {
					// non-nullable fields should exactly match
					assert.Equal(t, reflect.TypeOf(value).String(), v.Type.String())
				} else {
					// nullable fields should not exactly match
					assert.Equal(t, reflect.TypeOf(value).String(), reflect.PtrTo(v.Type).String())
				}
			})
		})
	}
}
