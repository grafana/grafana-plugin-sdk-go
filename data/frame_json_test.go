package data_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrameJSON(t *testing.T) {

	f := goldenDF()
	a, err := f.MarshalArrow()
	require.NoError(t, err)

	b, err := data.FrameToJSON(f, true, true) // json.Marshal(f2)
	require.NoError(t, err)

	str := string(b)
	fmt.Printf(">>> %s\n", str)

	b, err = data.ArrowBufferToJSON(a)
	require.NoError(t, err)

	str2 := string(b)
	fmt.Printf(">>> %s\n", str2)

	assert.JSONEq(t, str, str2)
}

func TestGENERATE(t *testing.T) {
	t.Skip()

	types := []string{
		"uint8", "uint16", "uint32", "uint64", "int8", "int16", "int32", "int64", "float32", "float64", "string", "bool",
	}

	code := `
func writeArrowData{{TYPE}}(stream *jsoniter.Stream, col array.Interface) []*fieldEntityLookup {
	var entities []*fieldEntityLookup
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
			txt := fmt.Sprintf("%v", v)
			if entities == nil {
				entities = make([]*fieldEntityLookup, count)
			}
			if entities[i] == nil {
				entities[i] = &fieldEntityLookup{}
			}
			entities[i].add(txt, i)
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
