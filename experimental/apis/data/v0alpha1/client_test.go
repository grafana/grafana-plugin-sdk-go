package v0alpha1_test

import (
	"errors"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/apis/data/v0alpha1"
	"github.com/stretchr/testify/require"
)

func TestNewErrorQDR(t *testing.T) {
	q1 := v0alpha1.DataQuery{}
	q1.RefID = "A"

	q2 := v0alpha1.DataQuery{}
	q2.RefID = "B"

	err := errors.New("test error")

	t.Run("NewErrorQDR works with zero refIds", func(t *testing.T) {
		req := v0alpha1.QueryDataRequest{
			Queries: []v0alpha1.DataQuery{},
		}

		qdr := v0alpha1.NewErrorQDR(req, err)
		require.Empty(t, qdr.Responses)
	})

	t.Run("NewErrorQDR works with single refId", func(t *testing.T) {
		req := v0alpha1.QueryDataRequest{
			Queries: []v0alpha1.DataQuery{q1},
		}

		qdr := v0alpha1.NewErrorQDR(req, err)
		require.Len(t, qdr.Responses, 1)

		require.ErrorIs(t, qdr.Responses["A"].Error, err)
		require.Equal(t, qdr.Responses["A"].Status, backend.StatusBadRequest)
	})

	t.Run("NewErrorQDR works with multiple refIds", func(t *testing.T) {
		req := v0alpha1.QueryDataRequest{
			Queries: []v0alpha1.DataQuery{q1, q2},
		}

		qdr := v0alpha1.NewErrorQDR(req, err)
		require.Len(t, qdr.Responses, 2)

		require.ErrorIs(t, qdr.Responses["A"].Error, err)
		require.Equal(t, qdr.Responses["A"].Status, backend.StatusBadRequest)

		require.ErrorIs(t, qdr.Responses["B"].Error, err)
		require.Equal(t, qdr.Responses["B"].Status, backend.StatusBadRequest)
	})
}
