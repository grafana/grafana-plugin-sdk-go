package concurrent

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func Test_QueryData(t *testing.T) {
	t.Run("executes all queries concurrently", func(t *testing.T) {
		secondExecuted := make(chan bool, 1)
		fn := func(_ context.Context, query Query) (res backend.DataResponse) {
			if query.DataQuery.RefID == "A" {
				// Blocks until the second query is executed
				<-secondExecuted
			}
			if query.DataQuery.RefID == "B" {
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
		mux := sync.Mutex{}
		counter := 0
		fn := func(_ context.Context, query Query) (res backend.DataResponse) {
			mux.Lock()
			defer mux.Unlock()
			// Checks that the query with RefID "C" is executed after a previous query is done
			if query.DataQuery.RefID == "C" {
				if counter == 0 {
					return backend.DataResponse{
						Error: errors.New("query executed without respecting the limit"),
					}
				}
			}
			counter++
			return backend.DataResponse{}
		}

		req := &backend.QueryDataRequest{}
		req.Queries = append(req.Queries, backend.DataQuery{RefID: "A"})
		req.Queries = append(req.Queries, backend.DataQuery{RefID: "B"})
		req.Queries = append(req.Queries, backend.DataQuery{RefID: "C"})
		// Limit to 2 queries concurrently
		res, err := QueryData(context.Background(), req, fn, 2)
		require.NoError(t, err)
		require.Len(t, res.Responses, 3)
		require.Equal(t, nil, res.Responses["C"].Error)
	})

	t.Run("handles panics", func(t *testing.T) {
		fn := func(_ context.Context, query Query) (res backend.DataResponse) {
			if query.DataQuery.RefID == "A" {
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
