package example

import (
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/resource"
)

// Supported expression types
// +enum
type QueryType string

const (
	// Math query type
	QueryTypeMath QueryType = "math"

	// Reduce query type
	QueryTypeReduce QueryType = "reduce"

	// Reduce query type
	QueryTypeResample QueryType = "resample"
)

type ExpressionQuery interface {
	ExpressionQueryType() QueryType
	Variables() []string
}

var _ resource.TypedQueryParser[ExpressionQuery] = (*QueyHandler)(nil)

type QueyHandler struct{}

func (*QueyHandler) ParseQuery(
	common resource.CommonQueryProperties,
	iter *jsoniter.Iterator,
	_ time.Time,
) (ExpressionQuery, error) {
	qt := QueryType(common.QueryType)
	switch qt {
	case QueryTypeMath:
		return readMathQuery(iter)

	case QueryTypeReduce:
		q := &ReduceQuery{}
		err := iter.ReadVal(q)
		return q, err

	case QueryTypeResample:
		return nil, nil
	}
	return nil, fmt.Errorf("unknown query type")
}
