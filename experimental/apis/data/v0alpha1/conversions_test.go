package v0alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConversionsDefaults(t *testing.T) {
	res, err := toBackendDataQuery(DataQuery{}, nil)

	require.NoError(t, err)

	// we used to default the refId to "A",
	// we do not do that anymore,
	// we verify the new behavior
	require.Equal(t, "", res.RefID)

	require.Equal(t, int64(100), res.MaxDataPoints)
	require.Equal(t, time.Second, res.Interval)
}
