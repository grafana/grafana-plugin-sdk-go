package entity

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimpleEntities(t *testing.T) {
	kinds, err := NewKindRegistry(
		NewPlainTextKind(KindInfo{
			ID:         "text",
			FileSuffix: ".txt",
		}), NewPlainTextKind(KindInfo{
			ID:         "readme",
			FileSuffix: "README.md",
		}), NewGenericKind(KindInfo{
			ID:          "x",
			Description: "example",
			FileSuffix:  "x.json",
		}))
	require.NoError(t, err)
	require.NotNil(t, kinds)

	kind := kinds.Get("x")
	require.NoError(t, err)
	require.Equal(t, "example", kind.Info().Description)

	payload, err := ioutil.ReadFile("testdata/generic.x.json")
	require.NoError(t, err)

	rsp := kind.Validate(payload, false)
	require.True(t, rsp.Valid)

	ggg := &GenericEntity{
		Body: map[string]interface{}{
			"hello":  "world",
			"number": 123,
		},
	}
	ggg.Kind = "x"
	ggg.Name = "Test"
	ggg.Description = "some description here"

	out, err := kind.Write(ggg)
	require.NoError(t, err)
	require.Equal(t, string(payload), string(out))

	fmt.Printf("HELLO: %+v\n", string(rsp.Result))

	//t.FailNow()
}
