package data_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoldenFrameJSON(t *testing.T) {
	f := goldenDF()
	a, err := f.MarshalArrow()
	require.NoError(t, err)

	b, err := data.FrameToJSON(f, true, true) // json.Marshal(f2)
	require.NoError(t, err)
	strF := string(b)

	b, err = data.ArrowBufferToJSON(a)
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
		ioutil.WriteFile(goldenFile, b, 0600)
		assert.FailNow(t, "wrote golden file")
	}

	b, err = ioutil.ReadFile(goldenFile)
	require.NoError(t, err)

	strG := string(b)
	assert.JSONEq(t, strF, strG, "saved json must match produced json")

	//assert.Equal(t, 1, 2)
}

func TestGENERATE(t *testing.T) {
	t.Skip()

	types := []string{
		"uint8", "uint16", "uint32", "uint64", "int8", "int16", "int32", "int64", "float32", "float64", "string", "bool",
	}

	code := `
func writeArrowData{{TYPE}}(stream *jsoniter.Stream, col array.Interface) *fieldEntityLookup {
	var entities *fieldEntityLookup
	count := col.Len()

	v := array.New{{TYPEX}}Data(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		stream.Write{{TYPE}}(v.Value(i))
		if stream.Error != nil { // NaN +Inf/-Inf
			txt := fmt.Sprintf("%v", v.Value(i))
			if entities == nil {
				entities = &fieldEntityLookup{}
			}
			entities.add(txt, i)
			stream.Error = nil
			stream.WriteNil()
		}
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

		ttt := strings.Title(tstr)
		str := strings.ReplaceAll(code, "{{TYPE}}", ttt)
		if tstr == "bool" {
			ttt = "Boolean"
		}
		str = strings.ReplaceAll(str, "{{TYPEX}}", ttt)

		fmt.Printf("%s\n\n\n", str)
	}

	assert.Equal(t, 1, 2)
}
