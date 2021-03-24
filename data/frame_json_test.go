package data_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoldenFrameJSON makes sure that the JSON produced from arrow and dataframes match
func TestGoldenFrameJSON(t *testing.T) {
	f := goldenDF()
	a, err := f.MarshalArrow()
	require.NoError(t, err)

	b, err := data.FrameToJSON(f, data.WithSchmaAndData) // json.Marshal(f2)
	require.NoError(t, err)
	strF := string(b)

	b, err = data.ArrowBufferToJSON(a, true, true)
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
		_, err := data.FrameToJSON(f, data.WithSchmaAndData)
		if err != nil {
			b.Fatal(err)
		}
	}
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
}`

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
			HasSpecialEntities bool
		}{
			Type:               strings.Title(tstr),
			Typex:              strings.Title(typex),
			HasSpecialEntities: hasSpecialEntities,
		}
		tmpl, err := template.New("").Parse(code)
		require.NoError(t, err)
		err = tmpl.Execute(os.Stdout, tmplData)
		require.NoError(t, err)
		fmt.Printf("\n")
	}

	assert.Equal(t, 1, 2)
}
