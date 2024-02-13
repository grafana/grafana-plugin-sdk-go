package expr

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/query"
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

var _ query.TypedQueryHandler[ExpressionQuery] = (*QueyHandler)(nil)

type QueyHandler struct{}

func (*QueyHandler) QueryTypeField() string {
	return "queryType"
}

// QueryTypes implements query.TypedQueryHandler.
func (*QueyHandler) QueryTypeDefinitions() []query.QueryTypeDefinitionSpec {
	return []query.QueryTypeDefinitionSpec{}
}

// ReadQuery implements query.TypedQueryHandler.
func (*QueyHandler) ReadQuery(
	// The query type split by version (when multiple exist)
	queryType string, version string,
	// Properties that have been parsed off the same node
	common query.CommonQueryProperties,
	// An iterator with context for the full node (include common values)
	iter *jsoniter.Iterator,
) (ExpressionQuery, error) {
	qt := QueryType(queryType)
	switch qt {
	case QueryTypeMath:
		return readMathQuery(version, iter)

	case QueryTypeReduce:
		q := &ReduceQuery{}
		err := iter.ReadVal(q)
		return q, err

	case QueryTypeResample:
		return nil, nil
	}
	return nil, fmt.Errorf("unknown query type")
}
