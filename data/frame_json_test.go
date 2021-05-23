package data_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"text/template"

	jsoniter "github.com/json-iterator/go"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoldenFrameJSON makes sure that the JSON produced from arrow and dataframes match
func TestGoldenFrameJSON(t *testing.T) {
	f := goldenDF()
	a, err := f.MarshalArrow()
	require.NoError(t, err)

	fjs, err := data.FrameToJSON(f, data.SchemaAndData) // json.Marshal(f2)
	require.NoError(t, err)
	b := fjs.Bytes(data.SchemaAndData)
	strF := string(b)

	b, err = data.ArrowBufferToJSON(a, data.SchemaAndData)
	require.NoError(t, err)
	strA := string(b)

	fmt.Println(`{ "arrow": `)
	fmt.Println(strA)
	fmt.Println(`, "slice": `)
	fmt.Println(strF)
	fmt.Println(`}`)

	assert.JSONEq(t, strF, strA, "arrow and frames should produce the same json")

	goldenFile := filepath.Join("testdata", "all_types.golden.json")
	if _, err := os.Stat(goldenFile); os.IsNotExist(err) {
		_ = ioutil.WriteFile(goldenFile, b, 0600)
		assert.FailNow(t, "wrote golden file")
	}

	b, err = ioutil.ReadFile(goldenFile)
	require.NoError(t, err)

	strG := string(b)
	assert.JSONEq(t, strF, strG, "saved json must match produced json")

	// Read the frame from json
	out := &data.Frame{}
	err = json.Unmarshal(b, out)
	require.NoError(t, err)

	if diff := cmp.Diff(f, out, data.FrameTestCompareOptions()...); diff != "" {
		t.Errorf("Result mismatch (-want +got):\n%s", diff)
	}
}

type simpleTestObj struct {
	Name   string          `json:"name,omitempty"`
	FType  data.FieldType  `json:"type,omitempty"`
	FType2 *data.FieldType `json:"typePtr,omitempty"`
}

// TestFieldTypeToJSON makes sure field type will read/write to json
func TestFieldTypeToJSON(t *testing.T) {
	v := simpleTestObj{
		Name: "hello",
	}

	b, err := json.Marshal(v)
	require.NoError(t, err)
	assert.Equal(t, data.FieldTypeUnknown, v.FType)

	assert.Equal(t, `{"name":"hello"}`, string(b))

	ft := data.FieldTypeInt8

	v.FType = data.FieldTypeFloat64
	v.FType2 = &ft
	v.Name = ""
	b, err = json.Marshal(v)
	require.NoError(t, err)
	assert.Equal(t, `{"type":"float64","typePtr":"int8"}`, string(b))

	err = json.Unmarshal([]byte(`{"type":"int8","typePtr":"time"}`), &v)
	require.NoError(t, err)
	assert.Equal(t, data.FieldTypeInt8, v.FType)
	assert.Equal(t, data.FieldTypeTime, *v.FType2)
}

func BenchmarkFrameToJSON(b *testing.B) {
	f := goldenDF()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := data.FrameToJSON(f, data.SchemaAndData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFrameMarshalJSONStd(b *testing.B) {
	f := goldenDF()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(f)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFrameMarshalJSONIter(b *testing.B) {
	f := goldenDF()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := jsoniter.Marshal(f)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestFrameMarshalJSONConcurrent(t *testing.T) {
	f := goldenDF()
	initialJSON, err := json.Marshal(f)
	require.NoError(t, err)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				jsonData, err := json.Marshal(f)
				require.NoError(t, err)
				require.JSONEq(t, string(initialJSON), string(jsonData))
			}
		}()
	}
	wg.Wait()
}

type testWrapper struct {
	Data data.FrameJSON
}

func TestFrame_UnmarshalJSON_SchemaOnly(t *testing.T) {
	f := data.NewFrame("test", data.NewField("test", nil, []int64{1}))
	d, err := data.FrameToJSON(f, data.OnlySchema)
	require.NoError(t, err)
	_, err = json.Marshal(testWrapper{Data: d})
	require.NoError(t, err)
	var newFrame data.Frame
	err = json.Unmarshal(d.Bytes(data.SchemaAndData), &newFrame)
	require.NoError(t, err)
	require.Equal(t, 0, newFrame.Fields[0].Len())
}

func TestFrameMarshalJSON_DataOnly(t *testing.T) {
	f := goldenDF()
	d, err := data.FrameToJSON(f, data.OnlyData)
	require.NoError(t, err)
	_, err = json.Marshal(testWrapper{Data: d})
	require.NoError(t, err)
	var newFrame data.Frame
	err = json.Unmarshal(d.Bytes(data.SchemaAndData), &newFrame)
	require.Error(t, err)
}

func TestFrame_UnmarshalJSON_SchemaAndData_WrongOrder(t *testing.T) {
	// At this moment we can only unmarshal frames with "schema" key first.
	d := []byte(`{"data":{"values":[[]]}, "schema":{"name":"test","fields":[{"name":"test","type":"number","typeInfo":{"frame":"int64"}}]}}`)
	var newFrame data.Frame
	err := json.Unmarshal(d, &newFrame)
	require.Error(t, err)
}

func TestFrame_UnmarshalJSON_DataOnly(t *testing.T) {
	f := data.NewFrame("test", data.NewField("test", nil, []int64{}))
	d, err := data.FrameToJSON(f, data.OnlyData)

	require.NoError(t, err)
	var newFrame data.Frame
	err = json.Unmarshal(d.Bytes(data.SchemaAndData), &newFrame)
	require.Error(t, err)
}

// This function will write code to the console that should be copy/pasted into frame_json.gen.go
// when changes are required. Typically this function will always be skipped.
func TestGenerateGenericArrowCode(t *testing.T) {
	t.Skip()

	types := []string{
		"uint8", "uint16", "uint32", "uint64",
		"int8", "int16", "int32", "int64",
		"float32", "float64", "string", "bool",
	}

	code := `
func writeArrowData{{.Type}}(stream *jsoniter.Stream, col array.Interface) *fieldEntityLookup {
	var entities *fieldEntityLookup
	count := col.Len()

	v := array.New{{.Typex}}Data(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
{{- if .HasSpecialEntities }}
		val := v.Value(i)
		f64 := float64(val)
		if entityType, found := isSpecialEntity(f64); found {
			if entities == nil {
				entities = &fieldEntityLookup{}
			}
			entities.add(entityType, i)
			stream.WriteNil()
		} else {
			stream.Write{{.Type}}(val)
		}
{{ else }}
		stream.Write{{.Type}}(v.Value(i)){{ end }}
	}
	stream.WriteArrayEnd()
	return entities
}

func read{{.Type}}VectorJSON(iter *jsoniter.Iterator, size int) (*{{.Typen}}Vector, error) {
	arr := new{{.Type}}Vector(size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("read{{.Type}}VectorJSON", "expected array")
			return nil, iter.Error
		}

		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.Read{{.Type}}()
			arr.Set(i, v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("read", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}


func readNullable{{.Type}}VectorJSON(iter *jsoniter.Iterator, size int) (*nullable{{.Type}}Vector, error) {
	arr := newNullable{{.Type}}Vector(size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readNullable{{.Type}}VectorJSON", "expected array")
			return nil, iter.Error
		}
		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.Read{{.Type}}()
			arr.Set(i, &v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("readNullable{{.Type}}VectorJSON", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

`

	// switch col.DataType().ID() {
	// 	// case arrow.STRING:
	// 	// 	ent := writeArrowSTRING(stream, col)
	for _, tstr := range types {
		tname := strings.Title(tstr)
		tuppr := strings.ToUpper(tstr)

		fmt.Printf("    case arrow." + tuppr + ":\n\t\tent = writeArrowData" + tname + "(stream, col)\n")
	}

	for _, tstr := range types {
		typex := tstr
		if tstr == "bool" {
			typex = "Boolean"
		}
		hasSpecialEntities := tstr == "float32" || tstr == "float64"
		tmplData := struct {
			Type               string
			Typex              string
			Typen              string
			HasSpecialEntities bool
		}{
			Type:               strings.Title(tstr),
			Typex:              strings.Title(typex),
			Typen:              tstr,
			HasSpecialEntities: hasSpecialEntities,
		}
		tmpl, err := template.New("").Parse(code)
		require.NoError(t, err)
		err = tmpl.Execute(os.Stdout, tmplData)
		require.NoError(t, err)
		fmt.Printf("\n")
	}

	for _, tstr := range types {
		tname := strings.Title(tstr)
		fmt.Printf("    case FieldType" + tname + ": return read" + tname + "VectorJSON(iter, size)\n")
		fmt.Printf("    case FieldTypeNullable" + tname + ": return readNullable" + tname + "VectorJSON(iter, size)\n")
	}

	assert.Equal(t, 1, 2)
}
