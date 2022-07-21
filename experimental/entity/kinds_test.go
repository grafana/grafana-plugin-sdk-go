package entity

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimpleEntities(t *testing.T) {
	kinds, err := NewKindRegistry(
		NewPlainTextKind(&KindInfo{
			ID:         "text",
			PathSuffix: ".txt",
		}), NewPlainTextKind(&KindInfo{
			ID:         "readme",
			PathSuffix: "README.md",
		}), NewGenericKind(&KindInfo{
			ID:          "x",
			Description: "example",
			PathSuffix:  ".x.json",
		}), NewGenericKind(&KindInfo{
			ID:          "yx",
			Description: "longer extension",
			PathSuffix:  ".y.x.json",
		}))
	require.NoError(t, err)
	require.NotNil(t, kinds)

	kind := kinds.Get("x")
	require.NoError(t, err)
	require.Equal(t, "example", kind.Info().Description)

	payload, err := ioutil.ReadFile("testdata/generic.x.json")
	require.NoError(t, err)

	rsp := kind.Normalize(payload, false)
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

	kind = kinds.GetFromSuffix("hello/world.x.json")
	require.Equal(t, "x", kind.Info().ID)

	kind = kinds.GetFromSuffix("hello/world.y.x.json")
	require.Equal(t, "yx", kind.Info().ID)

	// t.FailNow()
}
