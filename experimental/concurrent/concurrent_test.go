package concurrent

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func Test_QueryData(t *testing.T) {
	t.Run("executes all queries concurrently", func(t *testing.T) {
		secondExecuted := make(chan bool, 1)
		fn := func(_ context.Context, query Query) (res backend.DataResponse) {
			if query.Index == 0 {
				// Blocks until the second query is executed
				<-secondExecuted
			}
			if query.Index == 1 {
				secondExecuted <- true
			}
			return backend.DataResponse{}
		}

		req := &backend.QueryDataRequest{}
		req.Queries = append(req.Queries, backend.DataQuery{RefID: "A"})
		req.Queries = append(req.Queries, backend.DataQuery{RefID: "B"})
		// Limit to 2 queries
		_, err := QueryData(context.Background(), req, fn, 2)
		require.NoError(t, err)
	})

	t.Run("executes all queries concurrently with limit", func(t *testing.T) {
		secondExecutedChannel := make(chan bool, 1)
		secondExecuted := false
		fn := func(_ context.Context, query Query) (res backend.DataResponse) {
			if query.Index == 0 {
				// Blocks until the second query is executed
				secondExecuted = <-secondExecutedChannel
			}
			if query.Index == 1 {
				secondExecutedChannel <- true
				close(secondExecutedChannel)
			}
			if query.Index == 2 {
				// Should not be executed until the second query is done
				require.True(t, secondExecuted)
			}

			return backend.DataResponse{}
		}

		req := &backend.QueryDataRequest{}
		req.Queries = append(req.Queries, backend.DataQuery{RefID: "A"})
		req.Queries = append(req.Queries, backend.DataQuery{RefID: "B"})
		// Limit to 2 queries
		_, err := QueryData(context.Background(), req, fn, 2)
		require.NoError(t, err)
	})

	t.Run("handles panics", func(t *testing.T) {
		fn := func(_ context.Context, query Query) (res backend.DataResponse) {
			if query.Index == 0 {
				panic("panic")
			}
			return backend.DataResponse{}
		}

		req := &backend.QueryDataRequest{}
		req.Queries = append(req.Queries, backend.DataQuery{RefID: "A"})
		req.Queries = append(req.Queries, backend.DataQuery{RefID: "B"})
		// Limit to 2 queries
		res, err := QueryData(context.Background(), req, fn, 2)
		require.NoError(t, err)
		require.Len(t, res.Responses, 2)
		require.Equal(t, "panic", res.Responses["A"].Error.Error())
		require.Equal(t, nil, res.Responses["B"].Error)
	})
}
