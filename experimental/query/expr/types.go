package expr

import (
	"embed"
	"encoding/json"
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

var _ query.TypedQueryReader[ExpressionQuery] = (*QueyHandler)(nil)

type QueyHandler struct{}

//go:embed types.json
var f embed.FS

func (*QueyHandler) QueryTypeDefinitionsJSON() (json.RawMessage, error) {
	return f.ReadFile("types.json")
}

// ReadQuery implements query.TypedQueryHandler.
func (*QueyHandler) ReadQuery(
	// Properties that have been parsed off the same node
	common query.CommonQueryProperties,
	// An iterator with context for the full node (include common values)
	iter *jsoniter.Iterator,
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
