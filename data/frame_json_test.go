package data_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrameJSON(t *testing.T) {

	f := goldenDF()
	//f2 := (*data.FrameJSON)(f)

	b, err := data.FrameToJSON(f, true, true) // json.Marshal(f2)
	require.NoError(t, err)

	str := string(b)
	fmt.Printf(">>> %s\n", str)

	assert.Equal(t, "", str)
}
