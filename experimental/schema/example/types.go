package example

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/schema"
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

var _ schema.TypedQueryParser[ExpressionQuery] = (*QueyHandler)(nil)

type QueyHandler struct{}

//go:embed query.types.json
var f embed.FS

func (*QueyHandler) QueryTypeDefinitionsJSON() (json.RawMessage, error) {
	return f.ReadFile("query.types.json")
}

// ReadQuery implements query.TypedQueryHandler.
func (*QueyHandler) ParseQuery(
	// Properties that have been parsed off the same node
	common schema.CommonQueryProperties,
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
