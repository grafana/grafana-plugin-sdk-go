package standard

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/entity"
	"github.com/stretchr/testify/require"
)

func TestSimpleEntities(t *testing.T) {
	kinds, err := entity.NewKindRegistry(
		NewPlainTextKind(entity.KindInfo{
			ID:         "text",
			FileSuffix: ".txt",
		}), NewPlainTextKind(entity.KindInfo{
			ID:         "readme",
			FileSuffix: "README.md",
		}), NewGenericKind(entity.KindInfo{
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

	// ggg := GenericEntity{
	// 	Envelope: entity.Envelope{
	// 		Kind: "xxx",
	// 		Props: &entity.EntityProperties{
	// 			Name:        "Some name here",
	// 			Description: "description",
	// 			Labels: map[string]string{
	// 				"s0": "lookup0",
	// 				"s1": "lookup1",
	// 			},
	// 		},
	// 	},
	// 	Body: map[string]interface{}{
	// 		"hello":  "world",
	// 		"number": 123,
	// 	},
	// }

	fmt.Printf("HELLO: %+v\n", string(rsp.Result))

	t.FailNow()
}
