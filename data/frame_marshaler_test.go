package data

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type tcase struct {
	description string
	name        string
	value       interface{}
	fields      []MarshalField
	shouldError bool
	err         error
	frame       *Frame
}

func TestMarshalBasic(t *testing.T) {
	type timeseriesExample struct {
		Timestamp time.Time `frame:"ts"`
		Value     float64   `frame:"value"`
	}

	tc := tcase{
		description: "should convert a basic timeseries example",
		name:        "frame",
		value: []timeseriesExample{
			{
				Timestamp: time.Now(),
				Value:     1,
			},
			{
				Timestamp: time.Now().Add(time.Second),
				Value:     2,
			},
		},
		frame: NewFrame("frame",
			NewField("ts", nil, []time.Time{time.Now(), time.Now().Add(time.Second)}),
			NewField("value", nil, []float64{1, 2}),
		),
	}

	testMarshalCase(t, tc)
}

func TestMarshalEmbeddedTimesries(t *testing.T) {
	type embeddedDimension struct {
		Descriptor string
		Value      float64 `frame:"val"`
	}
	type embeddedTimeseriesData struct {
		Timestamp time.Time `frame:"ts"`
		Dimension embeddedDimension
	}

	tc := tcase{
		description: "should convert a basic timeseries example with embedded data",
		name:        "frame",
		value: []embeddedTimeseriesData{
			{
				Timestamp: time.Now(),
				Dimension: embeddedDimension{
					Descriptor: "1",
					Value:      1,
				},
			},
			{
				Timestamp: time.Now().Add(time.Second),
				Dimension: embeddedDimension{
					Descriptor: "1",
					Value:      1,
				},
			},
		},
		frame: NewFrame("frame",
			NewField("ts", nil, []time.Time{time.Now(), time.Now().Add(time.Second)}),
			NewField("val", nil, []float64{0, 1}),
		),
	}

	testMarshalCase(t, tc)
}

func TestMarshalTable(t *testing.T) {
	type table struct {
		Field1 string `frame:"field1"`
		Field2 string `frame:"field2"`
		Field3 string `frame:"field3"`
	}
	tc := tcase{
		description: "should convert a basic timeseries example with embedded data",
		name:        "frame",
		fields: []MarshalField{
			{Name: "field1"},
			{Name: "field2"},
			{Name: "field3"},
		},
		value: []table{
			{
				Field1: "1",
				Field2: "2",
				Field3: "3",
			},
			{
				Field1: "1",
				Field2: "2",
				Field3: "3",
			},
		},
		frame: NewFrame("frame",
			NewField("field1", nil, []string{"1", "1"}),
			NewField("field2", nil, []string{"2", "2"}),
			NewField("field3", nil, []string{"3", "3"}),
		),
	}

	testMarshalCase(t, tc)
}

func TestMarshalAllTypes(t *testing.T) {
	type typeTable struct {
		Int8       int8       `frame:"int_8"`
		Int8Ptr    *int8      `frame:"int_8_ptr"`
		Int16      int16      `frame:"int_16"`
		Int16Ptr   *int16     `frame:"int_16_ptr"`
		Int32      int32      `frame:"int_32"`
		Int32Ptr   *int32     `frame:"int_32_ptr"`
		Int64      int64      `frame:"int_64"`
		Int64Ptr   *int64     `frame:"int_64_ptr"`
		Uint8      int8       `frame:"uint_8"`
		Uint8Ptr   *uint8     `frame:"uint_8_ptr"`
		Uint16     uint16     `frame:"uint_16"`
		Uint16Ptr  *uint16    `frame:"uint_16_ptr"`
		Uint32     uint32     `frame:"uint_32"`
		Uint32Ptr  *uint32    `frame:"uint_32_ptr"`
		Uint64     uint64     `frame:"uint_64"`
		Uint64Ptr  *uint64    `frame:"uint_64_ptr"`
		Float32    float32    `frame:"float_32"`
		Float32Ptr *float32   `frame:"float_32_ptr"`
		Float64    float64    `frame:"float_64"`
		Float64Ptr *float64   `frame:"float_64_ptr"`
		String     string     `frame:"string"`
		StringPtr  *string    `frame:"string_ptr"`
		Bool       bool       `frame:"bool"`
		BoolPtr    *bool      `frame:"bool_ptr"`
		Time       time.Time  `frame:"time"`
		TimePtr    *time.Time `frame:"time_ptr"`
	}

	tc := tcase{
		description: "should convert a basic timeseries example with embedded data",
		name:        "frame",
		value:       []typeTable{},
		frame: NewFrame("frame",
			NewField("int_8", nil, []int8{}),
			NewField("int_8_ptr", nil, []*int8{}),
			NewField("int_16", nil, []int16{}),
			NewField("int_16_ptr", nil, []*int16{}),
			NewField("int_32", nil, []int32{}),
			NewField("int_32_ptr", nil, []*int32{}),
			NewField("int_64", nil, []int64{}),
			NewField("int_64_ptr", nil, []*int64{}),
			NewField("uint_8", nil, []uint8{}),
			NewField("uint_8_ptr", nil, []*uint8{}),
			NewField("uint_16", nil, []uint16{}),
			NewField("uint_16_ptr", nil, []*uint16{}),
			NewField("uint_32", nil, []uint32{}),
			NewField("uint_32_ptr", nil, []*uint32{}),
			NewField("uint_64", nil, []uint64{}),
			NewField("uint_64_ptr", nil, []*uint64{}),
			NewField("float_32", nil, []float32{}),
			NewField("float_32_ptr", nil, []*float32{}),
			NewField("float_64", nil, []float64{}),
			NewField("float_64_ptr", nil, []*float64{}),
			NewField("string", nil, []string{}),
			NewField("string_ptr", nil, []*string{}),
			NewField("bool", nil, []bool{}),
			NewField("bool_ptr", nil, []*bool{}),
			NewField("time", nil, []time.Time{}),
			NewField("time_ptr", nil, []*time.Time{}),
		),
	}

	testMarshalCase(t, tc)
}

func TestMarshalWithLabels(t *testing.T) {
	type typeTable struct {
		Int8     int8   `frame:"int_8"`
		Int8Ptr  *int8  `frame:"int_8_ptr"`
		Int16    int16  `frame:"int_16"`
		Int16Ptr int16  `frame:"int_16"`
		Label    string `frame:"label,label"`
	}

	tc := tcase{
		description: "should convert a basic timeseries example with embedded data",
		name:        "frame",
		value:       []typeTable{},
	}

	testMarshalCase(t, tc)
}

func testMarshalCase(t *testing.T, tc tcase) {
	t.Run(tc.description, func(t *testing.T) {
		frame, err := Marshal(tc.name, tc.fields, tc.value)
		if err != nil && !tc.shouldError {
			t.Fatal("got unexpected error", err)
		}

		if err == nil && tc.shouldError {
			t.Fatal("expected error but did not receive one")
		}

		if err != nil && tc.shouldError {
			if !errors.Is(err, tc.err) {
				t.Fatal("expected error but received wrong type", err)
			}
			return
		}

		if diff := cmp.Diff(Frames{frame}, Frames{tc.frame}, FrameTestCompareOptions()...); diff != "" {
			t.Errorf("Result mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestMarshal(t *testing.T) {
	t.Run("should return an error if a slice element is not a struct or a map", func(t *testing.T) {
		d := []string{"not a struct or map"}
		_, err := Marshal("test", nil, d)
		if err == nil {
			t.Fatal("no error returned")
		}
		if !errors.Is(err, ErrorNotCollection) {
			t.Fatalf("error '%s' is not an ErrorNotCollection", err)
		}
	})
}

func BenchmarkMarshal(b *testing.B) {
}
