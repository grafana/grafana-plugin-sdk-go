package sqlutil_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
)

var errorQueryCompleted = errors.New("query completed")

type testConnection struct {
	QueryWait     time.Duration
	QueryRunCount int
}

func (t *testConnection) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	t.QueryRunCount++

	done := make(chan bool)
	go func() {
		time.Sleep(t.QueryWait)
		done <- true
	}()

	select {
	case <-ctx.Done():
		return nil, context.Canceled
	case <-done:
		return nil, errorQueryCompleted
	}
}

func TestQuery_Timeout(t *testing.T) {
	t.Run("it should return context.Canceled if the query timeout is exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()

		conn := &testConnection{
			QueryWait: time.Second * 5,
		}

		_, err := sqlutil.QueryDB(ctx, conn, []sqlutil.Converter{}, nil, &sqlutil.Query{})

		if !errors.Is(err, context.Canceled) {
			t.Fatal("expected error to be context.Canceled, received", err)
		}

		if conn.QueryRunCount != 1 {
			t.Fatal("expected the querycontext function to run only once, but ran", conn.QueryRunCount, "times")
		}
	})

	t.Run("it should run to completion and not return a query timeout error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()

		conn := &testConnection{
			QueryWait: time.Second,
		}

		_, err := sqlutil.QueryDB(ctx, conn, []sqlutil.Converter{}, nil, &sqlutil.Query{})

		if !errors.Is(err, sqlutil.ErrorQuery) {
			t.Fatal("expected function to complete, received error: ", err)
		}
	})
}
