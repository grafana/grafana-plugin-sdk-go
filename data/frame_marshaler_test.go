package data

import (
	"errors"
	"reflect"
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
		Timestamp    time.Time  `frame:"ts"`
		TimestampPtr *time.Time `frame:"ts2"`
		Value        float64    `frame:"value"`
	}

	now := time.Now()
	tc := tcase{
		description: "should convert a basic timeseries example",
		name:        "frame",
		shouldError: false,
		value: []timeseriesExample{
			{
				Timestamp:    now,
				TimestampPtr: &now,
				Value:        1,
			},
			{
				Timestamp:    now.Add(time.Second),
				TimestampPtr: &now,
				Value:        2,
			},
		},

		fields: MarshalFields("ts", "ts2", "value"),

		frame: NewFrame("frame",
			NewField("ts", nil, []time.Time{now, now.Add(time.Second)}),
			NewField("ts2", nil, []*time.Time{&now, &now}),
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

	now := time.Now()
	tc := tcase{
		description: "should convert a basic timeseries example with embedded data",
		name:        "frame",
		value: []embeddedTimeseriesData{
			{
				Timestamp: now,
				Dimension: embeddedDimension{
					Descriptor: "1",
					Value:      0,
				},
			},
			{
				Timestamp: now.Add(time.Second),
				Dimension: embeddedDimension{
					Descriptor: "1",
					Value:      1,
				},
			},
		},
		frame: NewFrame("frame",
			NewField("ts", nil, []time.Time{now, now.Add(time.Second)}),
			NewField("val", nil, []float64{0, 1}),
		),
		fields: []MarshalField{
			{Name: "ts"},
			{Name: "Dimension.val", Alias: "val"},
		},
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
		Uint8      uint8      `frame:"uint_8"`
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
		description: "should convert a complex example with a every possible type",
		name:        "frame",
		value:       []typeTable{{}},
		frame: NewFrame("frame",
			NewField("int_8", nil, make([]int8, 1)),
			NewField("int_8_ptr", nil, make([]*int8, 1)),
			NewField("int_16", nil, make([]int16, 1)),
			NewField("int_16_ptr", nil, make([]*int16, 1)),
			NewField("int_32", nil, make([]int32, 1)),
			NewField("int_32_ptr", nil, make([]*int32, 1)),
			NewField("int_64", nil, make([]int64, 1)),
			NewField("int_64_ptr", nil, make([]*int64, 1)),
			NewField("uint_8", nil, make([]uint8, 1)),
			NewField("uint_8_ptr", nil, make([]*uint8, 1)),
			NewField("uint_16", nil, make([]uint16, 1)),
			NewField("uint_16_ptr", nil, make([]*uint16, 1)),
			NewField("uint_32", nil, make([]uint32, 1)),
			NewField("uint_32_ptr", nil, make([]*uint32, 1)),
			NewField("uint_64", nil, make([]uint64, 1)),
			NewField("uint_64_ptr", nil, make([]*uint64, 1)),
			NewField("float_32", nil, make([]float32, 1)),
			NewField("float_32_ptr", nil, make([]*float32, 1)),
			NewField("float_64", nil, make([]float64, 1)),
			NewField("float_64_ptr", nil, make([]*float64, 1)),
			NewField("string", nil, make([]string, 1)),
			NewField("string_ptr", nil, make([]*string, 1)),
			NewField("bool", nil, make([]bool, 1)),
			NewField("bool_ptr", nil, make([]*bool, 1)),
			NewField("time", nil, make([]time.Time, 1)),
			NewField("time_ptr", nil, make([]*time.Time, 1)),
		),
		fields: MarshalFields("int_8", "int_8_ptr", "int_16", "int_16_ptr", "int_32", "int_32_ptr", "int_64", "int_64_ptr", "uint_8", "uint_8_ptr", "uint_16", "uint_16_ptr", "uint_32", "uint_32_ptr", "uint_64", "uint_64_ptr", "float_32", "float_32_ptr", "float_64", "float_64_ptr", "string", "string_ptr", "bool", "bool_ptr", "time", "time_ptr"),
	}

	testMarshalCase(t, tc)
}

// func TestMarshalWithLabels(t *testing.T) {
// 	type typeTable struct {
// 		Int8  int8   `frame:"int_8"`
// 		Int16 int16  `frame:"int_16"`
// 		Host  string `frame:"host,label"`
// 	}
//
// 	tc := tcase{
// 		description: "should convert a basic timeseries example with embedded data",
// 		name:        "frame",
// 		value: []typeTable{
// 			{
// 				Int8:  int8(13),
// 				Int16: int16(22),
// 				Host:  "server_1",
// 			},
// 			{
// 				Int8:  int8(22),
// 				Int16: int16(32),
// 				Host:  "server_1",
// 			},
// 			{
// 				Int8:  int8(32),
// 				Int16: int16(2),
// 				Host:  "server_2",
// 			},
// 			{
// 				Int8:  int8(35),
// 				Int16: int16(13),
// 				Host:  "server_2",
// 			},
// 		},
// 		fields: MarshalFields("int8", "int_16"),
// 		frame: NewFrame("frame",
// 			NewField("int_8", Labels{
// 				"host": "server_1",
// 			}, []int8{13, 22}),
// 			NewField("int_8", Labels{
// 				"host": "server_2",
// 			}, []int8{32, 35}),
// 			NewField("int_16", Labels{
// 				"host": "server_1",
// 			}, []int16{22, 32}),
// 			NewField("int_16", Labels{
// 				"host": "server_2",
// 			}, []int16{2, 13}),
// 		),
// 	}
//
// 	testMarshalCase(t, tc)
// }

func testMarshalCase(t *testing.T, tc tcase) {
	t.Run(tc.description, func(t *testing.T) {
		frame, err := Marshal(tc.name, tc.fields, tc.value)
		if err != nil {
			if !tc.shouldError {
				t.Fatal("got unexpected error", err)
			}
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

func TestTree(t *testing.T) {
	type ex struct {
		Name      string    `frame:"name"`
		Timestamp time.Time `frame:"timestamp"`
		Value     float64   `frame:"value"`
	}

	d := []ex{
		{
			Name:      "first",
			Timestamp: time.Now(),
			Value:     100,
		},
		{
			Name:      "second",
			Timestamp: time.Now().Add(time.Minute),
			Value:     101,
		},
		{
			Name:      "third",
			Timestamp: time.Now().Add(time.Minute * 2),
			Value:     102,
		},
	}

	treeList, err := newTreeList(d)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("compare tree", func(t *testing.T) {
		expect := nodes{
			{
				Children: map[string]*node{
					"name": {
						Val: reflect.ValueOf(d[0].Name),
						T:   reflect.TypeOf(d[0].Name),
					},
					"timestamp": {
						Val: reflect.ValueOf(d[0].Timestamp),
						T:   reflect.TypeOf(d[0].Timestamp),
					},
					"value": {
						Val: reflect.ValueOf(d[0].Value),
						T:   reflect.TypeOf(d[0].Value),
					},
				},
				Val: reflect.ValueOf(d[0]),
				T:   reflect.TypeOf(d[0]),
			},
			{
				Children: map[string]*node{
					"name": {
						Val: reflect.ValueOf(d[1].Name),
						T:   reflect.TypeOf(d[1].Name),
					},
					"timestamp": {
						Val: reflect.ValueOf(d[1].Timestamp),
						T:   reflect.TypeOf(d[1].Timestamp),
					},
					"value": {
						Val: reflect.ValueOf(d[1].Value),
						T:   reflect.TypeOf(d[1].Value),
					},
				},
				Val: reflect.ValueOf(d[1]),
				T:   reflect.TypeOf(d[1]),
			},
			{
				Children: map[string]*node{
					"name": {
						Val: reflect.ValueOf(d[2].Name),
						T:   reflect.TypeOf(d[2].Name),
					},
					"timestamp": {
						Val: reflect.ValueOf(d[2].Timestamp),
						T:   reflect.TypeOf(d[2].Timestamp),
					},
					"value": {
						Val: reflect.ValueOf(d[2].Value),
						T:   reflect.TypeOf(d[2].Value),
					},
				},
				Val: reflect.ValueOf(d[2]),
				T:   reflect.TypeOf(d[2]),
			},
		}

		if len(expect) != len(treeList) {
			t.Fatalf("unexpected number of trees in list. got '%d', expected '%d'", len(expect), len(treeList))
		}

		for i, node := range expect {
			for k, v := range node.Children {
				if _, ok := treeList[i].Children[k]; !ok {
					t.Fatalf("expected child `%s', but was not found", k)
				}
				kind := treeList[i].Children[k].T.Kind()
				if v.T.Kind() != kind {
					t.Fatalf("child '%s' had unexpected type '%s', expected '%s'", k, kind, v.T.Kind())
				}
			}
		}
	})

	t.Run("get name", func(t *testing.T) {
		n, err := treeList.get("name")
		if err != nil {
			t.Fatal(err)
		}

		for i, v := range n {
			if v.T.Kind() != reflect.String {
				t.Errorf("unexpted type on tree '%d', expected '%s', got '%s'", i, reflect.String, v.T.Kind())
			}
		}
	})

	t.Run("get value", func(t *testing.T) {
		n, err := treeList.get("value")
		if err != nil {
			t.Fatal(err)
		}

		for i, v := range n {
			if v.T.Kind() != reflect.Float64 {
				t.Errorf("unexpted type on tree '%d', expected '%s', got '%s'", i, reflect.Float64, v.T.Kind())
			}
		}
	})
}

func TestComplexTree(t *testing.T) {
	type exInner struct {
		Dimension string  `frame:"dimension"`
		Value     float64 `frame:"value"`
	}

	type ex struct {
		Name   string  `frame:"name"`
		Inner1 exInner `frame:"inner_1"`
		Inner2 exInner `frame:"inner_2"`
	}
	i1 := exInner{
		Dimension: "inner1",
		Value:     102.3,
	}
	i2 := exInner{
		Dimension: "inner1",
		Value:     102.3,
	}

	d := []ex{
		{
			Name:   "ex1",
			Inner1: i1,
			Inner2: i2,
		},
		{
			Name:   "ex2",
			Inner1: i1,
			Inner2: i2,
		},
		{
			Name:   "ex3",
			Inner1: i1,
			Inner2: i2,
		},
	}

	treeList, err := newTreeList(d)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("get inner struct node", func(t *testing.T) {
		n, err := treeList.get("inner_1.value")
		if err != nil {
			t.Fatal(err)
		}
		kind := reflect.Float64
		for i, v := range n {
			if v.T.Kind() != kind {
				t.Errorf("unexpected type on tree '%d', expected '%s', got '%s'", i, kind, v.T.Kind())
			}
		}
	})
}

func BenchmarkMarshalTable_10(b *testing.B) {
	type table struct {
		Field1 string `frame:"field1"`
		Field2 string `frame:"field2"`
		Field3 string `frame:"field3"`
	}
	fields := MarshalFields("field1", "field2", "field3")
	value := make([]table, 10)

	for i := range value {
		value[i] = table{
			Field1: "field1",
			Field2: "field2",
			Field3: "field3",
		}
	}

	for i := 0; i < b.N; i++ {
		Marshal("frame", fields, value)
	}
}

func BenchmarkMarshalTable_100(b *testing.B) {
	type table struct {
		Field1 string `frame:"field1"`
		Field2 string `frame:"field2"`
		Field3 string `frame:"field3"`
	}
	fields := MarshalFields("field1", "field2", "field3")
	value := make([]table, 100)

	for i := range value {
		value[i] = table{
			Field1: "field1",
			Field2: "field2",
			Field3: "field3",
		}
	}

	for i := 0; i < b.N; i++ {
		Marshal("frame", fields, value)
	}
}

func BenchmarkMarshalTable_1000(b *testing.B) {
	type table struct {
		Field1 string `frame:"field1"`
		Field2 string `frame:"field2"`
		Field3 string `frame:"field3"`
	}
	fields := MarshalFields("field1", "field2", "field3")
	value := make([]table, 1000)

	for i := range value {
		value[i] = table{
			Field1: "field1",
			Field2: "field2",
			Field3: "field3",
		}
	}

	for i := 0; i < b.N; i++ {
		Marshal("frame", fields, value)
	}
}

func BenchmarkEmbeddedStruct_10(b *testing.B) {
	type embeddedDimension struct {
		Descriptor string
		Value      float64 `frame:"val"`
	}

	type tsd struct {
		Timestamp time.Time         `frame:"ts"`
		Dimension embeddedDimension `frame:"dimension"`
	}

	now := time.Now()
	value := make([]tsd, 10)

	for i := range value {
		value[i] = tsd{
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Dimension: embeddedDimension{
				Descriptor: "descriptor",
				Value:      123.0,
			},
		}
	}

	fields := []MarshalField{
		{Name: "ts"},
		{Name: "dimension.val", Alias: "val"},
	}

	for i := 0; i < b.N; i++ {
		Marshal("frame", fields, value)
	}
}

func BenchmarkEmbeddedStruct_100(b *testing.B) {
	type embeddedDimension struct {
		Descriptor string
		Value      float64 `frame:"val"`
	}

	type tsd struct {
		Timestamp time.Time         `frame:"ts"`
		Dimension embeddedDimension `frame:"dimension"`
	}

	now := time.Now()
	value := make([]tsd, 100)

	for i := range value {
		value[i] = tsd{
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Dimension: embeddedDimension{
				Descriptor: "descriptor",
				Value:      123.0,
			},
		}
	}

	fields := []MarshalField{
		{Name: "ts"},
		{Name: "dimension.val", Alias: "val"},
	}

	for i := 0; i < b.N; i++ {
		Marshal("frame", fields, value)
	}
}

func BenchmarkEmbeddedStruct_1000(b *testing.B) {
	type embeddedDimension struct {
		Descriptor string
		Value      float64 `frame:"val"`
	}

	type tsd struct {
		Timestamp time.Time         `frame:"ts"`
		Dimension embeddedDimension `frame:"dimension"`
	}

	now := time.Now()
	value := make([]tsd, 1000)

	for i := range value {
		value[i] = tsd{
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Dimension: embeddedDimension{
				Descriptor: "descriptor",
				Value:      123.0,
			},
		}
	}

	fields := []MarshalField{
		{Name: "ts"},
		{Name: "dimension.val", Alias: "val"},
	}

	for i := 0; i < b.N; i++ {
		Marshal("frame", fields, value)
	}
}
